package system

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"time"
)

// ResourceChecker provides system resource checking capabilities
type ResourceChecker struct {
	logger Logger
}

// Logger interface for resource checker
type Logger interface {
	Debug(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
}

// NewResourceChecker creates a new resource checker
func NewResourceChecker(logger Logger) *ResourceChecker {
	return &ResourceChecker{
		logger: logger,
	}
}

// DiskSpaceInfo contains disk space information
type DiskSpaceInfo struct {
	Path      string `json:"path"`
	Total     uint64 `json:"total"`     // Total space in bytes
	Free      uint64 `json:"free"`      // Free space in bytes
	Available uint64 `json:"available"` // Available space in bytes
	Used      uint64 `json:"used"`      // Used space in bytes
	UsedPct   float64 `json:"used_pct"` // Used percentage
}

// MemoryInfo contains memory information
type MemoryInfo struct {
	Total     uint64  `json:"total"`     // Total memory in bytes
	Available uint64  `json:"available"` // Available memory in bytes
	Used      uint64  `json:"used"`      // Used memory in bytes
	UsedPct   float64 `json:"used_pct"`  // Used percentage
}

// SystemResourceInfo contains comprehensive system resource information
type SystemResourceInfo struct {
	Timestamp    time.Time       `json:"timestamp"`
	OS           string          `json:"os"`
	Architecture string          `json:"architecture"`
	CPUCount     int             `json:"cpu_count"`
	Memory       *MemoryInfo     `json:"memory"`
	DiskSpaces   []DiskSpaceInfo `json:"disk_spaces"`
	WorkingDir   string          `json:"working_dir"`
	TempDir      string          `json:"temp_dir"`
	HomeDir      string          `json:"home_dir"`
}

// ResourceRequirement defines resource requirements for operations
type ResourceRequirement struct {
	MinDiskSpace uint64 // Minimum disk space in bytes
	MinMemory    uint64 // Minimum memory in bytes
	RequiredDirs []string // Required directories
	RequiredPerms []PermissionCheck // Required permissions
}

// PermissionCheck defines a permission check
type PermissionCheck struct {
	Path        string `json:"path"`
	RequireRead bool   `json:"require_read"`
	RequireWrite bool  `json:"require_write"`
	RequireExec bool   `json:"require_exec"`
}

// ResourceCheckResult contains the result of a resource check
type ResourceCheckResult struct {
	Passed      bool                    `json:"passed"`
	Warnings    []string                `json:"warnings"`
	Errors      []string                `json:"errors"`
	Suggestions []string                `json:"suggestions"`
	Details     map[string]interface{}  `json:"details"`
	SystemInfo  *SystemResourceInfo     `json:"system_info"`
}

// CheckDiskSpace checks disk space for a given path
func (rc *ResourceChecker) CheckDiskSpace(path string) (*DiskSpaceInfo, error) {
	// Get absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}
	
	// Get disk usage statistics
	var stat syscall.Statfs_t
	if err := syscall.Statfs(absPath, &stat); err != nil {
		return nil, fmt.Errorf("failed to get disk statistics: %w", err)
	}
	
	// Calculate disk space information
	total := stat.Blocks * uint64(stat.Bsize)
	free := stat.Bfree * uint64(stat.Bsize)
	available := stat.Bavail * uint64(stat.Bsize)
	used := total - free
	usedPct := float64(used) / float64(total) * 100
	
	info := &DiskSpaceInfo{
		Path:      absPath,
		Total:     total,
		Free:      free,
		Available: available,
		Used:      used,
		UsedPct:   usedPct,
	}
	
	if rc.logger != nil {
		rc.logger.Debug("Disk space for %s: %.2f%% used (%.2f GB / %.2f GB)", 
			absPath, usedPct, float64(used)/(1024*1024*1024), float64(total)/(1024*1024*1024))
	}
	
	return info, nil
}

// CheckMemory checks system memory usage
func (rc *ResourceChecker) CheckMemory() (*MemoryInfo, error) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	// Note: This is a simplified memory check using Go runtime stats
	// For more accurate system memory info, we would need platform-specific code
	info := &MemoryInfo{
		Total:     m.Sys,
		Used:      m.Alloc,
		Available: m.Sys - m.Alloc,
		UsedPct:   float64(m.Alloc) / float64(m.Sys) * 100,
	}
	
	if rc.logger != nil {
		rc.logger.Debug("Memory usage: %.2f%% (%.2f MB / %.2f MB)", 
			info.UsedPct, float64(info.Used)/(1024*1024), float64(info.Total)/(1024*1024))
	}
	
	return info, nil
}

// CheckPermissions checks file/directory permissions
func (rc *ResourceChecker) CheckPermissions(checks []PermissionCheck) []string {
	var issues []string
	
	for _, check := range checks {
		if rc.logger != nil {
			rc.logger.Debug("Checking permissions for: %s", check.Path)
		}
		
		// Check if path exists
		info, err := os.Stat(check.Path)
		if err != nil {
			if os.IsNotExist(err) {
				issues = append(issues, fmt.Sprintf("Path does not exist: %s", check.Path))
			} else {
				issues = append(issues, fmt.Sprintf("Cannot access path %s: %v", check.Path, err))
			}
			continue
		}
		
		mode := info.Mode()
		
		// Check read permission
		if check.RequireRead {
			if err := rc.checkReadPermission(check.Path); err != nil {
				issues = append(issues, fmt.Sprintf("No read permission for %s: %v", check.Path, err))
			}
		}
		
		// Check write permission
		if check.RequireWrite {
			if err := rc.checkWritePermission(check.Path, info.IsDir()); err != nil {
				issues = append(issues, fmt.Sprintf("No write permission for %s: %v", check.Path, err))
			}
		}
		
		// Check execute permission
		if check.RequireExec && !info.IsDir() {
			if mode&0111 == 0 {
				issues = append(issues, fmt.Sprintf("No execute permission for %s", check.Path))
			}
		}
	}
	
	return issues
}

// checkReadPermission checks if a path is readable
func (rc *ResourceChecker) checkReadPermission(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	file.Close()
	return nil
}

// checkWritePermission checks if a path is writable
func (rc *ResourceChecker) checkWritePermission(path string, isDir bool) error {
	if isDir {
		// For directories, try to create a temporary file
		tempFile := filepath.Join(path, ".apkhub_write_test")
		file, err := os.Create(tempFile)
		if err != nil {
			return err
		}
		file.Close()
		os.Remove(tempFile)
		return nil
	} else {
		// For files, try to open in write mode
		file, err := os.OpenFile(path, os.O_WRONLY, 0)
		if err != nil {
			return err
		}
		file.Close()
		return nil
	}
}

// GetSystemResourceInfo gathers comprehensive system resource information
func (rc *ResourceChecker) GetSystemResourceInfo(paths []string) (*SystemResourceInfo, error) {
	info := &SystemResourceInfo{
		Timestamp:    time.Now(),
		OS:           runtime.GOOS,
		Architecture: runtime.GOARCH,
		CPUCount:     runtime.NumCPU(),
		DiskSpaces:   []DiskSpaceInfo{},
	}
	
	// Get memory info
	if memInfo, err := rc.CheckMemory(); err == nil {
		info.Memory = memInfo
	}
	
	// Get disk space info for specified paths
	checkedPaths := make(map[string]bool)
	for _, path := range paths {
		absPath, err := filepath.Abs(path)
		if err != nil {
			continue
		}
		
		// Avoid checking the same path multiple times
		if checkedPaths[absPath] {
			continue
		}
		checkedPaths[absPath] = true
		
		if diskInfo, err := rc.CheckDiskSpace(absPath); err == nil {
			info.DiskSpaces = append(info.DiskSpaces, *diskInfo)
		}
	}
	
	// Get directory information
	if wd, err := os.Getwd(); err == nil {
		info.WorkingDir = wd
	}
	
	info.TempDir = os.TempDir()
	
	if homeDir, err := os.UserHomeDir(); err == nil {
		info.HomeDir = homeDir
	}
	
	return info, nil
}

// CheckResourceRequirements checks if system meets the specified requirements
func (rc *ResourceChecker) CheckResourceRequirements(req ResourceRequirement, paths []string) *ResourceCheckResult {
	result := &ResourceCheckResult{
		Passed:      true,
		Warnings:    []string{},
		Errors:      []string{},
		Suggestions: []string{},
		Details:     make(map[string]interface{}),
	}
	
	// Get system info
	systemInfo, err := rc.GetSystemResourceInfo(paths)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to get system info: %v", err))
		result.Passed = false
	} else {
		result.SystemInfo = systemInfo
	}
	
	// Check disk space requirements
	if req.MinDiskSpace > 0 {
		rc.checkDiskSpaceRequirement(req.MinDiskSpace, systemInfo, result)
	}
	
	// Check memory requirements
	if req.MinMemory > 0 {
		rc.checkMemoryRequirement(req.MinMemory, systemInfo, result)
	}
	
	// Check required directories
	if len(req.RequiredDirs) > 0 {
		rc.checkRequiredDirectories(req.RequiredDirs, result)
	}
	
	// Check permissions
	if len(req.RequiredPerms) > 0 {
		permIssues := rc.CheckPermissions(req.RequiredPerms)
		if len(permIssues) > 0 {
			result.Errors = append(result.Errors, permIssues...)
			result.Passed = false
			result.Suggestions = append(result.Suggestions, "Fix file/directory permissions")
		}
	}
	
	return result
}

// checkDiskSpaceRequirement checks disk space requirements
func (rc *ResourceChecker) checkDiskSpaceRequirement(minSpace uint64, systemInfo *SystemResourceInfo, result *ResourceCheckResult) {
	if systemInfo == nil || len(systemInfo.DiskSpaces) == 0 {
		result.Warnings = append(result.Warnings, "Could not check disk space")
		return
	}
	
	minSpaceMB := float64(minSpace) / (1024 * 1024)
	result.Details["min_disk_space_mb"] = minSpaceMB
	
	for _, diskInfo := range systemInfo.DiskSpaces {
		availableMB := float64(diskInfo.Available) / (1024 * 1024)
		result.Details[fmt.Sprintf("available_space_%s_mb", diskInfo.Path)] = availableMB
		
		if diskInfo.Available < minSpace {
			result.Errors = append(result.Errors, 
				fmt.Sprintf("Insufficient disk space on %s: %.2f MB available, %.2f MB required", 
					diskInfo.Path, availableMB, minSpaceMB))
			result.Passed = false
			result.Suggestions = append(result.Suggestions, 
				fmt.Sprintf("Free up at least %.2f MB of disk space on %s", 
					minSpaceMB-availableMB, diskInfo.Path))
		} else if diskInfo.Available < minSpace*2 {
			// Warn if less than 2x the required space
			result.Warnings = append(result.Warnings, 
				fmt.Sprintf("Low disk space on %s: %.2f MB available", diskInfo.Path, availableMB))
		}
	}
}

// checkMemoryRequirement checks memory requirements
func (rc *ResourceChecker) checkMemoryRequirement(minMemory uint64, systemInfo *SystemResourceInfo, result *ResourceCheckResult) {
	if systemInfo == nil || systemInfo.Memory == nil {
		result.Warnings = append(result.Warnings, "Could not check memory usage")
		return
	}
	
	minMemoryMB := float64(minMemory) / (1024 * 1024)
	availableMB := float64(systemInfo.Memory.Available) / (1024 * 1024)
	
	result.Details["min_memory_mb"] = minMemoryMB
	result.Details["available_memory_mb"] = availableMB
	
	if systemInfo.Memory.Available < minMemory {
		result.Errors = append(result.Errors, 
			fmt.Sprintf("Insufficient memory: %.2f MB available, %.2f MB required", 
				availableMB, minMemoryMB))
		result.Passed = false
		result.Suggestions = append(result.Suggestions, "Close other applications to free up memory")
	} else if systemInfo.Memory.Available < minMemory*2 {
		result.Warnings = append(result.Warnings, 
			fmt.Sprintf("Low memory: %.2f MB available", availableMB))
	}
}

// checkRequiredDirectories checks if required directories exist and are accessible
func (rc *ResourceChecker) checkRequiredDirectories(requiredDirs []string, result *ResourceCheckResult) {
	for _, dir := range requiredDirs {
		if info, err := os.Stat(dir); err != nil {
			if os.IsNotExist(err) {
				result.Errors = append(result.Errors, fmt.Sprintf("Required directory does not exist: %s", dir))
				result.Suggestions = append(result.Suggestions, fmt.Sprintf("Create directory: %s", dir))
			} else {
				result.Errors = append(result.Errors, fmt.Sprintf("Cannot access required directory %s: %v", dir, err))
				result.Suggestions = append(result.Suggestions, fmt.Sprintf("Check permissions for directory: %s", dir))
			}
			result.Passed = false
		} else if !info.IsDir() {
			result.Errors = append(result.Errors, fmt.Sprintf("Required path is not a directory: %s", dir))
			result.Passed = false
		}
	}
}

// FormatResourceInfo formats system resource information for display
func (rc *ResourceChecker) FormatResourceInfo(info *SystemResourceInfo) string {
	if info == nil {
		return "No system information available"
	}
	
	output := fmt.Sprintf("System Information (as of %s):\n", info.Timestamp.Format("2006-01-02 15:04:05"))
	output += fmt.Sprintf("  OS: %s\n", info.OS)
	output += fmt.Sprintf("  Architecture: %s\n", info.Architecture)
	output += fmt.Sprintf("  CPU Cores: %d\n", info.CPUCount)
	
	if info.Memory != nil {
		output += fmt.Sprintf("  Memory: %.2f MB used / %.2f MB total (%.1f%%)\n", 
			float64(info.Memory.Used)/(1024*1024), 
			float64(info.Memory.Total)/(1024*1024), 
			info.Memory.UsedPct)
	}
	
	if len(info.DiskSpaces) > 0 {
		output += "  Disk Usage:\n"
		for _, disk := range info.DiskSpaces {
			output += fmt.Sprintf("    %s: %.2f GB used / %.2f GB total (%.1f%%) - %.2f GB available\n", 
				disk.Path,
				float64(disk.Used)/(1024*1024*1024),
				float64(disk.Total)/(1024*1024*1024),
				disk.UsedPct,
				float64(disk.Available)/(1024*1024*1024))
		}
	}
	
	output += fmt.Sprintf("  Working Directory: %s\n", info.WorkingDir)
	output += fmt.Sprintf("  Temp Directory: %s\n", info.TempDir)
	output += fmt.Sprintf("  Home Directory: %s\n", info.HomeDir)
	
	return output
}
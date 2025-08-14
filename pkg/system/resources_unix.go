//go:build !windows
// +build !windows

package system

import (
	"fmt"
	"path/filepath"
	"syscall"
)

// getDiskUsage returns disk usage information for Unix-like systems
func getDiskUsage(path string) (*DiskUsage, error) {
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

	return &DiskUsage{
		Total:     total,
		Used:      used,
		Free:      free,
		Available: available,
	}, nil
}
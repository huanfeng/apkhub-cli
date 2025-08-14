//go:build windows
// +build windows

package system

import (
	"fmt"
	"path/filepath"
	"syscall"
	"unsafe"
)

var (
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	getDiskFreeSpace = kernel32.NewProc("GetDiskFreeSpaceExW")
)

// getDiskUsage returns disk usage information for Windows systems
func getDiskUsage(path string) (*DiskUsage, error) {
	// Get absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Convert to UTF-16 for Windows API
	pathPtr, err := syscall.UTF16PtrFromString(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to convert path to UTF-16: %w", err)
	}

	var freeBytesAvailable, totalBytes, freeBytes uint64

	// Call GetDiskFreeSpaceEx
	ret, _, err := getDiskFreeSpace.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		uintptr(unsafe.Pointer(&freeBytesAvailable)),
		uintptr(unsafe.Pointer(&totalBytes)),
		uintptr(unsafe.Pointer(&freeBytes)),
	)

	if ret == 0 {
		return nil, fmt.Errorf("GetDiskFreeSpaceEx failed: %w", err)
	}

	used := totalBytes - freeBytes

	return &DiskUsage{
		Total:     totalBytes,
		Used:      used,
		Free:      freeBytes,
		Available: freeBytesAvailable,
	}, nil
}
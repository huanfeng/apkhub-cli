package client

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// LogCaptureOptions defines options for capturing device logs.
type LogCaptureOptions struct {
	DeviceID   string
	PackageID  string
	Level      string
	OutputPath string
}

// LogCaptureResult describes the outcome of a log capture.
type LogCaptureResult struct {
	DeviceID   string        `json:"device_id"`
	PackageID  string        `json:"package_id"`
	OutputPath string        `json:"output_path"`
	Level      string        `json:"level"`
	CapturedAt time.Time     `json:"captured_at"`
	SizeBytes  int64         `json:"size_bytes"`
	Note       string        `json:"note,omitempty"`
	Duration   time.Duration `json:"duration"`
}

// CaptureLogs collects logcat output for a package on a given device and saves it to disk.
func (a *ADBManager) CaptureLogs(opts LogCaptureOptions) (*LogCaptureResult, error) {
	if opts.PackageID == "" {
		return nil, fmt.Errorf("package ID is required for log capture")
	}

	if err := a.validateDeviceOnline(opts.DeviceID); err != nil {
		return nil, err
	}

	level := strings.ToUpper(strings.TrimSpace(opts.Level))
	if level == "" {
		level = "I"
	}

	validLevels := map[string]struct{}{"": {}, "V": {}, "D": {}, "I": {}, "W": {}, "E": {}, "F": {}, "S": {}}
	if _, ok := validLevels[level]; !ok {
		return nil, fmt.Errorf("unsupported log level: %s", level)
	}

	outputPath := opts.OutputPath
	if outputPath == "" {
		timestamp := time.Now().Format("20060102-150405")
		sanitizedDevice := strings.ReplaceAll(opts.DeviceID, ":", "_")
		outputPath = filepath.Join("logs", fmt.Sprintf("%s-%s.log", sanitizedDevice, timestamp))
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return nil, fmt.Errorf("failed to prepare log directory: %w", err)
	}

	start := time.Now()
	args := []string{}
	if opts.DeviceID != "" {
		args = append(args, "-s", opts.DeviceID)
	}

	args = append(args, "shell", "pidof", opts.PackageID)
	pidCmd := exec.Command(a.config.ADB.Path, args...)
	pidOutput, _ := pidCmd.Output()
	pid := strings.TrimSpace(string(pidOutput))

	logArgs := []string{}
	if opts.DeviceID != "" {
		logArgs = append(logArgs, "-s", opts.DeviceID)
	}

	logArgs = append(logArgs, "logcat", "-d")
	note := ""
	if pid != "" {
		logArgs = append(logArgs, "--pid", pid)
	} else {
		note = "Process not found, capturing full logcat stream"
	}

	if level != "" {
		logArgs = append(logArgs, fmt.Sprintf("*:%s", level))
	}

	cmd := exec.Command(a.config.ADB.Path, logArgs...)
	output, err := cmd.CombinedOutput()
	duration := time.Since(start)
	if err != nil {
		return nil, fmt.Errorf("failed to capture logs: %w - %s", err, string(output))
	}

	if err := os.WriteFile(outputPath, output, 0o644); err != nil {
		return nil, fmt.Errorf("failed to write logs: %w", err)
	}

	result := &LogCaptureResult{
		DeviceID:   opts.DeviceID,
		PackageID:  opts.PackageID,
		OutputPath: outputPath,
		Level:      level,
		CapturedAt: time.Now(),
		SizeBytes:  int64(len(output)),
		Note:       note,
		Duration:   duration,
	}

	return result, nil
}

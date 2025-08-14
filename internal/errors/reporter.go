package errors

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ErrorReport represents a comprehensive error report
type ErrorReport struct {
	Timestamp   time.Time            `json:"timestamp"`
	Error       *ApkHubError         `json:"error"`
	Environment *EnvironmentInfo     `json:"environment"`
	Context     *OperationContext    `json:"context"`
	Suggestions []RecoverySuggestion `json:"suggestions"`
}

// EnvironmentInfo contains information about the runtime environment
type EnvironmentInfo struct {
	OS           string            `json:"os"`
	Architecture string            `json:"architecture"`
	GoVersion    string            `json:"go_version"`
	ApkHubVersion string           `json:"apkhub_version"`
	WorkingDir   string            `json:"working_dir"`
	ConfigPath   string            `json:"config_path"`
	Dependencies map[string]string `json:"dependencies"`
}

// OperationContext contains information about the operation that failed
type OperationContext struct {
	Command     string            `json:"command"`
	Arguments   []string          `json:"arguments"`
	Flags       map[string]string `json:"flags"`
	WorkingDir  string            `json:"working_dir"`
	Duration    time.Duration     `json:"duration"`
	StepsFailed []string          `json:"steps_failed"`
}

// RecoverySuggestion represents a suggested recovery action
type RecoverySuggestion struct {
	Priority    int    `json:"priority"`    // 1 = high, 2 = medium, 3 = low
	Category    string `json:"category"`    // "immediate", "configuration", "environment"
	Action      string `json:"action"`      // Description of the action
	Command     string `json:"command"`     // Optional command to run
	Automated   bool   `json:"automated"`   // Whether this can be automated
	Description string `json:"description"` // Detailed description
}

// ErrorReporter handles error reporting and diagnosis
type ErrorReporter struct {
	reportDir string
	logger    Logger
}

// NewErrorReporter creates a new error reporter
func NewErrorReporter(reportDir string, logger Logger) *ErrorReporter {
	return &ErrorReporter{
		reportDir: reportDir,
		logger:    logger,
	}
}

// GenerateReport generates a comprehensive error report
func (er *ErrorReporter) GenerateReport(err *ApkHubError, context *OperationContext) (*ErrorReport, error) {
	report := &ErrorReport{
		Timestamp: time.Now(),
		Error:     err,
		Context:   context,
	}
	
	// Gather environment information
	envInfo, envErr := er.gatherEnvironmentInfo()
	if envErr != nil && er.logger != nil {
		er.logger.Warn("Failed to gather environment info: %v", envErr)
	}
	report.Environment = envInfo
	
	// Generate recovery suggestions
	report.Suggestions = er.generateRecoverySuggestions(err, context, envInfo)
	
	return report, nil
}

// SaveReport saves an error report to disk
func (er *ErrorReporter) SaveReport(report *ErrorReport) (string, error) {
	// Ensure report directory exists
	if err := os.MkdirAll(er.reportDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create report directory: %w", err)
	}
	
	// Generate filename
	timestamp := report.Timestamp.Format("20060102_150405")
	filename := fmt.Sprintf("error_report_%s_%s.json", timestamp, report.Error.Code)
	filepath := filepath.Join(er.reportDir, filename)
	
	// Marshal report to JSON
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal report: %w", err)
	}
	
	// Write to file
	if err := os.WriteFile(filepath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write report: %w", err)
	}
	
	return filepath, nil
}

// DisplayReport displays an error report in a user-friendly format
func (er *ErrorReporter) DisplayReport(report *ErrorReport) {
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("🚨 ERROR REPORT")
	fmt.Println(strings.Repeat("=", 80))
	
	// Error information
	fmt.Printf("⏰ Time: %s\n", report.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Printf("🏷️  Type: %s\n", report.Error.Type.String())
	fmt.Printf("🔍 Code: %s\n", report.Error.Code)
	fmt.Printf("💬 Message: %s\n", report.Error.Message)
	
	if report.Error.Cause != nil {
		fmt.Printf("🔗 Cause: %v\n", report.Error.Cause)
	}
	
	// Context information
	if report.Context != nil {
		fmt.Println("\n📋 OPERATION CONTEXT")
		fmt.Println(strings.Repeat("-", 40))
		fmt.Printf("Command: %s\n", report.Context.Command)
		if len(report.Context.Arguments) > 0 {
			fmt.Printf("Arguments: %s\n", strings.Join(report.Context.Arguments, " "))
		}
		if len(report.Context.Flags) > 0 {
			fmt.Println("Flags:")
			for flag, value := range report.Context.Flags {
				fmt.Printf("  --%s: %s\n", flag, value)
			}
		}
		if report.Context.Duration > 0 {
			fmt.Printf("Duration: %v\n", report.Context.Duration)
		}
	}
	
	// Environment information
	if report.Environment != nil {
		fmt.Println("\n🖥️  ENVIRONMENT")
		fmt.Println(strings.Repeat("-", 40))
		fmt.Printf("OS: %s\n", report.Environment.OS)
		fmt.Printf("Architecture: %s\n", report.Environment.Architecture)
		fmt.Printf("Working Directory: %s\n", report.Environment.WorkingDir)
		
		if len(report.Environment.Dependencies) > 0 {
			fmt.Println("Dependencies:")
			for dep, version := range report.Environment.Dependencies {
				fmt.Printf("  %s: %s\n", dep, version)
			}
		}
	}
	
	// Error context
	if len(report.Error.Context) > 0 {
		fmt.Println("\n📝 ERROR CONTEXT")
		fmt.Println(strings.Repeat("-", 40))
		for key, value := range report.Error.Context {
			fmt.Printf("%s: %s\n", key, value)
		}
	}
	
	// Recovery suggestions
	if len(report.Suggestions) > 0 {
		fmt.Println("\n💡 RECOVERY SUGGESTIONS")
		fmt.Println(strings.Repeat("-", 40))
		
		// Sort by priority
		highPriority := []RecoverySuggestion{}
		mediumPriority := []RecoverySuggestion{}
		lowPriority := []RecoverySuggestion{}
		
		for _, suggestion := range report.Suggestions {
			switch suggestion.Priority {
			case 1:
				highPriority = append(highPriority, suggestion)
			case 2:
				mediumPriority = append(mediumPriority, suggestion)
			default:
				lowPriority = append(lowPriority, suggestion)
			}
		}
		
		// Display high priority suggestions first
		if len(highPriority) > 0 {
			fmt.Println("\n🔴 HIGH PRIORITY:")
			for i, suggestion := range highPriority {
				er.displaySuggestion(i+1, suggestion)
			}
		}
		
		if len(mediumPriority) > 0 {
			fmt.Println("\n🟡 MEDIUM PRIORITY:")
			for i, suggestion := range mediumPriority {
				er.displaySuggestion(i+1, suggestion)
			}
		}
		
		if len(lowPriority) > 0 {
			fmt.Println("\n🟢 LOW PRIORITY:")
			for i, suggestion := range lowPriority {
				er.displaySuggestion(i+1, suggestion)
			}
		}
	}
	
	fmt.Println(strings.Repeat("=", 80))
}

// displaySuggestion displays a single recovery suggestion
func (er *ErrorReporter) displaySuggestion(index int, suggestion RecoverySuggestion) {
	fmt.Printf("%d. %s\n", index, suggestion.Action)
	if suggestion.Description != "" {
		fmt.Printf("   %s\n", suggestion.Description)
	}
	if suggestion.Command != "" {
		fmt.Printf("   💻 Command: %s\n", suggestion.Command)
	}
	if suggestion.Automated {
		fmt.Printf("   🤖 This can be automated\n")
	}
	fmt.Println()
}

// gatherEnvironmentInfo gathers information about the runtime environment
func (er *ErrorReporter) gatherEnvironmentInfo() (*EnvironmentInfo, error) {
	info := &EnvironmentInfo{
		Dependencies: make(map[string]string),
	}
	
	// Get working directory
	if wd, err := os.Getwd(); err == nil {
		info.WorkingDir = wd
	}
	
	// TODO: Add more environment gathering logic
	// This would include:
	// - OS detection
	// - Architecture detection
	// - Go version
	// - ApkHub version
	// - Dependency versions (adb, aapt, etc.)
	
	return info, nil
}

// generateRecoverySuggestions generates recovery suggestions based on the error
func (er *ErrorReporter) generateRecoverySuggestions(err *ApkHubError, context *OperationContext, env *EnvironmentInfo) []RecoverySuggestion {
	var suggestions []RecoverySuggestion
	
	// Add error-specific suggestions
	switch err.Type {
	case ErrorTypeDependency:
		suggestions = append(suggestions, RecoverySuggestion{
			Priority:    1,
			Category:    "immediate",
			Action:      "Check and install missing dependencies",
			Command:     "apkhub doctor",
			Automated:   true,
			Description: "Run the doctor command to diagnose and fix dependency issues",
		})
		
		suggestions = append(suggestions, RecoverySuggestion{
			Priority:    2,
			Category:    "environment",
			Action:      "Update system PATH",
			Description: "Ensure all required tools are in your system PATH",
		})
		
	case ErrorTypeNetwork:
		suggestions = append(suggestions, RecoverySuggestion{
			Priority:    1,
			Category:    "immediate",
			Action:      "Check internet connectivity",
			Description: "Verify that you have a stable internet connection",
		})
		
		if strings.Contains(strings.ToLower(err.Message), "timeout") {
			suggestions = append(suggestions, RecoverySuggestion{
				Priority:    2,
				Category:    "configuration",
				Action:      "Increase timeout values",
				Description: "Configure longer timeout values in your settings",
			})
		}
		
	case ErrorTypeFileSystem:
		suggestions = append(suggestions, RecoverySuggestion{
			Priority:    1,
			Category:    "immediate",
			Action:      "Check file permissions",
			Description: "Ensure you have read/write access to the required files and directories",
		})
		
		if strings.Contains(strings.ToLower(err.Message), "space") {
			suggestions = append(suggestions, RecoverySuggestion{
				Priority:    1,
				Category:    "immediate",
				Action:      "Free up disk space",
				Description: "Delete unnecessary files or move them to another location",
			})
		}
		
	case ErrorTypeDevice:
		suggestions = append(suggestions, RecoverySuggestion{
			Priority:    1,
			Category:    "immediate",
			Action:      "Check device connection",
			Command:     "adb devices",
			Description: "Verify that your Android device is properly connected and authorized",
		})
		
		suggestions = append(suggestions, RecoverySuggestion{
			Priority:    2,
			Category:    "configuration",
			Action:      "Enable USB debugging",
			Description: "Make sure USB debugging is enabled in Developer Options on your device",
		})
		
	case ErrorTypeConfiguration:
		suggestions = append(suggestions, RecoverySuggestion{
			Priority:    1,
			Category:    "configuration",
			Action:      "Regenerate configuration",
			Command:     "apkhub init",
			Automated:   true,
			Description: "Create a new configuration file with default settings",
		})
		
	case ErrorTypeParsing:
		suggestions = append(suggestions, RecoverySuggestion{
			Priority:    1,
			Category:    "immediate",
			Action:      "Verify file integrity",
			Description: "Check if the file is corrupted or in an unexpected format",
		})
		
		suggestions = append(suggestions, RecoverySuggestion{
			Priority:    2,
			Category:    "environment",
			Action:      "Update parsing tools",
			Command:     "apkhub doctor --fix",
			Description: "Update aapt, aapt2, and other parsing tools to the latest versions",
		})
	}
	
	// Add general suggestions
	suggestions = append(suggestions, RecoverySuggestion{
		Priority:    3,
		Category:    "environment",
		Action:      "Run system diagnostics",
		Command:     "apkhub doctor --verbose",
		Description: "Get detailed information about your system configuration",
	})
	
	return suggestions
}

// DiagnoseError performs automated diagnosis of an error
func (er *ErrorReporter) DiagnoseError(err *ApkHubError) *DiagnosisResult {
	result := &DiagnosisResult{
		Error:       err,
		Timestamp:   time.Now(),
		Checks:      []DiagnosisCheck{},
		Confidence:  0.0,
		Actionable:  false,
	}
	
	// Perform various diagnostic checks
	result.Checks = append(result.Checks, er.checkDependencies(err))
	result.Checks = append(result.Checks, er.checkFileSystem(err))
	result.Checks = append(result.Checks, er.checkNetwork(err))
	result.Checks = append(result.Checks, er.checkConfiguration(err))
	
	// Calculate overall confidence and actionability
	totalChecks := len(result.Checks)
	passedChecks := 0
	actionableChecks := 0
	
	for _, check := range result.Checks {
		if check.Passed {
			passedChecks++
		}
		if check.Actionable {
			actionableChecks++
		}
	}
	
	if totalChecks > 0 {
		result.Confidence = float64(passedChecks) / float64(totalChecks)
		result.Actionable = actionableChecks > 0
	}
	
	return result
}

// DiagnosisResult represents the result of error diagnosis
type DiagnosisResult struct {
	Error      *ApkHubError     `json:"error"`
	Timestamp  time.Time        `json:"timestamp"`
	Checks     []DiagnosisCheck `json:"checks"`
	Confidence float64          `json:"confidence"`
	Actionable bool             `json:"actionable"`
}

// DiagnosisCheck represents a single diagnostic check
type DiagnosisCheck struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Passed      bool   `json:"passed"`
	Message     string `json:"message"`
	Actionable  bool   `json:"actionable"`
	Action      string `json:"action,omitempty"`
}

// Diagnostic check implementations
func (er *ErrorReporter) checkDependencies(err *ApkHubError) DiagnosisCheck {
	check := DiagnosisCheck{
		Name:        "Dependencies",
		Description: "Check if required dependencies are available",
	}
	
	if err.Type == ErrorTypeDependency {
		check.Passed = false
		check.Message = "Dependency issues detected"
		check.Actionable = true
		check.Action = "Run 'apkhub doctor --fix' to install missing dependencies"
	} else {
		check.Passed = true
		check.Message = "No dependency issues detected"
	}
	
	return check
}

func (er *ErrorReporter) checkFileSystem(err *ApkHubError) DiagnosisCheck {
	check := DiagnosisCheck{
		Name:        "File System",
		Description: "Check file system access and permissions",
	}
	
	if err.Type == ErrorTypeFileSystem || err.Type == ErrorTypePermission {
		check.Passed = false
		check.Message = "File system issues detected"
		check.Actionable = true
		check.Action = "Check file permissions and disk space"
	} else {
		check.Passed = true
		check.Message = "No file system issues detected"
	}
	
	return check
}

func (er *ErrorReporter) checkNetwork(err *ApkHubError) DiagnosisCheck {
	check := DiagnosisCheck{
		Name:        "Network",
		Description: "Check network connectivity",
	}
	
	if err.Type == ErrorTypeNetwork || err.Type == ErrorTypeTimeout {
		check.Passed = false
		check.Message = "Network issues detected"
		check.Actionable = true
		check.Action = "Check internet connection and firewall settings"
	} else {
		check.Passed = true
		check.Message = "No network issues detected"
	}
	
	return check
}

func (er *ErrorReporter) checkConfiguration(err *ApkHubError) DiagnosisCheck {
	check := DiagnosisCheck{
		Name:        "Configuration",
		Description: "Check configuration validity",
	}
	
	if err.Type == ErrorTypeConfiguration {
		check.Passed = false
		check.Message = "Configuration issues detected"
		check.Actionable = true
		check.Action = "Run 'apkhub init' to regenerate configuration"
	} else {
		check.Passed = true
		check.Message = "No configuration issues detected"
	}
	
	return check
}
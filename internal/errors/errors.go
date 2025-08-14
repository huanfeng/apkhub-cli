package errors

import (
	"fmt"
	"runtime"
	"strings"
	"time"
)

// ErrorType represents the type of error
type ErrorType int

const (
	ErrorTypeUnknown ErrorType = iota
	ErrorTypeValidation
	ErrorTypeNetwork
	ErrorTypeFileSystem
	ErrorTypeParsing
	ErrorTypeDependency
	ErrorTypeConfiguration
	ErrorTypeDevice
	ErrorTypePermission
	ErrorTypeTimeout
	ErrorTypeNotFound
)

// String returns the string representation of the error type
func (et ErrorType) String() string {
	switch et {
	case ErrorTypeValidation:
		return "VALIDATION"
	case ErrorTypeNetwork:
		return "NETWORK"
	case ErrorTypeFileSystem:
		return "FILESYSTEM"
	case ErrorTypeParsing:
		return "PARSING"
	case ErrorTypeDependency:
		return "DEPENDENCY"
	case ErrorTypeConfiguration:
		return "CONFIGURATION"
	case ErrorTypeDevice:
		return "DEVICE"
	case ErrorTypePermission:
		return "PERMISSION"
	case ErrorTypeTimeout:
		return "TIMEOUT"
	case ErrorTypeNotFound:
		return "NOT_FOUND"
	default:
		return "UNKNOWN"
	}
}

// ApkHubError represents an enhanced error with context and suggestions
type ApkHubError struct {
	Type        ErrorType         `json:"type"`
	Code        string            `json:"code"`
	Message     string            `json:"message"`
	Cause       error             `json:"cause,omitempty"`
	Context     map[string]string `json:"context,omitempty"`
	Suggestions []string          `json:"suggestions,omitempty"`
	Timestamp   time.Time         `json:"timestamp"`
	Stack       []string          `json:"stack,omitempty"`
	Retryable   bool              `json:"retryable"`
}

// Error implements the error interface
func (e *ApkHubError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

// Unwrap returns the underlying error
func (e *ApkHubError) Unwrap() error {
	return e.Cause
}

// Is checks if the error matches the target
func (e *ApkHubError) Is(target error) bool {
	if t, ok := target.(*ApkHubError); ok {
		return e.Type == t.Type && e.Code == t.Code
	}
	return false
}

// WithContext adds context to the error
func (e *ApkHubError) WithContext(key, value string) *ApkHubError {
	if e.Context == nil {
		e.Context = make(map[string]string)
	}
	e.Context[key] = value
	return e
}

// WithSuggestion adds a suggestion to the error
func (e *ApkHubError) WithSuggestion(suggestion string) *ApkHubError {
	e.Suggestions = append(e.Suggestions, suggestion)
	return e
}

// WithSuggestions adds multiple suggestions to the error
func (e *ApkHubError) WithSuggestions(suggestions []string) *ApkHubError {
	e.Suggestions = append(e.Suggestions, suggestions...)
	return e
}

// SetRetryable marks the error as retryable or not
func (e *ApkHubError) SetRetryable(retryable bool) *ApkHubError {
	e.Retryable = retryable
	return e
}

// FormatDetailed returns a detailed error message with context and suggestions
func (e *ApkHubError) FormatDetailed() string {
	var builder strings.Builder
	
	// Error header
	builder.WriteString(fmt.Sprintf("❌ %s Error [%s]: %s\n", e.Type.String(), e.Code, e.Message))
	
	// Context
	if len(e.Context) > 0 {
		builder.WriteString("\n📋 Context:\n")
		for key, value := range e.Context {
			builder.WriteString(fmt.Sprintf("   %s: %s\n", key, value))
		}
	}
	
	// Underlying cause
	if e.Cause != nil {
		builder.WriteString(fmt.Sprintf("\n🔍 Underlying cause: %v\n", e.Cause))
	}
	
	// Suggestions
	if len(e.Suggestions) > 0 {
		builder.WriteString("\n💡 Suggestions:\n")
		for _, suggestion := range e.Suggestions {
			builder.WriteString(fmt.Sprintf("   • %s\n", suggestion))
		}
	}
	
	// Retry information
	if e.Retryable {
		builder.WriteString("\n🔄 This operation can be retried\n")
	}
	
	return builder.String()
}

// NewError creates a new ApkHubError
func NewError(errorType ErrorType, code, message string) *ApkHubError {
	return &ApkHubError{
		Type:      errorType,
		Code:      code,
		Message:   message,
		Timestamp: time.Now(),
		Context:   make(map[string]string),
		Stack:     captureStack(),
	}
}

// WrapError wraps an existing error with ApkHubError
func WrapError(err error, errorType ErrorType, code, message string) *ApkHubError {
	return &ApkHubError{
		Type:      errorType,
		Code:      code,
		Message:   message,
		Cause:     err,
		Timestamp: time.Now(),
		Context:   make(map[string]string),
		Stack:     captureStack(),
	}
}

// captureStack captures the current stack trace
func captureStack() []string {
	var stack []string
	
	// Skip the first few frames (this function and error creation)
	for i := 2; i < 10; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		
		fn := runtime.FuncForPC(pc)
		if fn == nil {
			continue
		}
		
		// Only include frames from our project
		if strings.Contains(file, "apkhub-cli") {
			stack = append(stack, fmt.Sprintf("%s:%d %s", file, line, fn.Name()))
		}
	}
	
	return stack
}

// Common error constructors

// NewValidationError creates a validation error
func NewValidationError(code, message string) *ApkHubError {
	return NewError(ErrorTypeValidation, code, message).
		WithSuggestion("Check the input parameters and try again")
}

// NewNetworkError creates a network error
func NewNetworkError(code, message string) *ApkHubError {
	return NewError(ErrorTypeNetwork, code, message).
		SetRetryable(true).
		WithSuggestions([]string{
			"Check your internet connection",
			"Verify the server is accessible",
			"Try again in a few moments",
		})
}

// NewFileSystemError creates a filesystem error
func NewFileSystemError(code, message string) *ApkHubError {
	return NewError(ErrorTypeFileSystem, code, message).
		WithSuggestions([]string{
			"Check file permissions",
			"Ensure the path exists",
			"Verify disk space availability",
		})
}

// NewParsingError creates a parsing error
func NewParsingError(code, message string) *ApkHubError {
	return NewError(ErrorTypeParsing, code, message).
		WithSuggestions([]string{
			"Verify the file format is correct",
			"Check if the file is corrupted",
			"Try with a different file",
		})
}

// NewDependencyError creates a dependency error
func NewDependencyError(code, message string) *ApkHubError {
	return NewError(ErrorTypeDependency, code, message).
		WithSuggestions([]string{
			"Run 'apkhub doctor' to check dependencies",
			"Install missing dependencies",
			"Update existing tools to latest versions",
		})
}

// NewConfigurationError creates a configuration error
func NewConfigurationError(code, message string) *ApkHubError {
	return NewError(ErrorTypeConfiguration, code, message).
		WithSuggestions([]string{
			"Check the configuration file syntax",
			"Verify all required settings are present",
			"Run 'apkhub init' to regenerate configuration",
		})
}

// NewDeviceError creates a device error
func NewDeviceError(code, message string) *ApkHubError {
	return NewError(ErrorTypeDevice, code, message).
		WithSuggestions([]string{
			"Check device connection",
			"Enable USB debugging",
			"Authorize this computer on the device",
			"Try reconnecting the device",
		})
}

// NewPermissionError creates a permission error
func NewPermissionError(code, message string) *ApkHubError {
	return NewError(ErrorTypePermission, code, message).
		WithSuggestions([]string{
			"Check file/directory permissions",
			"Run with appropriate privileges",
			"Ensure you have write access to the target location",
		})
}

// NewTimeoutError creates a timeout error
func NewTimeoutError(code, message string) *ApkHubError {
	return NewError(ErrorTypeTimeout, code, message).
		SetRetryable(true).
		WithSuggestions([]string{
			"Increase timeout duration",
			"Check network connectivity",
			"Try the operation again",
		})
}

// NewNotFoundError creates a not found error
func NewNotFoundError(code, message string) *ApkHubError {
	return NewError(ErrorTypeNotFound, code, message).
		WithSuggestions([]string{
			"Verify the resource exists",
			"Check the path or identifier",
			"Ensure proper permissions to access the resource",
		})
}

// ErrorHandler provides centralized error handling
type ErrorHandler struct {
	logger Logger
	stats  *ErrorStats
}

// Logger interface for error logging
type Logger interface {
	Error(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Debug(msg string, args ...interface{})
}

// ErrorStats tracks error statistics
type ErrorStats struct {
	TotalErrors   int                    `json:"total_errors"`
	ErrorsByType  map[ErrorType]int      `json:"errors_by_type"`
	ErrorsByCode  map[string]int         `json:"errors_by_code"`
	LastError     *ApkHubError           `json:"last_error,omitempty"`
	LastErrorTime time.Time              `json:"last_error_time"`
}

// NewErrorHandler creates a new error handler
func NewErrorHandler(logger Logger) *ErrorHandler {
	return &ErrorHandler{
		logger: logger,
		stats: &ErrorStats{
			ErrorsByType: make(map[ErrorType]int),
			ErrorsByCode: make(map[string]int),
		},
	}
}

// Handle handles an error with logging and statistics
func (eh *ErrorHandler) Handle(err error) {
	if err == nil {
		return
	}
	
	// Convert to ApkHubError if needed
	var apkErr *ApkHubError
	if e, ok := err.(*ApkHubError); ok {
		apkErr = e
	} else {
		apkErr = WrapError(err, ErrorTypeUnknown, "UNKNOWN", err.Error())
	}
	
	// Update statistics
	eh.updateStats(apkErr)
	
	// Log the error
	if eh.logger != nil {
		eh.logger.Error("Error occurred: %s [%s] %s", apkErr.Type.String(), apkErr.Code, apkErr.Message)
		
		// Log context if available
		if len(apkErr.Context) > 0 {
			for key, value := range apkErr.Context {
				eh.logger.Debug("Error context: %s = %s", key, value)
			}
		}
	}
}

// HandleWithRecovery handles an error and provides recovery suggestions
func (eh *ErrorHandler) HandleWithRecovery(err error) *ApkHubError {
	if err == nil {
		return nil
	}
	
	// Convert to ApkHubError if needed
	var apkErr *ApkHubError
	if e, ok := err.(*ApkHubError); ok {
		apkErr = e
	} else {
		apkErr = WrapError(err, ErrorTypeUnknown, "UNKNOWN", err.Error())
	}
	
	// Add recovery suggestions based on error type
	eh.addRecoverySuggestions(apkErr)
	
	// Handle the error
	eh.Handle(apkErr)
	
	return apkErr
}

// updateStats updates error statistics
func (eh *ErrorHandler) updateStats(err *ApkHubError) {
	eh.stats.TotalErrors++
	eh.stats.ErrorsByType[err.Type]++
	eh.stats.ErrorsByCode[err.Code]++
	eh.stats.LastError = err
	eh.stats.LastErrorTime = time.Now()
}

// addRecoverySuggestions adds recovery suggestions based on error patterns
func (eh *ErrorHandler) addRecoverySuggestions(err *ApkHubError) {
	// Add type-specific suggestions
	switch err.Type {
	case ErrorTypeNetwork:
		if strings.Contains(strings.ToLower(err.Message), "timeout") {
			err.WithSuggestion("Consider increasing the timeout value")
		}
		if strings.Contains(strings.ToLower(err.Message), "connection refused") {
			err.WithSuggestion("Check if the service is running")
		}
	case ErrorTypeFileSystem:
		if strings.Contains(strings.ToLower(err.Message), "permission denied") {
			err.WithSuggestion("Run with elevated privileges or check file permissions")
		}
		if strings.Contains(strings.ToLower(err.Message), "no space left") {
			err.WithSuggestion("Free up disk space and try again")
		}
	case ErrorTypeDependency:
		if strings.Contains(strings.ToLower(err.Message), "not found") {
			err.WithSuggestion("Install the missing dependency using your package manager")
		}
	}
}

// GetStats returns error statistics
func (eh *ErrorHandler) GetStats() *ErrorStats {
	return eh.stats
}

// Reset resets error statistics
func (eh *ErrorHandler) Reset() {
	eh.stats = &ErrorStats{
		ErrorsByType: make(map[ErrorType]int),
		ErrorsByCode: make(map[string]int),
	}
}

// Global error handler
var globalErrorHandler *ErrorHandler

// InitGlobalErrorHandler initializes the global error handler
func InitGlobalErrorHandler(logger Logger) {
	globalErrorHandler = NewErrorHandler(logger)
}

// GetGlobalErrorHandler returns the global error handler
func GetGlobalErrorHandler() *ErrorHandler {
	if globalErrorHandler == nil {
		globalErrorHandler = NewErrorHandler(nil)
	}
	return globalErrorHandler
}

// Handle handles an error using the global error handler
func Handle(err error) {
	GetGlobalErrorHandler().Handle(err)
}

// HandleWithRecovery handles an error with recovery using the global error handler
func HandleWithRecovery(err error) *ApkHubError {
	return GetGlobalErrorHandler().HandleWithRecovery(err)
}
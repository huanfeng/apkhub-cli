package utils

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// LogLevel represents the logging level
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
	LogLevelFatal
)

// String returns the string representation of the log level
func (l LogLevel) String() string {
	switch l {
	case LogLevelDebug:
		return "DEBUG"
	case LogLevelInfo:
		return "INFO"
	case LogLevelWarn:
		return "WARN"
	case LogLevelError:
		return "ERROR"
	case LogLevelFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// ColorCode returns the ANSI color code for the log level
func (l LogLevel) ColorCode() string {
	switch l {
	case LogLevelDebug:
		return "\033[36m" // Cyan
	case LogLevelInfo:
		return "\033[32m" // Green
	case LogLevelWarn:
		return "\033[33m" // Yellow
	case LogLevelError:
		return "\033[31m" // Red
	case LogLevelFatal:
		return "\033[35m" // Magenta
	default:
		return "\033[0m" // Reset
	}
}

// Logger interface defines the logging contract
type Logger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	Fatal(msg string, args ...interface{})

	SetLevel(level LogLevel)
	SetOutput(w io.Writer)
	SetFormat(format LogFormat)

	WithField(key string, value interface{}) Logger
	WithFields(fields map[string]interface{}) Logger
}

// LogFormat represents the log output format
type LogFormat int

const (
	LogFormatText LogFormat = iota
	LogFormatJSON
	LogFormatCompact
)

// LoggerConfig contains logger configuration
type LoggerConfig struct {
	Level       LogLevel
	Format      LogFormat
	Output      io.Writer
	EnableFile  bool
	FilePath    string
	MaxSize     int64 // Max file size in bytes
	MaxFiles    int   // Max number of log files to keep
	EnableColor bool
}

// DefaultLoggerConfig returns a default logger configuration
func DefaultLoggerConfig() *LoggerConfig {
	return &LoggerConfig{
		Level:       LogLevelInfo,
		Format:      LogFormatText,
		Output:      os.Stdout,
		EnableFile:  false,
		EnableColor: true,
	}
}

// ApkHubLogger is the main logger implementation
type ApkHubLogger struct {
	config *LoggerConfig
	logger *log.Logger
	fields map[string]interface{}
	file   *os.File
}

// NewLogger creates a new logger with the given configuration
func NewLogger(config *LoggerConfig) (*ApkHubLogger, error) {
	if config == nil {
		config = DefaultLoggerConfig()
	}

	logger := &ApkHubLogger{
		config: config,
		fields: make(map[string]interface{}),
	}

	// Setup output
	if err := logger.setupOutput(); err != nil {
		return nil, fmt.Errorf("failed to setup logger output: %w", err)
	}

	return logger, nil
}

// setupOutput configures the logger output
func (l *ApkHubLogger) setupOutput() error {
	var output io.Writer = l.config.Output

	// Setup file output if enabled
	if l.config.EnableFile && l.config.FilePath != "" {
		// Ensure directory exists
		dir := filepath.Dir(l.config.FilePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create log directory: %w", err)
		}

		// Open log file
		file, err := os.OpenFile(l.config.FilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return fmt.Errorf("failed to open log file: %w", err)
		}

		l.file = file

		// Use both stdout and file
		output = io.MultiWriter(l.config.Output, file)
	}

	l.logger = log.New(output, "", 0)
	return nil
}

// Debug logs a debug message
func (l *ApkHubLogger) Debug(msg string, args ...interface{}) {
	if l.config.Level <= LogLevelDebug {
		l.log(LogLevelDebug, msg, args...)
	}
}

// Info logs an info message
func (l *ApkHubLogger) Info(msg string, args ...interface{}) {
	if l.config.Level <= LogLevelInfo {
		l.log(LogLevelInfo, msg, args...)
	}
}

// Warn logs a warning message
func (l *ApkHubLogger) Warn(msg string, args ...interface{}) {
	if l.config.Level <= LogLevelWarn {
		l.log(LogLevelWarn, msg, args...)
	}
}

// Error logs an error message
func (l *ApkHubLogger) Error(msg string, args ...interface{}) {
	if l.config.Level <= LogLevelError {
		l.log(LogLevelError, msg, args...)
	}
}

// Fatal logs a fatal message and exits
func (l *ApkHubLogger) Fatal(msg string, args ...interface{}) {
	l.log(LogLevelFatal, msg, args...)
	os.Exit(1)
}

// log performs the actual logging
func (l *ApkHubLogger) log(level LogLevel, msg string, args ...interface{}) {
	// Format message
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}

	// Create log entry
	entry := l.createLogEntry(level, msg)

	// Output log entry
	l.logger.Print(entry)
}

// createLogEntry creates a formatted log entry
func (l *ApkHubLogger) createLogEntry(level LogLevel, msg string) string {
	timestamp := time.Now().Format("2006-01-02 15:04:05")

	switch l.config.Format {
	case LogFormatJSON:
		return l.createJSONEntry(level, msg, timestamp)
	case LogFormatCompact:
		return l.createCompactEntry(level, msg, timestamp)
	default:
		return l.createTextEntry(level, msg, timestamp)
	}
}

// createTextEntry creates a text format log entry
func (l *ApkHubLogger) createTextEntry(level LogLevel, msg string, timestamp string) string {
	var builder strings.Builder

	// Add color if enabled
	if l.config.EnableColor {
		builder.WriteString(level.ColorCode())
	}

	// Add timestamp and level
	builder.WriteString(fmt.Sprintf("[%s] %s", timestamp, level.String()))

	// Add fields if any
	if len(l.fields) > 0 {
		builder.WriteString(" {")
		first := true
		for k, v := range l.fields {
			if !first {
				builder.WriteString(", ")
			}
			builder.WriteString(fmt.Sprintf("%s=%v", k, v))
			first = false
		}
		builder.WriteString("}")
	}

	// Add message
	builder.WriteString(fmt.Sprintf(" %s", msg))

	// Reset color if enabled
	if l.config.EnableColor {
		builder.WriteString("\033[0m")
	}

	return builder.String()
}

// createCompactEntry creates a compact format log entry
func (l *ApkHubLogger) createCompactEntry(level LogLevel, msg string, timestamp string) string {
	levelChar := string(level.String()[0])
	timeShort := timestamp[11:19] // Just time part

	var builder strings.Builder

	if l.config.EnableColor {
		builder.WriteString(level.ColorCode())
	}

	builder.WriteString(fmt.Sprintf("%s %s %s", levelChar, timeShort, msg))

	if l.config.EnableColor {
		builder.WriteString("\033[0m")
	}

	return builder.String()
}

// createJSONEntry creates a JSON format log entry
func (l *ApkHubLogger) createJSONEntry(level LogLevel, msg string, timestamp string) string {
	entry := map[string]interface{}{
		"timestamp": timestamp,
		"level":     level.String(),
		"message":   msg,
	}

	// Add fields
	for k, v := range l.fields {
		entry[k] = v
	}

	// Simple JSON formatting (avoiding external dependencies)
	var parts []string
	for k, v := range entry {
		parts = append(parts, fmt.Sprintf(`"%s":"%v"`, k, v))
	}

	return fmt.Sprintf("{%s}", strings.Join(parts, ","))
}

// SetLevel sets the logging level
func (l *ApkHubLogger) SetLevel(level LogLevel) {
	l.config.Level = level
}

// SetOutput sets the output writer
func (l *ApkHubLogger) SetOutput(w io.Writer) {
	l.config.Output = w
	l.setupOutput()
}

// SetFormat sets the log format
func (l *ApkHubLogger) SetFormat(format LogFormat) {
	l.config.Format = format
}

// WithField returns a logger with an additional field
func (l *ApkHubLogger) WithField(key string, value interface{}) Logger {
	newLogger := &ApkHubLogger{
		config: l.config,
		logger: l.logger,
		fields: make(map[string]interface{}),
		file:   l.file,
	}

	// Copy existing fields
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}

	// Add new field
	newLogger.fields[key] = value

	return newLogger
}

// WithFields returns a logger with additional fields
func (l *ApkHubLogger) WithFields(fields map[string]interface{}) Logger {
	newLogger := &ApkHubLogger{
		config: l.config,
		logger: l.logger,
		fields: make(map[string]interface{}),
		file:   l.file,
	}

	// Copy existing fields
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}

	// Add new fields
	for k, v := range fields {
		newLogger.fields[k] = v
	}

	return newLogger
}

// Close closes the logger and any open files
func (l *ApkHubLogger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// Global logger instance
var globalLogger Logger

// InitGlobalLogger initializes the global logger
func InitGlobalLogger(config *LoggerConfig) error {
	logger, err := NewLogger(config)
	if err != nil {
		return err
	}
	globalLogger = logger
	return nil
}

// GetGlobalLogger returns the global logger instance
func GetGlobalLogger() Logger {
	if globalLogger == nil {
		// Initialize with default config if not set
		logger, _ := NewLogger(DefaultLoggerConfig())
		globalLogger = logger
	}
	return globalLogger
}

// Convenience functions for global logger
func Debug(msg string, args ...interface{}) {
	GetGlobalLogger().Debug(msg, args...)
}

func Info(msg string, args ...interface{}) {
	GetGlobalLogger().Info(msg, args...)
}

func Warn(msg string, args ...interface{}) {
	GetGlobalLogger().Warn(msg, args...)
}

func Error(msg string, args ...interface{}) {
	GetGlobalLogger().Error(msg, args...)
}

func Fatal(msg string, args ...interface{}) {
	GetGlobalLogger().Fatal(msg, args...)
}

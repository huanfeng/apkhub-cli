package apk

import (
	"fmt"
	"time"
)

// APKParser defines the interface for APK parsers
type APKParser interface {
	ParseAPK(path string) (*APKInfo, error)
	GetParserInfo() ParserInfo
	CanParse(path string) bool
}

// ParserInfo contains information about a parser
type ParserInfo struct {
	Name         string
	Version      string
	Capabilities []string
	Available    bool
	Priority     int // Lower number = higher priority
}

// ParseResult contains the result of parsing with metadata
type ParseResult struct {
	APKInfo  *APKInfo
	Parser   string
	Duration time.Duration
	Warnings []string
	Errors   []string
}

// ParserChain manages multiple APK parsers
type ParserChain struct {
	parsers []APKParser
	logger  Logger
}

// Logger interface for parser chain logging
type Logger interface {
	Debug(format string, args ...interface{})
	Info(format string, args ...interface{})
	Warn(format string, args ...interface{})
	Error(format string, args ...interface{})
}

// SimpleLogger is a basic logger implementation
type SimpleLogger struct{}

func (l *SimpleLogger) Debug(format string, args ...interface{}) {
	// Debug messages are not shown by default
}

func (l *SimpleLogger) Info(format string, args ...interface{}) {
	// Info messages could be shown with verbose flag
}

func (l *SimpleLogger) Warn(format string, args ...interface{}) {
	// Always show warnings
	if len(args) > 0 {
		fmt.Printf("Warning: "+format+"\n", args...)
	} else {
		fmt.Printf("Warning: %s\n", format)
	}
}

func (l *SimpleLogger) Error(format string, args ...interface{}) {
	// Always show errors
	if len(args) > 0 {
		fmt.Printf("Error: "+format+"\n", args...)
	} else {
		fmt.Printf("Error: %s\n", format)
	}
}

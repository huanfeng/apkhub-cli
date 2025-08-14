package apk

import (
	"fmt"
	"sort"
	"time"
)

// NewParserChain creates a new parser chain
func NewParserChain(logger Logger) *ParserChain {
	if logger == nil {
		logger = &SimpleLogger{}
	}

	return &ParserChain{
		parsers: make([]APKParser, 0),
		logger:  logger,
	}
}

// AddParser adds a parser to the chain
func (pc *ParserChain) AddParser(parser APKParser) {
	pc.parsers = append(pc.parsers, parser)
	pc.sortParsersByPriority()
}

// sortParsersByPriority sorts parsers by their priority (lower number = higher priority)
func (pc *ParserChain) sortParsersByPriority() {
	sort.Slice(pc.parsers, func(i, j int) bool {
		return pc.parsers[i].GetParserInfo().Priority < pc.parsers[j].GetParserInfo().Priority
	})
}

// ParseAPK attempts to parse APK using the parser chain
func (pc *ParserChain) ParseAPK(path string) (*ParseResult, error) {
	if len(pc.parsers) == 0 {
		return nil, fmt.Errorf("no parsers available")
	}

	var lastErr error
	var warnings []string
	var errors []string

	pc.logger.Debug("Starting APK parsing with %d parsers", len(pc.parsers))

	for _, parser := range pc.parsers {
		info := parser.GetParserInfo()

		// Skip unavailable parsers
		if !info.Available {
			pc.logger.Debug("Skipping unavailable parser: %s", info.Name)
			continue
		}

		// Check if parser can handle this file
		if !parser.CanParse(path) {
			pc.logger.Debug("Parser %s cannot parse file: %s", info.Name, path)
			continue
		}

		pc.logger.Debug("Trying parser: %s", info.Name)
		startTime := time.Now()

		apkInfo, err := parser.ParseAPK(path)
		duration := time.Since(startTime)

		if err != nil {
			pc.logger.Warn("Parser %s failed: %v", info.Name, err)
			errors = append(errors, fmt.Sprintf("%s: %v", info.Name, err))
			lastErr = err
			continue
		}

		// Success!
		pc.logger.Info("Successfully parsed APK using %s (took %v)", info.Name, duration)

		result := &ParseResult{
			APKInfo:  apkInfo,
			Parser:   info.Name,
			Duration: duration,
			Warnings: warnings,
			Errors:   errors,
		}

		return result, nil
	}

	// All parsers failed
	if lastErr != nil {
		// Check if AAPT is the issue
		aaptAvailable := false
		for _, parser := range pc.parsers {
			if parser.GetParserInfo().Name == "AAPT" && parser.GetParserInfo().Available {
				aaptAvailable = true
				break
			}
		}

		if !aaptAvailable {
			return nil, fmt.Errorf("APK parsing failed. AndroidBinary parser failed and AAPT is not available.\n\n"+
				"To fix this issue:\n"+
				"1. Install aapt2 tools (run 'apkhub doctor' for installation instructions)\n"+
				"2. Or ensure the APK file is not corrupted\n\n"+
				"Last error: %w", lastErr)
		}

		return nil, fmt.Errorf("all available parsers failed, last error: %w", lastErr)
	}

	return nil, fmt.Errorf("no suitable parser found for file: %s", path)
}

// GetAvailableParsers returns information about all available parsers
func (pc *ParserChain) GetAvailableParsers() []ParserInfo {
	var infos []ParserInfo
	for _, parser := range pc.parsers {
		infos = append(infos, parser.GetParserInfo())
	}
	return infos
}

// GetParserStats returns statistics about parser usage
func (pc *ParserChain) GetParserStats() map[string]int {
	// This would be implemented with actual usage tracking
	// For now, return empty map
	return make(map[string]int)
}

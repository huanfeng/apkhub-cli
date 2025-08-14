package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/huanfeng/apkhub-cli/internal/errors"
	"github.com/huanfeng/apkhub-cli/internal/version"
	"github.com/huanfeng/apkhub-cli/pkg/utils"
	"github.com/spf13/cobra"
)

var (
	cfgFile    string
	workDir    string
	verbose    bool
	debug      bool
	logFile    string
	noColor    bool
)

var rootCmd = &cobra.Command{
	Use:   "apkhub",
	Short: "ApkHub CLI - A tool for managing APK repositories",
	Long: `ApkHub CLI is a command-line tool for managing distributed APK repositories.
It supports parsing APK files, generating repository indexes, and maintaining APK collections.`,
	Version: version.Short(),
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return initializeGlobalSystems()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		// Use enhanced error handling if available
		if apkErr := errors.HandleWithRecovery(err); apkErr != nil {
			fmt.Fprintf(os.Stderr, "%s\n", apkErr.FormatDetailed())
		} else {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		os.Exit(1)
	}
}

// initializeGlobalSystems initializes the global logging and error handling systems
func initializeGlobalSystems() error {
	// Configure logger
	logConfig := utils.DefaultLoggerConfig()
	
	// Set log level based on flags
	if debug {
		logConfig.Level = utils.LogLevelDebug
	} else if verbose {
		logConfig.Level = utils.LogLevelInfo
	} else {
		logConfig.Level = utils.LogLevelWarn
	}
	
	// Configure color output
	logConfig.EnableColor = !noColor
	
	// Configure file output if specified
	if logFile != "" {
		logConfig.EnableFile = true
		if filepath.IsAbs(logFile) {
			logConfig.FilePath = logFile
		} else {
			logConfig.FilePath = filepath.Join(workDir, logFile)
		}
	}
	
	// Initialize global logger
	if err := utils.InitGlobalLogger(logConfig); err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	
	// Initialize global error handler
	logger := utils.GetGlobalLogger()
	errors.InitGlobalErrorHandler(logger)
	
	// Log initialization
	if debug {
		logger.Debug("Global systems initialized")
		logger.Debug("Working directory: %s", workDir)
		logger.Debug("Config file: %s", cfgFile)
		logger.Debug("Log level: %s", logConfig.Level.String())
	}
	
	return nil
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./apkhub.yaml)")
	rootCmd.PersistentFlags().StringVarP(&workDir, "work-dir", "w", ".", "working directory for relative paths")
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "enable verbose output")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "enable debug output")
	rootCmd.PersistentFlags().StringVar(&logFile, "log-file", "", "write logs to file")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "disable colored output")
}

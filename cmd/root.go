package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/huanfeng/apkhub/internal/errors"
	"github.com/huanfeng/apkhub/internal/i18n"
	"github.com/huanfeng/apkhub/internal/version"
	"github.com/huanfeng/apkhub/pkg/utils"
	"github.com/spf13/cobra"
)

var (
	cfgFile string
	workDir string
	verbose bool
	debug   bool
	logFile string
	noColor bool
	lang    string
)

var rootCmd = &cobra.Command{
	Use:     "apkhub",
	Short:   i18n.T("cmd.root.short"),
	Long:    i18n.T("cmd.root.long"),
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
			fmt.Fprintf(os.Stderr, "%s\n", i18n.T("errors.generic", map[string]interface{}{
				"error": err,
			}))
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
	cobra.OnInitialize(initLocalization)
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", i18n.T("flags.config"))
	rootCmd.PersistentFlags().StringVarP(&workDir, "work-dir", "w", ".", i18n.T("flags.workDir"))
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, i18n.T("flags.verbose"))
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, i18n.T("flags.debug"))
	rootCmd.PersistentFlags().StringVar(&logFile, "log-file", "", i18n.T("flags.logFile"))
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, i18n.T("flags.noColor"))
	rootCmd.PersistentFlags().StringVar(&lang, "lang", "", i18n.T("flags.lang"))

	// Ensure help also goes through localization.
	defaultHelp := rootCmd.HelpFunc()
	rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		initLocalization()
		defaultHelp(cmd, args)
	})
}

func initLocalization() {
	if err := i18n.Init(lang); err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize localization: %v\n", err)
		return
	}
	applyCommandLocalization()
}

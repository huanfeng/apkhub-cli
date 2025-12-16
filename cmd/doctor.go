package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/huanfeng/apkhub-cli/internal/errors"
	"github.com/huanfeng/apkhub-cli/internal/i18n"
	"github.com/huanfeng/apkhub-cli/pkg/system"
	"github.com/huanfeng/apkhub-cli/pkg/utils"
	"github.com/spf13/cobra"
)

var (
	doctorFix     bool
	doctorVerbose bool
	doctorCheck   string
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: i18n.T("cmd.doctor.short"),
	Long:  i18n.T("cmd.doctor.long"),
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := utils.GetGlobalLogger()
		logger.Info(i18n.T("cmd.doctor.log.start"))

		fmt.Println(i18n.T("cmd.doctor.title"))
		fmt.Println(strings.Repeat("=", 50))

		// Initialize checkers
		depManager := system.NewDependencyManager()
		resourceChecker := system.NewResourceChecker(logger)

		var allPassed = true
		var issues []string
		var suggestions []string

		// 1. Check dependencies
		fmt.Println()
		fmt.Println(i18n.T("cmd.doctor.check.dependencies"))
		if err := checkDependencies(depManager, &allPassed, &issues, &suggestions); err != nil {
			logger.Error(i18n.T("cmd.doctor.log.depFailed"), err)
		}

		// 2. Skip system resource checks - avoid accessing sensitive system data
		// Basic file system access will be checked separately

		// 3. Skip configuration check - apkhub.yaml is for self-hosted repos only
		// Configuration is optional and not required for basic CLI functionality

		// 3. Check file system access
		fmt.Println()
		fmt.Println(i18n.T("cmd.doctor.check.fs"))
		if err := checkFileSystemAccess(resourceChecker, &allPassed, &issues, &suggestions); err != nil {
			logger.Error(i18n.T("cmd.doctor.log.fsFailed"), err)
		}

		// 4. Check network connectivity (if needed)
		if doctorCheck == "all" || doctorCheck == "network" {
			fmt.Println()
			fmt.Println(i18n.T("cmd.doctor.check.network"))
			networkChecker := system.NewNetworkChecker(logger)
			if err := checkNetworkConnectivity(networkChecker, &allPassed, &issues, &suggestions); err != nil {
				logger.Error(i18n.T("cmd.doctor.log.netFailed"), err)
			}
		}

		// Display results
		fmt.Println()
		fmt.Println(strings.Repeat("=", 50))
		fmt.Println(i18n.T("cmd.doctor.results.title"))
		fmt.Println(strings.Repeat("=", 50))

		if allPassed {
			fmt.Println(i18n.T("cmd.doctor.results.allPassed"))
		} else {
			fmt.Printf(i18n.T("cmd.doctor.results.issueCount")+"\n\n", len(issues))

			for i, issue := range issues {
				fmt.Printf(i18n.T("cmd.doctor.results.issueItem")+"\n", i+1, issue)
			}

			if len(suggestions) > 0 {
				fmt.Println()
				fmt.Println(i18n.T("cmd.doctor.results.suggestionTitle"))
				for i, suggestion := range suggestions {
					fmt.Printf(i18n.T("cmd.doctor.results.suggestionItem")+"\n", i+1, suggestion)
				}
			}

			if doctorFix {
				fmt.Println()
				fmt.Println(i18n.T("cmd.doctor.autofix.start"))
				if err := attemptDoctorAutoFix(depManager, issues, suggestions); err != nil {
					logger.Error(i18n.T("cmd.doctor.log.autofixFailed"), err)
					return errors.WrapError(err, errors.ErrorTypeConfiguration, "AUTO_FIX_FAILED",
						i18n.T("cmd.doctor.autofix.errWrap")).
						WithSuggestion(i18n.T("cmd.doctor.autofix.suggestion"))
				}
			} else {
				fmt.Println()
				fmt.Println(i18n.T("cmd.doctor.autofix.hint"))
			}
		}

		if !allPassed {
			return fmt.Errorf(i18n.T("cmd.doctor.errFound"))
		}

		return nil
	},
}

// checkDependencies checks all required dependencies
func checkDependencies(depManager system.DependencyManager, allPassed *bool, issues *[]string, suggestions *[]string) error {
	depsMap := depManager.CheckAll()

	// Convert map to slice for easier processing
	var deps []system.DependencyStatus
	for _, dep := range depsMap {
		deps = append(deps, dep)
	}

	var missingRequired []string
	var missingOptional []string

	for _, dep := range deps {
		if dep.Available {
			fmt.Printf(i18n.T("cmd.doctor.dep.available")+"\n", dep.Name, dep.Version)
		} else {
			if dep.Required {
				fmt.Printf(i18n.T("cmd.doctor.dep.missingRequired")+"\n", dep.Name)
				missingRequired = append(missingRequired, dep.Name)
				*allPassed = false
			} else {
				fmt.Printf(i18n.T("cmd.doctor.dep.missingOptional")+"\n", dep.Name)
				missingOptional = append(missingOptional, dep.Name)
			}
		}
	}

	if len(missingRequired) > 0 {
		*issues = append(*issues, fmt.Sprintf(i18n.T("cmd.doctor.dep.issueRequired"), strings.Join(missingRequired, ", ")))
		*suggestions = append(*suggestions, i18n.T("cmd.doctor.dep.suggestionInstall"))
		*suggestions = append(*suggestions, i18n.T("cmd.doctor.dep.suggestionFix"))
	}

	if len(missingOptional) > 0 {
		*suggestions = append(*suggestions, fmt.Sprintf(i18n.T("cmd.doctor.dep.suggestionOptional"), strings.Join(missingOptional, ", ")))
	}

	return nil
}

// checkSystemResources checks system resources
func checkSystemResources(resourceChecker *system.ResourceChecker, allPassed *bool, issues *[]string, suggestions *[]string) error {
	// Define minimum requirements
	requirements := system.ResourceRequirement{
		MinDiskSpace: 100 * 1024 * 1024, // 100 MB
		MinMemory:    0,                 // Disable memory check for now
		RequiredDirs: []string{"."},
	}

	// Check current directory and common paths
	paths := []string{".", os.TempDir()}
	if homeDir, err := os.UserHomeDir(); err == nil {
		paths = append(paths, homeDir)
	}

	result := resourceChecker.CheckResourceRequirements(requirements, paths)

	if result.SystemInfo != nil {
		fmt.Printf("   💾 Memory: %.1f%% used\n", result.SystemInfo.Memory.UsedPct)
		for _, disk := range result.SystemInfo.DiskSpaces {
			fmt.Printf("   💿 Disk %s: %.1f%% used (%.2f GB available)\n",
				disk.Path, disk.UsedPct, float64(disk.Available)/(1024*1024*1024))
		}
	}

	if !result.Passed {
		*allPassed = false
		*issues = append(*issues, result.Errors...)
		*suggestions = append(*suggestions, result.Suggestions...)
	}

	if len(result.Warnings) > 0 {
		for _, warning := range result.Warnings {
			fmt.Printf("   ⚠️  %s\n", warning)
		}
	}

	return nil
}

// checkConfiguration checks configuration files
func checkConfiguration(allPassed *bool, issues *[]string, suggestions *[]string) error {
	logger := utils.GetGlobalLogger()
	configManager := system.NewConfigManager(logger)

	// Determine config file path
	configPath := cfgFile
	if configPath == "" {
		configPath = "apkhub.yaml"
	}

	// Validate configuration
	result := configManager.ValidateConfig(configPath)

	if result.Valid {
		fmt.Printf(i18n.T("cmd.doctor.config.valid")+"\n", configPath)

		// Show details if available
		if count, exists := result.Details["config_keys"]; exists {
			fmt.Printf(i18n.T("cmd.doctor.config.sections")+"\n", count)
		}
		if repoCount, exists := result.Details["repository_count"]; exists {
			fmt.Printf(i18n.T("cmd.doctor.config.repositories")+"\n", repoCount)
		}
	} else {
		fmt.Printf(i18n.T("cmd.doctor.config.invalid")+"\n", configPath)
		*allPassed = false

		// Add errors to issues
		for _, err := range result.Errors {
			*issues = append(*issues, fmt.Sprintf("Config error: %s", err))
		}

		// Add suggestions
		*suggestions = append(*suggestions, result.Suggestions...)
	}

	// Show warnings
	for _, warning := range result.Warnings {
		fmt.Printf(i18n.T("cmd.doctor.config.warning")+"\n", warning)
	}

	// Add warnings as suggestions if not already failing
	if result.Valid && len(result.Warnings) > 0 {
		*suggestions = append(*suggestions, "Address configuration warnings for better reliability")
	}

	return nil
}

// checkFileSystemAccess checks file system access permissions
func checkFileSystemAccess(resourceChecker *system.ResourceChecker, allPassed *bool, issues *[]string, suggestions *[]string) error {
	// Define permission checks
	checks := []system.PermissionCheck{
		{Path: ".", RequireRead: true, RequireWrite: true},
		{Path: os.TempDir(), RequireRead: true, RequireWrite: true},
	}

	// Add home directory if available
	if homeDir, err := os.UserHomeDir(); err == nil {
		checks = append(checks, system.PermissionCheck{
			Path: homeDir, RequireRead: true, RequireWrite: false,
		})
	}

	permIssues := resourceChecker.CheckPermissions(checks)

	if len(permIssues) == 0 {
		fmt.Println(i18n.T("cmd.doctor.fs.ok"))
	} else {
		*allPassed = false
		for _, issue := range permIssues {
			fmt.Printf(i18n.T("cmd.doctor.fs.issue")+"\n", issue)
			*issues = append(*issues, issue)
		}
		*suggestions = append(*suggestions, i18n.T("cmd.doctor.fs.suggestionPerm"))
		*suggestions = append(*suggestions, i18n.T("cmd.doctor.fs.suggestionAccess"))
	}

	return nil
}

// checkNetworkConnectivity checks network connectivity
func checkNetworkConnectivity(networkChecker *system.NetworkChecker, allPassed *bool, issues *[]string, suggestions *[]string) error {
	// Test basic connectivity
	basicStatus := networkChecker.CheckBasicConnectivity()

	if basicStatus.Connected {
		fmt.Printf(i18n.T("cmd.doctor.net.basic")+"\n",
			float64(basicStatus.Latency.Nanoseconds())/1000000)
		fmt.Printf(i18n.T("cmd.doctor.net.dns")+"\n", basicStatus.DNSWorking)
		fmt.Printf(i18n.T("cmd.doctor.net.https")+"\n", basicStatus.HTTPSWorking)
	} else {
		fmt.Printf(i18n.T("cmd.doctor.net.basicFail")+"\n", basicStatus.Error)
		*allPassed = false
		*issues = append(*issues, fmt.Sprintf(i18n.T("cmd.doctor.net.issue"), basicStatus.Error))

		// Add specific suggestions based on error type
		networkSuggestions := networkChecker.DiagnoseNetworkIssue(fmt.Errorf(basicStatus.Error))
		*suggestions = append(*suggestions, networkSuggestions...)

		return nil // Don't fail completely, just report the issue
	}

	// Test connectivity to common services
	tests := system.GetDefaultConnectivityTests()
	diagnostic := networkChecker.CheckConnectivity(tests)

	var failedTests []string
	for name, result := range diagnostic.Results {
		test := diagnostic.Tests[name]
		if result.Connected {
			fmt.Printf(i18n.T("cmd.doctor.net.testOK")+"\n", name,
				float64(result.Latency.Nanoseconds())/1000000)
		} else {
			status := "⚠️"
			if test.Required {
				status = "❌"
				*allPassed = false
			}
			fmt.Printf(i18n.T("cmd.doctor.net.testFail")+"\n", status, name, result.Error)
			failedTests = append(failedTests, name)
		}
	}

	if len(failedTests) > 0 {
		*issues = append(*issues, fmt.Sprintf(i18n.T("cmd.doctor.net.unreachable"), strings.Join(failedTests, ", ")))
		*suggestions = append(*suggestions, diagnostic.Suggestions...)
	}

	return nil
}

// attemptDoctorAutoFix attempts to automatically fix detected issues
func attemptDoctorAutoFix(depManager system.DependencyManager, issues []string, suggestions []string) error {
	fmt.Println(i18n.T("cmd.doctor.autofix.todo"))
	fmt.Println(i18n.T("cmd.doctor.autofix.manual"))

	// In a full implementation, this would:
	// 1. Try to install missing dependencies
	// 2. Create missing directories
	// 3. Fix common permission issues
	// 4. Generate default configuration files

	return nil
}

func init() {
	rootCmd.AddCommand(doctorCmd)

	doctorCmd.Flags().BoolVar(&doctorFix, "fix", false, i18n.T("cmd.doctor.flag.fix"))
	doctorCmd.Flags().BoolVar(&doctorVerbose, "verbose", false, i18n.T("cmd.doctor.flag.verbose"))
	doctorCmd.Flags().StringVar(&doctorCheck, "check", "basic", i18n.T("cmd.doctor.flag.check"))
}

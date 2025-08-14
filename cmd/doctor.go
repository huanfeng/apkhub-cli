package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/huanfeng/apkhub-cli/internal/errors"
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
	Short: "Diagnose and fix system issues",
	Long: `The doctor command performs comprehensive system diagnostics to identify
and optionally fix issues that might prevent ApkHub CLI from working properly.

It checks:
- Required dependencies (adb, aapt, aapt2)
- System resources (disk space, memory, permissions)
- Configuration files
- Network connectivity
- File system access`,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := utils.GetGlobalLogger()
		logger.Info("Starting system diagnostics...")

		fmt.Println("🏥 ApkHub CLI System Doctor")
		fmt.Println(strings.Repeat("=", 50))

		// Initialize checkers
		depManager := system.NewDependencyManager()
		resourceChecker := system.NewResourceChecker(logger)

		var allPassed = true
		var issues []string
		var suggestions []string

		// 1. Check dependencies
		fmt.Println("\n🔍 Checking Dependencies...")
		if err := checkDependencies(depManager, &allPassed, &issues, &suggestions); err != nil {
			logger.Error("Dependency check failed: %v", err)
		}

		// 2. Check system resources
		fmt.Println("\n💾 Checking System Resources...")
		if err := checkSystemResources(resourceChecker, &allPassed, &issues, &suggestions); err != nil {
			logger.Error("Resource check failed: %v", err)
		}

		// 3. Check configuration
		fmt.Println("\n⚙️  Checking Configuration...")
		if err := checkConfiguration(&allPassed, &issues, &suggestions); err != nil {
			logger.Error("Configuration check failed: %v", err)
		}

		// 4. Check file system access
		fmt.Println("\n📁 Checking File System Access...")
		if err := checkFileSystemAccess(resourceChecker, &allPassed, &issues, &suggestions); err != nil {
			logger.Error("File system check failed: %v", err)
		}

		// 5. Check network connectivity (if needed)
		if doctorCheck == "all" || doctorCheck == "network" {
			fmt.Println("\n🌐 Checking Network Connectivity...")
			networkChecker := system.NewNetworkChecker(logger)
			if err := checkNetworkConnectivity(networkChecker, &allPassed, &issues, &suggestions); err != nil {
				logger.Error("Network check failed: %v", err)
			}
		}

		// Display results
		fmt.Println("\n" + strings.Repeat("=", 50))
		fmt.Println("📊 DIAGNOSTIC RESULTS")
		fmt.Println(strings.Repeat("=", 50))

		if allPassed {
			fmt.Println("✅ All checks passed! Your system is ready to use ApkHub CLI.")
		} else {
			fmt.Printf("❌ Found %d issues that need attention:\n\n", len(issues))
			
			for i, issue := range issues {
				fmt.Printf("%d. %s\n", i+1, issue)
			}

			if len(suggestions) > 0 {
				fmt.Println("\n💡 Suggestions to fix these issues:")
				for i, suggestion := range suggestions {
					fmt.Printf("%d. %s\n", i+1, suggestion)
				}
			}

			if doctorFix {
				fmt.Println("\n🔧 Attempting to fix issues automatically...")
				if err := attemptDoctorAutoFix(depManager, issues, suggestions); err != nil {
					logger.Error("Auto-fix failed: %v", err)
					return errors.WrapError(err, errors.ErrorTypeConfiguration, "AUTO_FIX_FAILED", 
						"Failed to automatically fix issues").
						WithSuggestion("Try fixing the issues manually using the suggestions above")
				}
			} else {
				fmt.Println("\n💡 Run 'apkhub doctor --fix' to attempt automatic fixes")
			}
		}

		if !allPassed {
			return fmt.Errorf("system diagnostics found issues")
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
			fmt.Printf("   ✅ %s: %s\n", dep.Name, dep.Version)
		} else {
			if dep.Required {
				fmt.Printf("   ❌ %s: Not found (required)\n", dep.Name)
				missingRequired = append(missingRequired, dep.Name)
				*allPassed = false
			} else {
				fmt.Printf("   ⚠️  %s: Not found (optional)\n", dep.Name)
				missingOptional = append(missingOptional, dep.Name)
			}
		}
	}
	
	if len(missingRequired) > 0 {
		*issues = append(*issues, fmt.Sprintf("Missing required dependencies: %s", strings.Join(missingRequired, ", ")))
		*suggestions = append(*suggestions, "Install missing dependencies using your package manager")
		*suggestions = append(*suggestions, "Run 'apkhub doctor --fix' to attempt automatic installation")
	}
	
	if len(missingOptional) > 0 {
		*suggestions = append(*suggestions, fmt.Sprintf("Consider installing optional dependencies for better functionality: %s", strings.Join(missingOptional, ", ")))
	}
	
	return nil
}

// checkSystemResources checks system resources
func checkSystemResources(resourceChecker *system.ResourceChecker, allPassed *bool, issues *[]string, suggestions *[]string) error {
	// Define minimum requirements
	requirements := system.ResourceRequirement{
		MinDiskSpace: 100 * 1024 * 1024, // 100 MB
		MinMemory:    0,                  // Disable memory check for now
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
		fmt.Printf("   ✅ Configuration file: Valid (%s)\n", configPath)
		
		// Show details if available
		if count, exists := result.Details["config_keys"]; exists {
			fmt.Printf("   ℹ️  Configuration sections: %v\n", count)
		}
		if repoCount, exists := result.Details["repository_count"]; exists {
			fmt.Printf("   ℹ️  Configured repositories: %v\n", repoCount)
		}
	} else {
		fmt.Printf("   ❌ Configuration file: Invalid (%s)\n", configPath)
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
		fmt.Printf("   ⚠️  %s\n", warning)
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
		fmt.Printf("   ✅ File system access: OK\n")
	} else {
		*allPassed = false
		for _, issue := range permIssues {
			fmt.Printf("   ❌ %s\n", issue)
			*issues = append(*issues, issue)
		}
		*suggestions = append(*suggestions, "Fix file/directory permissions")
		*suggestions = append(*suggestions, "Ensure you have read/write access to working directory")
	}
	
	return nil
}

// checkNetworkConnectivity checks network connectivity
func checkNetworkConnectivity(networkChecker *system.NetworkChecker, allPassed *bool, issues *[]string, suggestions *[]string) error {
	// Test basic connectivity
	basicStatus := networkChecker.CheckBasicConnectivity()
	
	if basicStatus.Connected {
		fmt.Printf("   ✅ Basic connectivity: OK (%.2fms)\n", 
			float64(basicStatus.Latency.Nanoseconds())/1000000)
		fmt.Printf("   ✅ DNS resolution: %v\n", basicStatus.DNSWorking)
		fmt.Printf("   ✅ HTTPS connectivity: %v\n", basicStatus.HTTPSWorking)
	} else {
		fmt.Printf("   ❌ Basic connectivity: Failed - %s\n", basicStatus.Error)
		*allPassed = false
		*issues = append(*issues, fmt.Sprintf("Network connectivity failed: %s", basicStatus.Error))
		
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
			fmt.Printf("   ✅ %s: OK (%.2fms)\n", name, 
				float64(result.Latency.Nanoseconds())/1000000)
		} else {
			status := "⚠️"
			if test.Required {
				status = "❌"
				*allPassed = false
			}
			fmt.Printf("   %s %s: Failed - %s\n", status, name, result.Error)
			failedTests = append(failedTests, name)
		}
	}
	
	if len(failedTests) > 0 {
		*issues = append(*issues, fmt.Sprintf("Some network services unreachable: %s", 
			strings.Join(failedTests, ", ")))
		*suggestions = append(*suggestions, diagnostic.Suggestions...)
	}
	
	return nil
}

// attemptDoctorAutoFix attempts to automatically fix detected issues
func attemptDoctorAutoFix(depManager system.DependencyManager, issues []string, suggestions []string) error {
	fmt.Println("🔧 Auto-fix is not fully implemented yet")
	fmt.Println("   Please follow the manual suggestions provided above")
	
	// In a full implementation, this would:
	// 1. Try to install missing dependencies
	// 2. Create missing directories
	// 3. Fix common permission issues
	// 4. Generate default configuration files
	
	return nil
}

func init() {
	rootCmd.AddCommand(doctorCmd)
	
	doctorCmd.Flags().BoolVar(&doctorFix, "fix", false, "Attempt to automatically fix detected issues")
	doctorCmd.Flags().BoolVar(&doctorVerbose, "verbose", false, "Show detailed diagnostic information")
	doctorCmd.Flags().StringVar(&doctorCheck, "check", "basic", "Type of check to perform: basic, all, network")
}
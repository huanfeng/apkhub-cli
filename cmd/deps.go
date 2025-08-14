package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/huanfeng/apkhub-cli/pkg/system"
	"github.com/spf13/cobra"
)

var (
	depsCommand string
	depsJSON    bool
	depsCache   bool
)

var depsCmd = &cobra.Command{
	Use:   "deps",
	Short: "Check dependencies for ApkHub CLI",
	Long:  `Check system dependencies required for ApkHub CLI commands and show installation guidance.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		depManager := system.NewDependencyManager()
		installer := system.NewInstallationGuide()

		if depsCache {
			depManager.ClearCache()
			fmt.Println("✅ Dependency cache cleared")
			return nil
		}

		if depsCommand != "" {
			return checkCommandDependencies(depManager, installer, depsCommand)
		}

		return checkAllDependencies(depManager, installer)
	},
}

func checkCommandDependencies(depManager system.DependencyManager, installer system.InstallationGuide, command string) error {
	fmt.Printf("🔍 Checking dependencies for command: %s\n", command)
	fmt.Println("=" + strings.Repeat("=", len(command)+35))
	fmt.Println()

	deps := depManager.CheckForCommand(command)

	if len(deps) == 0 {
		fmt.Printf("✅ Command '%s' has no external dependencies\n", command)
		return nil
	}

	if depsJSON {
		return outputJSON(deps)
	}

	// Display dependencies
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "DEPENDENCY\tSTATUS\tVERSION\tPATH\tREQUIRED")
	fmt.Fprintln(w, "----------\t------\t-------\t----\t--------")

	var missingRequired []string
	var missingOptional []string

	for _, dep := range deps {
		status := "❌ Missing"
		if dep.Available {
			status = "✅ Available"
		}

		required := "No"
		if dep.Required {
			required = "Yes"
			if !dep.Available {
				missingRequired = append(missingRequired, dep.Name)
			}
		} else if !dep.Available {
			missingOptional = append(missingOptional, dep.Name)
		}

		path := dep.Path
		if len(path) > 30 {
			path = "..." + path[len(path)-27:]
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			dep.Name, status, dep.Version, path, required)
	}

	w.Flush()

	// Show installation guidance
	if len(missingRequired) > 0 || len(missingOptional) > 0 {
		fmt.Println("\n📋 Installation Guidance:")
		fmt.Println("========================")

		allMissing := append(missingRequired, missingOptional...)
		for _, depName := range allMissing {
			fmt.Printf("\n🔧 %s:\n", depName)

			steps := installer.GetPlatformInstructions(depName)
			for i, step := range steps {
				if step.Manual {
					fmt.Printf("   %d. %s (Manual)\n", i+1, step.Description)
				} else {
					fmt.Printf("   %d. %s\n", i+1, step.Description)
					if step.Command != "" {
						fmt.Printf("      Command: %s\n", step.Command)
					}
				}
			}
		}

		if len(missingRequired) > 0 {
			fmt.Printf("\n❌ Command '%s' cannot run without: %s\n", command, strings.Join(missingRequired, ", "))
		}
		if len(missingOptional) > 0 {
			fmt.Printf("\n⚠️  Command '%s' will have limited functionality without: %s\n", command, strings.Join(missingOptional, ", "))
		}
	} else {
		fmt.Printf("\n✅ All dependencies for '%s' are satisfied!\n", command)
	}

	return nil
}

func checkAllDependencies(depManager system.DependencyManager, installer system.InstallationGuide) error {
	fmt.Println("🔍 Checking all ApkHub CLI dependencies")
	fmt.Println("======================================")
	fmt.Println()

	allDeps := depManager.CheckAll()

	if depsJSON {
		return outputJSON(allDeps)
	}

	// Display all dependencies
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "DEPENDENCY\tSTATUS\tVERSION\tPATH\tUSED BY")
	fmt.Fprintln(w, "----------\t------\t-------\t----\t-------")

	var missingRequired []string
	var missingOptional []string

	for _, dep := range allDeps {
		status := "❌ Missing"
		if dep.Available {
			status = "✅ Available"
		}

		if dep.Required && !dep.Available {
			missingRequired = append(missingRequired, dep.Name)
		} else if !dep.Required && !dep.Available {
			missingOptional = append(missingOptional, dep.Name)
		}

		usedBy := strings.Join(dep.UsedBy, ", ")
		if len(usedBy) > 40 {
			usedBy = usedBy[:37] + "..."
		}

		path := dep.Path
		if len(path) > 30 {
			path = "..." + path[len(path)-27:]
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			dep.Name, status, dep.Version, path, usedBy)
	}

	w.Flush()

	// Summary
	fmt.Println("\n📊 Summary:")
	fmt.Println("===========")

	available := 0
	total := len(allDeps)
	for _, dep := range allDeps {
		if dep.Available {
			available++
		}
	}

	fmt.Printf("Available: %d/%d dependencies\n", available, total)

	if len(missingRequired) > 0 {
		fmt.Printf("❌ Missing required: %s\n", strings.Join(missingRequired, ", "))
	}
	if len(missingOptional) > 0 {
		fmt.Printf("⚠️  Missing optional: %s\n", strings.Join(missingOptional, ", "))
	}

	if len(missingRequired) == 0 && len(missingOptional) == 0 {
		fmt.Println("✅ All dependencies are satisfied!")
	} else {
		fmt.Println("\n💡 Run 'apkhub doctor --fix' to install missing dependencies")
	}

	return nil
}

func outputJSON(data interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

func init() {
	rootCmd.AddCommand(depsCmd)

	depsCmd.Flags().StringVarP(&depsCommand, "command", "c", "", "Check dependencies for specific command")
	depsCmd.Flags().BoolVar(&depsJSON, "json", false, "Output in JSON format")
	depsCmd.Flags().BoolVar(&depsCache, "clear-cache", false, "Clear dependency cache")
}

package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/huanfeng/apkhub-cli/internal/i18n"
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
	Short: i18n.T("cmd.deps.short"),
	Long:  i18n.T("cmd.deps.long"),
	RunE: func(cmd *cobra.Command, args []string) error {
		depManager := system.NewDependencyManager()
		installer := system.NewInstallationGuide()

		if depsCache {
			depManager.ClearCache()
			fmt.Println(i18n.T("cmd.deps.cacheCleared"))
			return nil
		}

		if depsCommand != "" {
			return checkCommandDependencies(depManager, installer, depsCommand)
		}

		return checkAllDependencies(depManager, installer)
	},
}

func checkCommandDependencies(depManager system.DependencyManager, installer system.InstallationGuide, command string) error {
	fmt.Printf("%s\n", i18n.T("cmd.deps.checkCommand", map[string]interface{}{"command": command}))
	fmt.Println("=" + strings.Repeat("=", len(command)+35))
	fmt.Println()

	deps := depManager.CheckForCommand(command)

	if len(deps) == 0 {
		fmt.Printf("%s\n", i18n.T("cmd.deps.noDeps", map[string]interface{}{"command": command}))
		return nil
	}

	if depsJSON {
		return outputJSON(deps)
	}

	// Display dependencies
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, i18n.T("cmd.deps.table.command.header"))
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
		fmt.Println()
		fmt.Println(i18n.T("cmd.deps.install.title"))
		fmt.Println("========================")

		allMissing := append(missingRequired, missingOptional...)
		for _, depName := range allMissing {
			fmt.Printf("\n🔧 %s:\n", depName)
			fmt.Printf("%s\n", i18n.T("cmd.deps.install.for", map[string]interface{}{"name": depName}))

			steps := installer.GetPlatformInstructions(depName)
			for i, step := range steps {
				if step.Manual {
					fmt.Printf(i18n.T("cmd.deps.install.manual")+"\n", i+1, step.Description)
				} else {
					fmt.Printf(i18n.T("cmd.deps.install.step")+"\n", i+1, step.Description)
					if step.Command != "" {
						fmt.Printf(i18n.T("cmd.deps.install.command")+"\n", step.Command)
					}
				}
			}
		}

		if len(missingRequired) > 0 {
			fmt.Printf("\n%s\n", i18n.T("cmd.deps.install.missingRequired", map[string]interface{}{
				"command": command, "deps": strings.Join(missingRequired, ", "),
			}))
		}
		if len(missingOptional) > 0 {
			fmt.Printf("\n%s\n", i18n.T("cmd.deps.install.missingOptional", map[string]interface{}{
				"command": command, "deps": strings.Join(missingOptional, ", "),
			}))
		}
	} else {
		fmt.Printf("\n%s\n", i18n.T("cmd.deps.install.allGood", map[string]interface{}{"command": command}))
	}

	return nil
}

func checkAllDependencies(depManager system.DependencyManager, installer system.InstallationGuide) error {
	fmt.Println(i18n.T("cmd.deps.checkAll.title"))
	fmt.Println("======================================")
	fmt.Println()

	allDeps := depManager.CheckAll()

	if depsJSON {
		return outputJSON(allDeps)
	}

	// Display all dependencies
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, i18n.T("cmd.deps.table.all.header"))
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
	fmt.Println("\n" + i18n.T("cmd.deps.summary.title"))
	fmt.Println("===========")

	available := 0
	total := len(allDeps)
	for _, dep := range allDeps {
		if dep.Available {
			available++
		}
	}

	fmt.Printf("%s\n", i18n.T("cmd.deps.summary.available", map[string]interface{}{
		"available": available, "total": total,
	}))

	if len(missingRequired) > 0 {
		fmt.Printf("%s\n", i18n.T("cmd.deps.summary.missingRequired", map[string]interface{}{
			"deps": strings.Join(missingRequired, ", "),
		}))
	}
	if len(missingOptional) > 0 {
		fmt.Printf("%s\n", i18n.T("cmd.deps.summary.missingOptional", map[string]interface{}{
			"deps": strings.Join(missingOptional, ", "),
		}))
	}

	if len(missingRequired) == 0 && len(missingOptional) == 0 {
		fmt.Println(i18n.T("cmd.deps.summary.allGood"))
	} else {
		fmt.Println()
		fmt.Println(i18n.T("cmd.deps.summary.suggestion"))
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

	depsCmd.Flags().StringVarP(&depsCommand, "command", "c", "", i18n.T("cmd.deps.flag.command"))
	depsCmd.Flags().BoolVar(&depsJSON, "json", false, i18n.T("cmd.deps.flag.json"))
	depsCmd.Flags().BoolVar(&depsCache, "clear-cache", false, i18n.T("cmd.deps.flag.clearCache"))
}

package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var (
	doctorFix bool
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check system dependencies and environment",
	Long:  `Check system dependencies required for ApkHub CLI and provide installation instructions.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("🔍 ApkHub CLI Environment Check")
		fmt.Println("================================")
		fmt.Println()

		// Check dependencies
		deps := checkDependencies()
		
		// Display results
		displayDependencyResults(deps)
		
		// Provide recommendations
		provideRecommendations(deps)
		
		// Auto-fix if requested
		if doctorFix {
			fmt.Println("\n🔧 Attempting to fix issues...")
			return autoFixDependencies(deps)
		}
		
		return nil
	},
}

type DependencyCheck struct {
	Name        string
	Required    bool
	Available   bool
	Version     string
	Path        string
	UsedBy      []string
	InstallHint string
	Status      string
}

func checkDependencies() []DependencyCheck {
	var deps []DependencyCheck
	
	// Check aapt2
	aapt2 := DependencyCheck{
		Name:     "aapt2",
		Required: false, // Recommended but not required
		UsedBy:   []string{"repo scan", "repo add", "repo parse", "info"},
		InstallHint: getAAPTInstallHint(),
	}
	
	if path, version, available := checkAAPT("aapt2"); available {
		aapt2.Available = true
		aapt2.Version = version
		aapt2.Path = path
		aapt2.Status = "✅ Available"
	} else {
		aapt2.Status = "❌ Not found"
	}
	
	deps = append(deps, aapt2)
	
	// Check aapt (fallback)
	aapt := DependencyCheck{
		Name:     "aapt",
		Required: false,
		UsedBy:   []string{"repo scan (fallback)", "repo add (fallback)"},
		InstallHint: getAAPTInstallHint(),
	}
	
	if path, version, available := checkAAPT("aapt"); available {
		aapt.Available = true
		aapt.Version = version
		aapt.Path = path
		aapt.Status = "✅ Available"
	} else {
		aapt.Status = "❌ Not found"
	}
	
	deps = append(deps, aapt)
	
	// Check adb
	adb := DependencyCheck{
		Name:     "adb",
		Required: true, // Required for install functionality
		UsedBy:   []string{"install"},
		InstallHint: getADBInstallHint(),
	}
	
	if path, version, available := checkADB(); available {
		adb.Available = true
		adb.Version = version
		adb.Path = path
		adb.Status = "✅ Available"
	} else {
		adb.Status = "❌ Not found"
	}
	
	deps = append(deps, adb)
	
	return deps
}

func checkAAPT(toolName string) (string, string, bool) {
	// Try PATH first
	if path, err := exec.LookPath(toolName); err == nil {
		if version := getToolVersion(path, "version"); version != "" {
			return path, version, true
		}
	}
	
	// Try common paths
	commonPaths := getCommonAAPTPaths(toolName)
	for _, path := range commonPaths {
		if _, err := os.Stat(path); err == nil {
			if version := getToolVersion(path, "version"); version != "" {
				return path, version, true
			}
		}
	}
	
	return "", "", false
}

func checkADB() (string, string, bool) {
	// Try PATH first
	if path, err := exec.LookPath("adb"); err == nil {
		if version := getToolVersion(path, "version"); version != "" {
			return path, version, true
		}
	}
	
	// Try common paths
	commonPaths := getCommonADBPaths()
	for _, path := range commonPaths {
		if _, err := os.Stat(path); err == nil {
			if version := getToolVersion(path, "version"); version != "" {
				return path, version, true
			}
		}
	}
	
	return "", "", false
}

func getToolVersion(toolPath, versionArg string) string {
	cmd := exec.Command(toolPath, versionArg)
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	
	// Extract version from output
	lines := strings.Split(string(output), "\n")
	if len(lines) > 0 {
		return strings.TrimSpace(lines[0])
	}
	
	return "unknown"
}

func getCommonAAPTPaths(toolName string) []string {
	var paths []string
	
	switch runtime.GOOS {
	case "linux":
		paths = []string{
			"/usr/bin/" + toolName,
			"/usr/local/bin/" + toolName,
			"/opt/android-sdk/build-tools/*/aapt2",
			"/opt/android-sdk/build-tools/*/aapt",
		}
		
		// Add user-specific paths
		if home := os.Getenv("HOME"); home != "" {
			paths = append(paths, 
				filepath.Join(home, "Android/Sdk/build-tools/*/"+toolName),
				filepath.Join(home, ".android-sdk/build-tools/*/"+toolName),
			)
		}
		
	case "darwin":
		paths = []string{
			"/usr/local/bin/" + toolName,
			"/opt/homebrew/bin/" + toolName,
		}
		
		if home := os.Getenv("HOME"); home != "" {
			paths = append(paths,
				filepath.Join(home, "Library/Android/sdk/build-tools/*/"+toolName),
				filepath.Join(home, ".android-sdk/build-tools/*/"+toolName),
			)
		}
		
	case "windows":
		paths = []string{
			"C:\\Android\\Sdk\\build-tools\\*\\" + toolName + ".exe",
		}
		
		if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
			paths = append(paths, filepath.Join(localAppData, "Android\\Sdk\\build-tools\\*\\"+toolName+".exe"))
		}
		
		if programFiles := os.Getenv("PROGRAMFILES"); programFiles != "" {
			paths = append(paths, filepath.Join(programFiles, "Android\\Android Studio\\plugins\\android\\lib\\build-tools\\*\\"+toolName+".exe"))
		}
	}
	
	return paths
}

func getCommonADBPaths() []string {
	var paths []string
	
	switch runtime.GOOS {
	case "linux":
		paths = []string{
			"/usr/bin/adb",
			"/usr/local/bin/adb",
			"/opt/android-sdk/platform-tools/adb",
		}
		
		if home := os.Getenv("HOME"); home != "" {
			paths = append(paths,
				filepath.Join(home, "Android/Sdk/platform-tools/adb"),
				filepath.Join(home, ".android-sdk/platform-tools/adb"),
			)
		}
		
	case "darwin":
		paths = []string{
			"/usr/local/bin/adb",
			"/opt/homebrew/bin/adb",
		}
		
		if home := os.Getenv("HOME"); home != "" {
			paths = append(paths,
				filepath.Join(home, "Library/Android/sdk/platform-tools/adb"),
			)
		}
		
	case "windows":
		paths = []string{
			"C:\\Android\\Sdk\\platform-tools\\adb.exe",
		}
		
		if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
			paths = append(paths, filepath.Join(localAppData, "Android\\Sdk\\platform-tools\\adb.exe"))
		}
	}
	
	return paths
}

func displayDependencyResults(deps []DependencyCheck) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "TOOL\tSTATUS\tVERSION\tPATH\tUSED BY")
	fmt.Fprintln(w, "----\t------\t-------\t----\t-------")
	
	for _, dep := range deps {
		usedBy := strings.Join(dep.UsedBy, ", ")
		if len(usedBy) > 40 {
			usedBy = usedBy[:37] + "..."
		}
		
		path := dep.Path
		if len(path) > 30 {
			path = "..." + path[len(path)-27:]
		}
		
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			dep.Name, dep.Status, dep.Version, path, usedBy)
	}
	
	w.Flush()
}

func provideRecommendations(deps []DependencyCheck) {
	fmt.Println("\n📋 Recommendations:")
	fmt.Println("===================")
	
	hasIssues := false
	
	// Check for missing required dependencies
	for _, dep := range deps {
		if dep.Required && !dep.Available {
			fmt.Printf("\n❌ %s is REQUIRED but not found\n", dep.Name)
			fmt.Printf("   Used by: %s\n", strings.Join(dep.UsedBy, ", "))
			fmt.Printf("   Install: %s\n", dep.InstallHint)
			hasIssues = true
		}
	}
	
	// Check for missing recommended dependencies
	aaptAvailable := false
	for _, dep := range deps {
		if (dep.Name == "aapt2" || dep.Name == "aapt") && dep.Available {
			aaptAvailable = true
			break
		}
	}
	
	if !aaptAvailable {
		fmt.Printf("\n⚠️  Neither aapt2 nor aapt is available\n")
		fmt.Printf("   Impact: APK parsing will be limited, some APKs may fail to parse\n")
		fmt.Printf("   Recommendation: Install aapt2 for better APK parsing support\n")
		fmt.Printf("   Install: %s\n", getAAPTInstallHint())
		hasIssues = true
	}
	
	if !hasIssues {
		fmt.Println("\n✅ All dependencies are properly configured!")
		fmt.Println("   Your ApkHub CLI installation is ready to use.")
	} else {
		fmt.Printf("\n💡 Run 'apkhub doctor --fix' to attempt automatic fixes\n")
	}
}

func getAAPTInstallHint() string {
	switch runtime.GOOS {
	case "linux":
		return "sudo apt-get install aapt (Ubuntu/Debian) or brew install android-commandlinetools (Homebrew) or install Android SDK Build Tools"
	case "darwin":
		return "brew install --cask android-commandlinetools"
	case "windows":
		return "Install Android SDK Build Tools from https://developer.android.com/studio#command-tools"
	default:
		return "Install Android SDK Build Tools"
	}
}

func getADBInstallHint() string {
	switch runtime.GOOS {
	case "linux":
		return "sudo apt-get install adb (Ubuntu/Debian) or brew install android-platform-tools (Homebrew) or install Android SDK Platform Tools"
	case "darwin":
		return "brew install android-platform-tools"
	case "windows":
		return "Install Android SDK Platform Tools"
	default:
		return "Install Android SDK Platform Tools"
	}
}

func autoFixDependencies(deps []DependencyCheck) error {
	switch runtime.GOOS {
	case "linux":
		return autoFixLinux(deps)
	case "darwin":
		return autoFixMacOS(deps)
	case "windows":
		return autoFixWindows(deps)
	default:
		fmt.Println("🚧 Auto-fix is not supported on this platform")
		fmt.Println("Please follow the manual installation instructions above.")
		return nil
	}
}

func autoFixLinux(deps []DependencyCheck) error {
	fmt.Println("🐧 Attempting to fix dependencies on Linux...")
	
	// Detect available package managers
	var availableManagers []string
	
	if _, err := exec.LookPath("apt-get"); err == nil {
		availableManagers = append(availableManagers, "apt")
	}
	
	if _, err := exec.LookPath("brew"); err == nil {
		availableManagers = append(availableManagers, "brew")
	}
	
	if _, err := exec.LookPath("yum"); err == nil {
		availableManagers = append(availableManagers, "yum")
	}
	
	if _, err := exec.LookPath("dnf"); err == nil {
		availableManagers = append(availableManagers, "dnf")
	}
	
	if _, err := exec.LookPath("pacman"); err == nil {
		availableManagers = append(availableManagers, "pacman")
	}
	
	if len(availableManagers) == 0 {
		fmt.Println("❌ No supported package manager found")
		fmt.Println("Please install dependencies manually")
		return nil
	}
	
	// If only one manager is available, use it directly
	if len(availableManagers) == 1 {
		return autoFixWithPackageManager(availableManagers[0], deps)
	}
	
	// Multiple managers available, let user choose
	fmt.Printf("📦 Multiple package managers detected: %s\n", strings.Join(availableManagers, ", "))
	fmt.Println("\nWhich package manager would you like to use?")
	
	for i, manager := range availableManagers {
		fmt.Printf("  %d) %s %s\n", i+1, manager, getPackageManagerDescription(manager))
	}
	
	fmt.Print("\nEnter your choice (1-" + fmt.Sprintf("%d", len(availableManagers)) + "): ")
	
	var choice int
	if _, err := fmt.Scanln(&choice); err != nil || choice < 1 || choice > len(availableManagers) {
		fmt.Println("❌ Invalid choice")
		return nil
	}
	
	selectedManager := availableManagers[choice-1]
	return autoFixWithPackageManager(selectedManager, deps)
}

func getPackageManagerDescription(manager string) string {
	switch manager {
	case "apt":
		return "(Ubuntu/Debian - system packages, may be older versions)"
	case "brew":
		return "(Homebrew - usually more up-to-date versions)"
	case "yum":
		return "(RHEL/CentOS - system packages)"
	case "dnf":
		return "(Fedora - system packages)"
	case "pacman":
		return "(Arch Linux - system packages)"
	default:
		return ""
	}
}

func autoFixWithPackageManager(manager string, deps []DependencyCheck) error {
	switch manager {
	case "apt":
		return autoFixWithApt(deps)
	case "brew":
		return autoFixWithBrewLinux(deps)
	case "yum":
		return autoFixWithYum(deps)
	case "dnf":
		return autoFixWithDnf(deps)
	case "pacman":
		return autoFixWithPacman(deps)
	default:
		fmt.Printf("❌ Package manager %s is not supported\n", manager)
		return nil
	}
}

func autoFixWithApt(deps []DependencyCheck) error {
	fmt.Println("📦 Detected APT package manager")
	
	var packagesToInstall []string
	
	for _, dep := range deps {
		if !dep.Available {
			switch dep.Name {
			case "adb":
				packagesToInstall = append(packagesToInstall, "adb")
			case "aapt2", "aapt":
				if !contains(packagesToInstall, "aapt") {
					packagesToInstall = append(packagesToInstall, "aapt")
				}
			}
		}
	}
	
	if len(packagesToInstall) == 0 {
		fmt.Println("✅ No packages need to be installed")
		return nil
	}
	
	fmt.Printf("📥 Will install: %s\n", strings.Join(packagesToInstall, ", "))
	fmt.Print("Continue? [y/N]: ")
	
	var response string
	fmt.Scanln(&response)
	if strings.ToLower(response) != "y" {
		fmt.Println("❌ Installation cancelled")
		return nil
	}
	
	// Install packages
	for _, pkg := range packagesToInstall {
		fmt.Printf("📦 Installing %s...\n", pkg)
		cmd := exec.Command("sudo", "apt-get", "install", "-y", pkg)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		
		if err := cmd.Run(); err != nil {
			fmt.Printf("❌ Failed to install %s: %v\n", pkg, err)
			continue
		}
		
		fmt.Printf("✅ Successfully installed %s\n", pkg)
	}
	
	return nil
}

func autoFixMacOS(deps []DependencyCheck) error {
	fmt.Println("🍎 Attempting to fix dependencies on macOS...")
	
	// Check if brew is available
	if _, err := exec.LookPath("brew"); err == nil {
		return autoFixWithBrew(deps)
	}
	
	fmt.Println("❌ Homebrew not found")
	fmt.Println("Please install Homebrew first: https://brew.sh")
	return nil
}

func autoFixWithBrew(deps []DependencyCheck) error {
	fmt.Println("🍺 Detected Homebrew package manager")
	
	var packagesToInstall []string
	
	for _, dep := range deps {
		if !dep.Available {
			switch dep.Name {
			case "adb":
				packagesToInstall = append(packagesToInstall, "android-platform-tools")
			case "aapt2", "aapt":
				if !contains(packagesToInstall, "android-commandlinetools") {
					packagesToInstall = append(packagesToInstall, "--cask android-commandlinetools")
				}
			}
		}
	}
	
	if len(packagesToInstall) == 0 {
		fmt.Println("✅ No packages need to be installed")
		return nil
	}
	
	fmt.Printf("📥 Will install: %s\n", strings.Join(packagesToInstall, ", "))
	fmt.Print("Continue? [y/N]: ")
	
	var response string
	fmt.Scanln(&response)
	if strings.ToLower(response) != "y" {
		fmt.Println("❌ Installation cancelled")
		return nil
	}
	
	// Install packages
	for _, pkg := range packagesToInstall {
		fmt.Printf("📦 Installing %s...\n", pkg)
		
		var cmd *exec.Cmd
		if strings.Contains(pkg, "--cask") {
			parts := strings.Split(pkg, " ")
			cmd = exec.Command("brew", "install", parts[0], parts[1])
		} else {
			cmd = exec.Command("brew", "install", pkg)
		}
		
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		
		if err := cmd.Run(); err != nil {
			fmt.Printf("❌ Failed to install %s: %v\n", pkg, err)
			continue
		}
		
		fmt.Printf("✅ Successfully installed %s\n", pkg)
	}
	
	return nil
}

func autoFixWindows(deps []DependencyCheck) error {
	fmt.Println("🪟 Auto-fix for Windows is not yet implemented")
	fmt.Println("Please install dependencies manually:")
	fmt.Println("1. Download Android SDK Command Line Tools")
	fmt.Println("2. Extract to C:\\Android\\Sdk\\")
	fmt.Println("3. Add platform-tools and build-tools to PATH")
	return nil
}

func autoFixWithBrewLinux(deps []DependencyCheck) error {
	fmt.Println("🍺 Using Homebrew on Linux")
	
	// First ensure homebrew/cask is tapped
	fmt.Println("📦 Ensuring homebrew/cask is available...")
	tapCmd := exec.Command("brew", "tap", "homebrew/cask")
	tapCmd.Run() // Ignore errors if already tapped
	
	var packagesToInstall []string
	var caskPackages []string
	
	for _, dep := range deps {
		if !dep.Available {
			switch dep.Name {
			case "adb":
				caskPackages = append(caskPackages, "android-platform-tools")
			case "aapt2", "aapt":
				if !contains(caskPackages, "android-commandlinetools") {
					caskPackages = append(caskPackages, "android-commandlinetools")
				}
			}
		}
	}
	
	if len(packagesToInstall) == 0 && len(caskPackages) == 0 {
		fmt.Println("✅ No packages need to be installed")
		return nil
	}
	
	allPackages := append(packagesToInstall, caskPackages...)
	fmt.Printf("📥 Will install: %s\n", strings.Join(allPackages, ", "))
	fmt.Println("💡 Note: Homebrew packages are usually more up-to-date than system packages")
	fmt.Print("Continue? [y/N]: ")
	
	var response string
	fmt.Scanln(&response)
	if strings.ToLower(response) != "y" {
		fmt.Println("❌ Installation cancelled")
		return nil
	}
	
	// Install regular packages
	for _, pkg := range packagesToInstall {
		fmt.Printf("📦 Installing %s...\n", pkg)
		cmd := exec.Command("brew", "install", pkg)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		
		if err := cmd.Run(); err != nil {
			fmt.Printf("❌ Failed to install %s: %v\n", pkg, err)
			continue
		}
		
		fmt.Printf("✅ Successfully installed %s\n", pkg)
	}
	
	// Install cask packages
	for _, pkg := range caskPackages {
		fmt.Printf("📦 Installing %s (cask)...\n", pkg)
		
		// Try cask first
		cmd := exec.Command("brew", "install", "--cask", pkg)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		
		if err := cmd.Run(); err != nil {
			fmt.Printf("⚠️  Cask installation failed, trying regular formula...\n")
			
			// Try regular formula
			cmd = exec.Command("brew", "install", pkg)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			
			if err := cmd.Run(); err != nil {
				fmt.Printf("❌ Failed to install %s: %v\n", pkg, err)
				
				// Provide specific guidance for common issues
				if pkg == "android-commandlinetools" {
					fmt.Printf("💡 Alternative installation methods for Android SDK:\n")
					fmt.Printf("   1. Manual download: https://developer.android.com/studio#command-tools\n")
					fmt.Printf("   2. Try APT: sudo apt-get install aapt\n")
					fmt.Printf("   3. Install Android Studio which includes these tools\n")
				} else {
					fmt.Printf("💡 Manual installation may be required\n")
				}
				continue
			}
		}
		
		fmt.Printf("✅ Successfully installed %s\n", pkg)
	}
	
	return nil
}

func autoFixWithYum(deps []DependencyCheck) error {
	fmt.Println("📦 Using YUM package manager")
	
	var packagesToInstall []string
	
	for _, dep := range deps {
		if !dep.Available {
			switch dep.Name {
			case "adb":
				packagesToInstall = append(packagesToInstall, "android-tools")
			case "aapt2", "aapt":
				fmt.Println("⚠️  aapt/aapt2 may not be available in YUM repositories")
				fmt.Println("💡 Consider installing Android SDK manually or using Homebrew")
			}
		}
	}
	
	if len(packagesToInstall) == 0 {
		fmt.Println("✅ No packages available for installation via YUM")
		return nil
	}
	
	fmt.Printf("📥 Will install: %s\n", strings.Join(packagesToInstall, ", "))
	fmt.Print("Continue? [y/N]: ")
	
	var response string
	fmt.Scanln(&response)
	if strings.ToLower(response) != "y" {
		fmt.Println("❌ Installation cancelled")
		return nil
	}
	
	// Install packages
	for _, pkg := range packagesToInstall {
		fmt.Printf("📦 Installing %s...\n", pkg)
		cmd := exec.Command("sudo", "yum", "install", "-y", pkg)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		
		if err := cmd.Run(); err != nil {
			fmt.Printf("❌ Failed to install %s: %v\n", pkg, err)
			continue
		}
		
		fmt.Printf("✅ Successfully installed %s\n", pkg)
	}
	
	return nil
}

func autoFixWithDnf(deps []DependencyCheck) error {
	fmt.Println("📦 Using DNF package manager")
	
	var packagesToInstall []string
	
	for _, dep := range deps {
		if !dep.Available {
			switch dep.Name {
			case "adb":
				packagesToInstall = append(packagesToInstall, "android-tools")
			case "aapt2", "aapt":
				fmt.Println("⚠️  aapt/aapt2 may not be available in DNF repositories")
				fmt.Println("💡 Consider installing Android SDK manually or using Homebrew")
			}
		}
	}
	
	if len(packagesToInstall) == 0 {
		fmt.Println("✅ No packages available for installation via DNF")
		return nil
	}
	
	fmt.Printf("📥 Will install: %s\n", strings.Join(packagesToInstall, ", "))
	fmt.Print("Continue? [y/N]: ")
	
	var response string
	fmt.Scanln(&response)
	if strings.ToLower(response) != "y" {
		fmt.Println("❌ Installation cancelled")
		return nil
	}
	
	// Install packages
	for _, pkg := range packagesToInstall {
		fmt.Printf("📦 Installing %s...\n", pkg)
		cmd := exec.Command("sudo", "dnf", "install", "-y", pkg)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		
		if err := cmd.Run(); err != nil {
			fmt.Printf("❌ Failed to install %s: %v\n", pkg, err)
			continue
		}
		
		fmt.Printf("✅ Successfully installed %s\n", pkg)
	}
	
	return nil
}

func autoFixWithPacman(deps []DependencyCheck) error {
	fmt.Println("📦 Using Pacman package manager")
	
	var packagesToInstall []string
	
	for _, dep := range deps {
		if !dep.Available {
			switch dep.Name {
			case "adb":
				packagesToInstall = append(packagesToInstall, "android-tools")
			case "aapt2", "aapt":
				packagesToInstall = append(packagesToInstall, "android-sdk-build-tools")
			}
		}
	}
	
	if len(packagesToInstall) == 0 {
		fmt.Println("✅ No packages need to be installed")
		return nil
	}
	
	fmt.Printf("📥 Will install: %s\n", strings.Join(packagesToInstall, ", "))
	fmt.Println("💡 Note: Some packages may be available in AUR")
	fmt.Print("Continue? [y/N]: ")
	
	var response string
	fmt.Scanln(&response)
	if strings.ToLower(response) != "y" {
		fmt.Println("❌ Installation cancelled")
		return nil
	}
	
	// Install packages
	for _, pkg := range packagesToInstall {
		fmt.Printf("📦 Installing %s...\n", pkg)
		cmd := exec.Command("sudo", "pacman", "-S", "--noconfirm", pkg)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		
		if err := cmd.Run(); err != nil {
			fmt.Printf("❌ Failed to install %s: %v\n", pkg, err)
			fmt.Printf("💡 Try installing from AUR: yay -S %s\n", pkg)
			continue
		}
		
		fmt.Printf("✅ Successfully installed %s\n", pkg)
	}
	
	return nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func init() {
	rootCmd.AddCommand(doctorCmd)
	
	doctorCmd.Flags().BoolVar(&doctorFix, "fix", false, "Attempt to automatically fix dependency issues")
}
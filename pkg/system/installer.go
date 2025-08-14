package system

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// InstallStep represents a single installation step
type InstallStep struct {
	Description string `json:"description"`
	Command     string `json:"command,omitempty"`
	Manual      bool   `json:"manual"`
	Platform    string `json:"platform"`
}

// InstallationGuide provides installation guidance for dependencies
type InstallationGuide interface {
	GetPlatformInstructions(tool string) []InstallStep
	CanAutoInstall(tool string) bool
	AutoInstall(tool string) error
	DetectPackageManagers() []string
}

// DefaultInstallationGuide is the default implementation
type DefaultInstallationGuide struct{}

// NewInstallationGuide creates a new installation guide
func NewInstallationGuide() InstallationGuide {
	return &DefaultInstallationGuide{}
}

// GetPlatformInstructions returns platform-specific installation instructions
func (ig *DefaultInstallationGuide) GetPlatformInstructions(tool string) []InstallStep {
	switch tool {
	case "aapt2", "aapt":
		return ig.getAAPTInstructions()
	case "adb":
		return ig.getADBInstructions()
	default:
		return []InstallStep{
			{
				Description: fmt.Sprintf("Unknown tool: %s", tool),
				Manual:      true,
				Platform:    runtime.GOOS,
			},
		}
	}
}

// getAAPTInstructions returns AAPT installation instructions
func (ig *DefaultInstallationGuide) getAAPTInstructions() []InstallStep {
	var steps []InstallStep

	switch runtime.GOOS {
	case "linux":
		// Check available package managers
		packageManagers := ig.DetectPackageManagers()

		for _, pm := range packageManagers {
			switch pm {
			case "apt":
				steps = append(steps, InstallStep{
					Description: "Install via APT (Ubuntu/Debian)",
					Command:     "sudo apt-get update && sudo apt-get install -y aapt",
					Platform:    "linux",
				})
			case "brew":
				steps = append(steps, InstallStep{
					Description: "Install via Homebrew (recommended for latest version)",
					Command:     "brew install android-commandlinetools",
					Platform:    "linux",
				})
			case "yum":
				steps = append(steps, InstallStep{
					Description: "Install via YUM (RHEL/CentOS)",
					Command:     "sudo yum install -y android-tools",
					Platform:    "linux",
				})
			case "dnf":
				steps = append(steps, InstallStep{
					Description: "Install via DNF (Fedora)",
					Command:     "sudo dnf install -y android-tools",
					Platform:    "linux",
				})
			case "pacman":
				steps = append(steps, InstallStep{
					Description: "Install via Pacman (Arch Linux)",
					Command:     "sudo pacman -S --noconfirm android-sdk-build-tools",
					Platform:    "linux",
				})
			}
		}

		// Always add manual option
		steps = append(steps, InstallStep{
			Description: "Manual installation",
			Manual:      true,
			Platform:    "linux",
		})

	case "darwin":
		if ig.hasPackageManager("brew") {
			steps = append(steps, InstallStep{
				Description: "Install via Homebrew",
				Command:     "brew install --cask android-commandlinetools",
				Platform:    "darwin",
			})
		}

		steps = append(steps, InstallStep{
			Description: "Manual installation",
			Manual:      true,
			Platform:    "darwin",
		})

	case "windows":
		steps = append(steps, InstallStep{
			Description: "Download Android SDK Build Tools",
			Manual:      true,
			Platform:    "windows",
		})
	}

	return steps
}

// getADBInstructions returns ADB installation instructions
func (ig *DefaultInstallationGuide) getADBInstructions() []InstallStep {
	var steps []InstallStep

	switch runtime.GOOS {
	case "linux":
		packageManagers := ig.DetectPackageManagers()

		for _, pm := range packageManagers {
			switch pm {
			case "apt":
				steps = append(steps, InstallStep{
					Description: "Install via APT (Ubuntu/Debian)",
					Command:     "sudo apt-get update && sudo apt-get install -y adb",
					Platform:    "linux",
				})
			case "brew":
				steps = append(steps, InstallStep{
					Description: "Install via Homebrew",
					Command:     "brew install android-platform-tools",
					Platform:    "linux",
				})
			case "yum":
				steps = append(steps, InstallStep{
					Description: "Install via YUM (RHEL/CentOS)",
					Command:     "sudo yum install -y android-tools",
					Platform:    "linux",
				})
			case "dnf":
				steps = append(steps, InstallStep{
					Description: "Install via DNF (Fedora)",
					Command:     "sudo dnf install -y android-tools",
					Platform:    "linux",
				})
			case "pacman":
				steps = append(steps, InstallStep{
					Description: "Install via Pacman (Arch Linux)",
					Command:     "sudo pacman -S --noconfirm android-tools",
					Platform:    "linux",
				})
			}
		}

	case "darwin":
		if ig.hasPackageManager("brew") {
			steps = append(steps, InstallStep{
				Description: "Install via Homebrew",
				Command:     "brew install android-platform-tools",
				Platform:    "darwin",
			})
		}

	case "windows":
		steps = append(steps, InstallStep{
			Description: "Download Android SDK Platform Tools",
			Manual:      true,
			Platform:    "windows",
		})
	}

	return steps
}

// DetectPackageManagers detects available package managers
func (ig *DefaultInstallationGuide) DetectPackageManagers() []string {
	var managers []string

	packageManagers := []string{"apt-get", "brew", "yum", "dnf", "pacman"}

	for _, pm := range packageManagers {
		if ig.hasPackageManager(pm) {
			// Normalize names
			switch pm {
			case "apt-get":
				managers = append(managers, "apt")
			default:
				managers = append(managers, pm)
			}
		}
	}

	return managers
}

// hasPackageManager checks if a package manager is available
func (ig *DefaultInstallationGuide) hasPackageManager(manager string) bool {
	_, err := exec.LookPath(manager)
	return err == nil
}

// CanAutoInstall checks if a tool can be automatically installed
func (ig *DefaultInstallationGuide) CanAutoInstall(tool string) bool {
	instructions := ig.GetPlatformInstructions(tool)

	for _, step := range instructions {
		if !step.Manual && step.Command != "" {
			return true
		}
	}

	return false
}

// AutoInstall attempts to automatically install a tool
func (ig *DefaultInstallationGuide) AutoInstall(tool string) error {
	if !ig.CanAutoInstall(tool) {
		return fmt.Errorf("automatic installation not supported for %s on %s", tool, runtime.GOOS)
	}

	instructions := ig.GetPlatformInstructions(tool)

	// Try the first available automatic installation method
	for _, step := range instructions {
		if !step.Manual && step.Command != "" {
			fmt.Printf("Executing: %s\n", step.Description)

			// Split command into parts
			parts := strings.Fields(step.Command)
			if len(parts) == 0 {
				continue
			}

			cmd := exec.Command(parts[0], parts[1:]...)
			cmd.Stdout = nil // We'll handle output in the caller
			cmd.Stderr = nil

			if err := cmd.Run(); err != nil {
				return fmt.Errorf("installation failed: %w", err)
			}

			return nil
		}
	}

	return fmt.Errorf("no automatic installation method available")
}

// GetManualInstructions returns manual installation instructions
func (ig *DefaultInstallationGuide) GetManualInstructions(tool string) []string {
	var instructions []string

	switch tool {
	case "aapt2", "aapt":
		instructions = append(instructions,
			"1. Download Android SDK Command Line Tools:",
			"   https://developer.android.com/studio#command-tools",
			"2. Extract the downloaded file",
			"3. Add the build-tools directory to your PATH",
			"4. Verify installation: aapt2 version",
		)

	case "adb":
		instructions = append(instructions,
			"1. Download Android SDK Platform Tools:",
			"   https://developer.android.com/studio/releases/platform-tools",
			"2. Extract the downloaded file",
			"3. Add the platform-tools directory to your PATH",
			"4. Verify installation: adb version",
		)

	default:
		instructions = append(instructions, fmt.Sprintf("No manual instructions available for %s", tool))
	}

	return instructions
}

// InstallationManager combines dependency management with installation guidance
type InstallationManager struct {
	depManager DependencyManager
	installer  InstallationGuide
}

// NewInstallationManager creates a new installation manager
func NewInstallationManager() *InstallationManager {
	return &InstallationManager{
		depManager: NewDependencyManager(),
		installer:  NewInstallationGuide(),
	}
}

// GetDependencyManager returns the dependency manager
func (im *InstallationManager) GetDependencyManager() DependencyManager {
	return im.depManager
}

// GetInstallationGuide returns the installation guide
func (im *InstallationManager) GetInstallationGuide() InstallationGuide {
	return im.installer
}

// CheckAndSuggestInstallation checks dependencies and suggests installation
func (im *InstallationManager) CheckAndSuggestInstallation(command string) ([]DependencyStatus, []InstallStep, error) {
	// Check dependencies for the command
	deps := im.depManager.CheckForCommand(command)

	var installSteps []InstallStep
	var missingRequired []string

	for _, dep := range deps {
		if !dep.Available {
			if dep.Required {
				missingRequired = append(missingRequired, dep.Name)
			}

			// Get installation instructions
			steps := im.installer.GetPlatformInstructions(dep.Name)
			installSteps = append(installSteps, steps...)
		}
	}

	if len(missingRequired) > 0 {
		return deps, installSteps, fmt.Errorf("required dependencies missing: %s", strings.Join(missingRequired, ", "))
	}

	return deps, installSteps, nil
}

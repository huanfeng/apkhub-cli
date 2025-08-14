package version

import (
	"fmt"
	"runtime"
)

var (
	// Version is the version of the application, set by build flags
	Version = "dev"
	// Commit is the git commit hash, set by build flags
	Commit = "unknown"
	// BuildDate is the build date, set by build flags
	BuildDate = "unknown"
)

// Info returns version information
func Info() string {
	return fmt.Sprintf("ApkHub CLI %s\nCommit: %s\nBuilt: %s\nGo: %s\nOS/Arch: %s/%s",
		Version,
		Commit,
		BuildDate,
		runtime.Version(),
		runtime.GOOS,
		runtime.GOARCH,
	)
}

// Short returns short version string
func Short() string {
	return Version
}

//go:build !windows

package i18n

// getPlatformLocales returns OS-specific locale identifiers.
// Non-Windows platforms rely on environment variables, so return nil here.
func getPlatformLocales() []string {
	return nil
}

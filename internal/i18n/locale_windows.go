//go:build windows

package i18n

import "golang.org/x/sys/windows"

// getPlatformLocales returns preferred UI languages on Windows.
func getPlatformLocales() []string {
	var locales []string

	if langs, err := windows.GetUserPreferredUILanguages(windows.MUI_LANGUAGE_NAME); err == nil {
		for _, l := range langs {
			if l != "" {
				locales = append(locales, l)
			}
		}
	}

	if len(locales) == 0 {
		if name, err := windows.GetUserDefaultLocaleName(); err == nil && name != "" {
			locales = append(locales, name)
		}
	}

	return locales
}

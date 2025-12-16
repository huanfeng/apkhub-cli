package i18n

import (
	"embed"
	"fmt"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
	goi18n "github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

var (
	bundle           *goi18n.Bundle
	localizer        *goi18n.Localizer
	currentLanguage  = language.English
	supportedMatcher = language.NewMatcher([]language.Tag{
		language.English,
		language.SimplifiedChinese,
		language.Chinese,
	})
)

//go:embed locales/*.toml
var localeFS embed.FS

// Init initializes the i18n bundle and chooses the best language using:
//  1. langOverride (from --lang)
//  2. APKHUB_LANG environment variable
//  3. LC_ALL / LC_MESSAGES / LANG
//  4. Fallback to English
func Init(langOverride string) error {
	bundle = goi18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)

	if err := loadMessageFiles(); err != nil {
		return fmt.Errorf("load locales: %w", err)
	}

	chosen := selectLanguage(langOverride)
	localizer = goi18n.NewLocalizer(bundle, chosen.String(), language.English.String())
	currentLanguage = chosen

	return nil
}

// T translates a message by ID with optional template data.
// If translation fails, it falls back to the message ID to avoid empty output.
func T(id string, data ...map[string]interface{}) string {
	templateData := map[string]interface{}{}
	if len(data) > 0 && data[0] != nil {
		templateData = data[0]
	}

	if localizer == nil {
		// Best-effort initialization without override.
		if err := Init(""); err != nil {
			fmt.Fprintf(os.Stderr, "i18n init failed: %v\n", err)
		}
	}

	if localizer == nil {
		// Still nil (likely init failure); return the message ID to avoid panic.
		return id
	}

	msg, err := localizer.Localize(&goi18n.LocalizeConfig{
		MessageID:      id,
		TemplateData:   templateData,
		PluralCount:    findPluralCount(templateData),
		DefaultMessage: &goi18n.Message{ID: id, Other: id},
	})
	if err != nil || msg == "" {
		return id
	}
	return msg
}

// CurrentLanguage returns the chosen language tag.
func CurrentLanguage() language.Tag {
	return currentLanguage
}

func selectLanguage(langOverride string) language.Tag {
	var candidates []string
	if langOverride != "" {
		candidates = append(candidates, langOverride)
	}

	for _, key := range []string{"APKHUB_LANG", "LC_ALL", "LC_MESSAGES", "LANG"} {
		if val := strings.TrimSpace(os.Getenv(key)); val != "" {
			candidates = append(candidates, val)
		}
	}

	// On Windows, environment variables for locale are often missing.
	if len(candidates) == 0 {
		candidates = append(candidates, getPlatformLocales()...)
	}

	if len(candidates) == 0 {
		return language.English
	}

	for _, cand := range candidates {
		lower := strings.ToLower(cand)
		switch {
		case strings.HasPrefix(lower, "zh"):
			return language.Chinese
		case strings.HasPrefix(lower, "en"):
			// Keep searching in case zh is later; otherwise fallback will match en.
		}
	}

	var tags []language.Tag
	for _, cand := range candidates {
		clean := strings.TrimSpace(cand)
		// Normalize common locale strings like zh_CN.UTF-8 -> zh-CN
		if idx := strings.Index(clean, "."); idx >= 0 {
			clean = clean[:idx]
		}
		clean = strings.ReplaceAll(clean, "_", "-")

		tag, err := language.Parse(clean)
		if err != nil {
			// Fallback heuristics for common locales
			lower := strings.ToLower(clean)
			switch {
			case strings.HasPrefix(lower, "zh"):
				tag = language.Chinese
				err = nil
			case strings.HasPrefix(lower, "en"):
				tag = language.English
				err = nil
			}
		}

		if err == nil {
			tags = append(tags, tag)
		}
	}

	if len(tags) == 0 {
		return language.English
	}

	tag, _, _ := supportedMatcher.Match(tags...)
	return tag
}

func loadMessageFiles() error {
	files := []string{
		"locales/active.en.toml",
		"locales/active.zh.toml",
	}

	for _, file := range files {
		if _, err := bundle.LoadMessageFileFS(localeFS, file); err != nil {
			return fmt.Errorf("load %s: %w", file, err)
		}
	}

	return nil
}

func findPluralCount(data map[string]interface{}) interface{} {
	if data == nil {
		return nil
	}

	for _, key := range []string{"count", "Count", "total", "Total"} {
		if val, ok := data[key]; ok {
			return val
		}
	}

	return nil
}

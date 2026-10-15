package models

import "strings"

// Locale is a language the platform can write to a user in. It matches the
// frontend's Paraglide locales.
type Locale string

const (
	LocaleEnglish Locale = "en"
	LocaleSpanish Locale = "es"

	DefaultLocale = LocaleEnglish
)

var Locales = []Locale{LocaleEnglish, LocaleSpanish}

// SupportedLocale reads the language of a tag ("es", "es-AR", "es_AR") and
// says whether the platform speaks it.
func SupportedLocale(tag string) (Locale, bool) {
	language, _, _ := strings.Cut(strings.ToLower(strings.TrimSpace(tag)), "-")
	language, _, _ = strings.Cut(language, "_")

	for _, locale := range Locales {
		if Locale(language) == locale {
			return locale, true
		}
	}
	return "", false
}

// ParseLocale is SupportedLocale for places where a bad tag must not fail the
// request: anything unsupported becomes the default.
func ParseLocale(tag string) Locale {
	if locale, ok := SupportedLocale(tag); ok {
		return locale
	}
	return DefaultLocale
}

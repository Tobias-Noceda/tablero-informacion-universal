package models

import "testing"

func TestParseLocale_ReadsLanguageTagsAndFallsBackToTheDefault(t *testing.T) {
	cases := []struct {
		tag  string
		want Locale
	}{
		{"es", LocaleSpanish},
		{"es-AR", LocaleSpanish},
		{" ES_ar ", LocaleSpanish},
		{"en-US", LocaleEnglish},
		{"fr", DefaultLocale},
		{"", DefaultLocale},
		{"spanish", DefaultLocale},
	}
	for _, c := range cases {
		if got := ParseLocale(c.tag); got != c.want {
			t.Errorf("ParseLocale(%q) = %q, want %q", c.tag, got, c.want)
		}
	}
}

func TestSupportedLocale_RefusesWhatItCannotServe(t *testing.T) {
	if locale, ok := SupportedLocale("es-AR"); !ok || locale != LocaleSpanish {
		t.Errorf("es-AR = %q, %v", locale, ok)
	}
	for _, tag := range []string{"", "fr", "spanish"} {
		if _, ok := SupportedLocale(tag); ok {
			t.Errorf("%q must not be supported", tag)
		}
	}
}

func TestUserPreferredLocale_DefaultsForUsersStoredWithoutOne(t *testing.T) {
	if got := (&User{}).PreferredLocale(); got != DefaultLocale {
		t.Errorf("empty = %q", got)
	}
	if got := (&User{Locale: LocaleSpanish}).PreferredLocale(); got != LocaleSpanish {
		t.Errorf("es = %q", got)
	}
	if got := (&User{Locale: LocaleSpanish}).Profile().Locale; got != LocaleSpanish {
		t.Errorf("profile locale = %q", got)
	}
}

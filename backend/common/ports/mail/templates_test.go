package mail

import (
	"strings"
	"testing"

	"github.com/Secreto31126/tesis/common/models"
)

func mustLoad(t *testing.T) *Templates {
	t.Helper()
	templates, err := LoadTemplates()
	if err != nil {
		t.Fatalf("load templates: %v", err)
	}
	return templates
}

func vars() map[string]string {
	return map[string]string{"Name": "Ana", "Link": "https://tiu.example/verify?token=abc"}
}

func TestRender_EveryTemplateInEveryLocaleCarriesItsValues(t *testing.T) {
	templates := mustLoad(t)

	for _, locale := range models.Locales {
		for _, template := range models.MailTemplates {
			rendered, err := templates.Render(models.Mail{To: "ana@example.com", Locale: locale, Template: template, Vars: vars()})
			if err != nil {
				t.Errorf("%s/%s: %v", locale, template, err)
				continue
			}
			if rendered.Subject == "" || strings.Contains(rendered.Subject, "\n") {
				t.Errorf("%s/%s: subject = %q", locale, template, rendered.Subject)
			}
			for part, body := range map[string]string{"text": rendered.Text, "html": rendered.HTML} {
				if !strings.Contains(body, "Ana") || !strings.Contains(body, "https://tiu.example/verify?token=abc") {
					t.Errorf("%s/%s: %s lacks the values: %q", locale, template, part, body)
				}
			}
			if !strings.Contains(rendered.HTML, `lang="`+string(locale)+`"`) {
				t.Errorf("%s/%s: html is not declared in its language", locale, template)
			}
		}
	}
}

func TestRender_WritesInTheMailsLocale(t *testing.T) {
	templates := mustLoad(t)

	english, _ := templates.Render(models.Mail{Locale: models.LocaleEnglish, Template: models.MailResetPassword, Vars: vars()})
	spanish, _ := templates.Render(models.Mail{Locale: models.LocaleSpanish, Template: models.MailResetPassword, Vars: vars()})
	if english.Subject == spanish.Subject || english.Text == spanish.Text {
		t.Errorf("both locales read the same: %q / %q", english.Subject, spanish.Subject)
	}
}

func TestRender_UnknownOrMissingLocaleUsesTheDefault(t *testing.T) {
	templates := mustLoad(t)

	want, _ := templates.Render(models.Mail{Locale: models.DefaultLocale, Template: models.MailVerifyEmail, Vars: vars()})
	for _, locale := range []models.Locale{"", "fr"} {
		got, err := templates.Render(models.Mail{Locale: locale, Template: models.MailVerifyEmail, Vars: vars()})
		if err != nil || got.Subject != want.Subject {
			t.Errorf("locale %q: subject = %q, err = %v", locale, got.Subject, err)
		}
	}
}

func TestRender_EscapesInjectedValuesInHTMLOnly(t *testing.T) {
	templates := mustLoad(t)
	values := vars()
	values["Name"] = `<script>alert(1)</script>`

	rendered, err := templates.Render(models.Mail{Locale: models.LocaleEnglish, Template: models.MailVerifyEmail, Vars: values})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(rendered.HTML, "<script>") {
		t.Errorf("html carries the raw value: %q", rendered.HTML)
	}
	if !strings.Contains(rendered.Text, "<script>") {
		t.Errorf("plain text must keep the value as is: %q", rendered.Text)
	}
}

func TestRender_RefusesAMissingValue(t *testing.T) {
	templates := mustLoad(t)

	_, err := templates.Render(models.Mail{Locale: models.LocaleEnglish, Template: models.MailVerifyEmail, Vars: map[string]string{"Name": "Ana"}})
	if err == nil {
		t.Error("a mail without its link must not render")
	}
}

func TestRender_RefusesAnUnknownTemplate(t *testing.T) {
	templates := mustLoad(t)

	if _, err := templates.Render(models.Mail{Locale: models.LocaleEnglish, Template: "nope", Vars: vars()}); err == nil {
		t.Error("an unknown template must not render")
	}
}

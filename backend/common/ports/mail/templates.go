package mail

import (
	"bytes"
	"embed"
	"fmt"
	htmltemplate "html/template"
	"io"
	"maps"
	"path"
	"strings"
	texttemplate "text/template"

	"github.com/Secreto31126/tesis/common/models"
)

//go:embed templates
var embedded embed.FS

const layoutFile = "templates/layout.html.tmpl"

// Rendered is a mail written out in one language, ready to deliver.
type Rendered struct {
	Subject string
	Text    string
	HTML    string
}

type templateKey struct {
	locale   models.Locale
	template models.MailTemplate
}

// Templates holds every mail in every locale, parsed once. Subject and text
// go through text/template; the HTML part goes through html/template so the
// values a user controls (their name) cannot inject markup.
type Templates struct {
	text map[templateKey]*texttemplate.Template
	html map[templateKey]*htmltemplate.Template
}

type executor interface {
	ExecuteTemplate(w io.Writer, name string, data any) error
}

// LoadTemplates refuses to start with a locale that lacks a template, so a
// missing translation shows up at boot instead of in someone's inbox.
func LoadTemplates() (*Templates, error) {
	templates := &Templates{
		text: make(map[templateKey]*texttemplate.Template),
		html: make(map[templateKey]*htmltemplate.Template),
	}

	for _, locale := range models.Locales {
		for _, template := range models.MailTemplates {
			file := path.Join("templates", string(locale), string(template)+".tmpl")
			key := templateKey{locale, template}

			text, err := texttemplate.New(path.Base(file)).Option("missingkey=error").ParseFS(embedded, file)
			if err != nil {
				return nil, fmt.Errorf("mail template %s: %w", file, err)
			}
			html, err := htmltemplate.New(path.Base(file)).Option("missingkey=error").ParseFS(embedded, layoutFile, file)
			if err != nil {
				return nil, fmt.Errorf("mail template %s: %w", file, err)
			}

			for _, part := range []string{"subject", "text", "html"} {
				if text.Lookup(part) == nil {
					return nil, fmt.Errorf("mail template %s: no %q block", file, part)
				}
			}

			templates.text[key] = text
			templates.html[key] = html
		}
	}

	return templates, nil
}

// Render writes a mail in its locale, or in the default one when the
// recipient's is empty or unknown.
func (t *Templates) Render(mail models.Mail) (Rendered, error) {
	locale := models.ParseLocale(string(mail.Locale))
	key := templateKey{locale, mail.Template}

	text, ok := t.text[key]
	if !ok {
		return Rendered{}, fmt.Errorf("unknown mail template %q", mail.Template)
	}

	data := make(map[string]string, len(mail.Vars)+1)
	maps.Copy(data, mail.Vars)
	data["Locale"] = string(locale)

	subject, err := execute(text, "subject", data)
	if err != nil {
		return Rendered{}, err
	}
	body, err := execute(text, "text", data)
	if err != nil {
		return Rendered{}, err
	}
	html, err := execute(t.html[key], "layout", data)
	if err != nil {
		return Rendered{}, err
	}

	return Rendered{
		Subject: strings.Join(strings.Fields(subject), " "),
		Text:    strings.TrimSpace(body) + "\n",
		HTML:    strings.TrimSpace(html) + "\n",
	}, nil
}

func execute(template executor, name string, data map[string]string) (string, error) {
	var out bytes.Buffer
	if err := template.ExecuteTemplate(&out, name, data); err != nil {
		return "", fmt.Errorf("render mail %s: %w", name, err)
	}
	return out.String(), nil
}

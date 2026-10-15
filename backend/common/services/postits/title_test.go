package postits

import (
	"net/url"
	"testing"

	"github.com/Secreto31126/tesis/common/models"
)

func TestTitleVars(t *testing.T) {
	outputs := map[string]string{"min": ".min", "max": ".max"}

	cases := map[string]bool{
		"":                          false,
		"Plain title":               false,
		"Min {{min}}":               true,
		"Min {{ min }}":             true,
		"Unknown {{nope}}":          false,
		"Mixed {{nope}} {{max}}":    true,
		"Section {{#min}}x{{/min}}": false,
		"Broken {{min":              false,
	}

	for text, want := range cases {
		if got := titleVars(text, outputs); got != want {
			t.Errorf("titleVars(%q) = %v, want %v", text, got, want)
		}
	}
}

func TestOutputsOf_ParamsWithoutResource(t *testing.T) {
	static := &models.PostIts{Params: map[string]string{"text": "hi"}, Query: map[string]string{"q": "."}}
	if _, ok := outputsOf(static)["text"]; !ok {
		t.Error("a resource-less post-it answers with its params")
	}

	fetched := &models.PostIts{Resource: &url.URL{}, Params: map[string]string{"$x": ""}, Query: map[string]string{"q": "."}}
	if _, ok := outputsOf(fetched)["q"]; !ok {
		t.Error("a fetched post-it answers with its query keys")
	}
}

package postits

import (
	"net/url"
	"testing"

	"github.com/Secreto31126/tesis/common/mocks"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
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

func createTitled(t *testing.T, title models.Title) models.Title {
	t.Helper()

	var got *models.PostIts
	db := &mocks.MockDB{
		CreatePostItFn: func(p *models.PostIts, _ string, _ models.Position) (*models.PostIts, error) {
			got = p
			return p, nil
		},
	}

	svc := newService(db, nil, nil)
	if _, err := svc.CreatePostIt(boardOwner, &models.PostIts{Board: uuid.New(), WellKnown: "dolar_oficial", Title: title}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return got.Title
}

func TestCreatePostIt_EmptyTitleKeepsTemplate(t *testing.T) {
	got := createTitled(t, models.Title{})
	want := configuredPostIts["dolar_oficial"].template.Title
	if got != want {
		t.Errorf("title = %+v, want the template's %+v", got, want)
	}
}

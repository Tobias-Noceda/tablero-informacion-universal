package postits

import (
	"encoding/json"
	"net/http"
	"net/url"
	"testing"

	"github.com/Secreto31126/tesis/common/mocks"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

func oneOffBody(board uuid.UUID, resource, method, query string) string {
	return `{"board":"` + board.String() + `","title":{"text":"{{fact}}","vars":true},` +
		`"resource":"` + resource + `","request":{"method":"` + method + `"},` +
		`"response":"json","query":` + query + `,"rate":600}`
}

func TestCreatePostIt_OneOff(t *testing.T) {
	var stored *models.PostIts
	var storedType string
	db := &mocks.MockDB{
		CreatePostItFn: func(p *models.PostIts, _ models.BoardPostItTitle, ptype string, _ models.Position) (*models.PostIts, error) {
			stored, storedType = p, ptype
			return p, nil
		},
	}
	r := setupRouter(db, nil, nil)

	w := do(r, http.MethodPost, "/post-its", oneOffBody(uuid.New(), "https://catfact.ninja/fact", "GET", `{"fact":".fact"}`))
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body: %s)", w.Code, w.Body.String())
	}
	if stored.Resource.String() != "https://catfact.ninja/fact" || storedType != "" {
		t.Errorf("stored %v with type %q, want the URL and no type", stored.Resource, storedType)
	}
	var got struct{ Resource string }
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil || got.Resource != "https://catfact.ninja/fact" {
		t.Errorf("resource in the answer = %q (%v), want the URL", got.Resource, err)
	}
}

func TestCreatePostIt_OneOffRefusals(t *testing.T) {
	cases := map[string]string{
		"file scheme": oneOffBody(uuid.New(), "file:///etc/passwd", "GET", `{"fact":".fact"}`),
		"post":        oneOffBody(uuid.New(), "https://catfact.ninja/fact", "POST", `{"fact":".fact"}`),
		"no outputs":  oneOffBody(uuid.New(), "https://catfact.ninja/fact", "GET", `{}`),
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			db := &mocks.MockDB{
				CreatePostItFn: func(p *models.PostIts, _ models.BoardPostItTitle, _ string, _ models.Position) (*models.PostIts, error) {
					t.Error("stored anyway")
					return p, nil
				},
			}
			w := do(setupRouter(db, nil, nil), http.MethodPost, "/post-its", body)
			if w.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want 400 (body: %s)", w.Code, w.Body.String())
			}
		})
	}
}

func TestEditPostIt_OneOffRequest(t *testing.T) {
	var gotSet map[string]any
	db := &mocks.MockDB{
		FindPostItFn: func(id uuid.UUID) (*models.PostIts, error) {
			return &models.PostIts{
				Id: id, Board: uuid.New(), RunAs: owner,
				Resource: &url.URL{Scheme: "https", Host: "catfact.ninja", Path: "/fact"},
				Request:  models.Request{Method: "GET"}, Response: "json",
				Query: map[string]string{"fact": ".fact"}, Rate: 600,
			}, nil
		},
		UpdatePostItFn: func(_ uuid.UUID, set map[string]any) error {
			gotSet = set
			return nil
		},
	}
	r := setupRouter(db, nil, nil)

	w := do(r, http.MethodPatch, "/post-its/"+uuid.New().String()+"/settings",
		`{"resource":"https://catfact.ninja/facts","request":{"method":"GET","queries":{"limit":"2"}}}`)
	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204 (body: %s)", w.Code, w.Body.String())
	}
	if resource, ok := gotSet["resource"].(*url.URL); !ok || resource.Path != "/facts" {
		t.Errorf("set = %v, want the parsed resource", gotSet)
	}
}

func TestEditPostIt_CardTypeRequestIsFixed(t *testing.T) {
	db := &mocks.MockDB{
		FindPostItFn: func(id uuid.UUID) (*models.PostIts, error) {
			return &models.PostIts{Id: id, Board: uuid.New(), WellKnown: "dog_facts"}, nil
		},
		UpdatePostItFn: func(uuid.UUID, map[string]any) error {
			t.Error("stored anyway")
			return nil
		},
	}
	w := do(setupRouter(db, nil, nil), http.MethodPatch, "/post-its/"+uuid.New().String()+"/settings",
		`{"resource":"https://evil.example"}`)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestCreatePostIt_OneOffRefusesAMalformedResource(t *testing.T) {
	db := &mocks.MockDB{
		CreatePostItFn: func(*models.PostIts, models.BoardPostItTitle, string, models.Position) (*models.PostIts, error) {
			t.Error("stored anyway")
			return nil, nil
		},
	}
	r := setupRouter(db, nil, nil)

	for name, resource := range map[string]string{
		"bad escape": "https://example.com/%zz",
		"missing":    "",
		"not http":   "file:///etc/passwd",
	} {
		w := do(r, http.MethodPost, "/post-its", oneOffBody(uuid.New(), resource, "GET", `{"fact":".fact"}`))
		if w.Code != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400", name, w.Code)
		}
	}
}

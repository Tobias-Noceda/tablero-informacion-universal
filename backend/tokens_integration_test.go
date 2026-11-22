//go:build integration

package main

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Secreto31126/tesis/common/models"
	t_srv "github.com/Secreto31126/tesis/common/services/tokens"
	"github.com/google/uuid"
)

// tokenFor creates a personal access token for c through the API.
func (c *client) tokenFor(name string) t_srv.Created {
	c.s.t.Helper()
	var created t_srv.Created
	c.s.mustJSON(c.do(http.MethodPost, "/api/v1/users/"+c.id+"/tokens", map[string]any{"name": name}), http.StatusCreated, &created)
	return created
}

func TestEndToEnd_AccessTokenReadsAsItsOwner(t *testing.T) {
	s := newStack(t)
	ana := s.signUp("ana")
	bob := s.signUp("bob")
	board := ana.newBoard("Ana's memories")
	strange := bob.newBoard("Bob's memories")

	created := ana.tokenFor("demo")
	if !strings.HasPrefix(created.Token, t_srv.PREFIX) {
		t.Fatalf("token = %q", created.Token)
	}
	if life := time.Until(created.ExpiresAt); life < 14*24*time.Hour || life > 15*24*time.Hour {
		t.Errorf("expires in %v, want 15 days", life)
	}
	pat := &client{s, ana.id, ana.email, created.Token}

	var boards []models.Board
	s.mustJSON(pat.do(http.MethodGet, "/api/v1/boards", nil), http.StatusOK, &boards)
	if len(boards) != 1 || boards[0].Id != board.Id {
		t.Errorf("boards = %+v", boards)
	}
	s.mustJSON(pat.do(http.MethodGet, "/api/v1/boards/"+board.Id.String(), nil), http.StatusOK, nil)
	s.mustJSON(pat.do(http.MethodGet, "/api/v1/boards/"+board.Id.String()+"/post-its", nil), http.StatusOK, nil)

	// Board access rules still apply: someone else's board does not exist.
	if w := pat.do(http.MethodGet, "/api/v1/boards/"+strange.Id.String(), nil); w.Code != http.StatusNotFound {
		t.Errorf("stranger's board: %d", w.Code)
	}

	// Boards and cards only: no sharing, no secrets, no system routes, no more tokens.
	s.mustJSON(pat.do(http.MethodPost, "/api/v1/boards", map[string]string{"name": "From a tool"}), http.StatusCreated, nil)
	refused := []struct{ method, path string }{
		{http.MethodPut, "/api/v1/boards/" + board.Id.String() + "/members"},
		{http.MethodGet, "/api/v1/boards/" + board.Id.String() + "/secrets"},
		{http.MethodGet, "/api/v1/system/keys"},
		{http.MethodGet, "/api/v1/users/" + ana.id + "/tokens"},
		{http.MethodPost, "/api/v1/users/" + ana.id + "/tokens"},
	}
	for _, r := range refused {
		if w := pat.do(r.method, r.path, map[string]string{"name": "x"}); w.Code != http.StatusForbidden {
			t.Errorf("%s %s with a token: %d, want 403", r.method, r.path, w.Code)
		}
	}

	var listed []models.AccessToken
	s.mustJSON(ana.do(http.MethodGet, "/api/v1/users/"+ana.id+"/tokens", nil), http.StatusOK, &listed)
	if len(listed) != 1 || listed[0].LastUsedAt == nil || listed[0].Prefix != created.Prefix {
		t.Errorf("listed = %+v", listed)
	}

	if w := bob.do(http.MethodDelete, "/api/v1/users/"+bob.id+"/tokens/"+created.Id.String(), nil); w.Code != http.StatusNotFound {
		t.Errorf("bob revoking ana's token: %d", w.Code)
	}
	if w := ana.do(http.MethodDelete, "/api/v1/users/"+ana.id+"/tokens/"+created.Id.String(), nil); w.Code != http.StatusNoContent {
		t.Fatalf("revoke: %d", w.Code)
	}
	if w := pat.do(http.MethodGet, "/api/v1/boards", nil); w.Code != http.StatusUnauthorized {
		t.Errorf("revoked token: %d", w.Code)
	}
	if w := s.request(t_srv.PREFIX+uuid.NewString(), http.MethodGet, "/api/v1/boards", nil); w.Code != http.StatusUnauthorized {
		t.Errorf("forged token: %d", w.Code)
	}

	// Only the creation response ever carries the token.
	s.responses = s.responses[len(s.responses)-1:]
	s.assertNoResponseContains(created.Token)
}

// Walk every mounted private route with a token: only the allowlist gets
// past the middleware.
// tokenWrites are the only writes a token may make: boards, cards and strands.
var tokenWrites = map[string]bool{
	"POST /api/v1/boards":                       true,
	"DELETE /api/v1/boards/:id":                 true,
	"POST /api/v1/boards/:id/strands":           true,
	"PATCH /api/v1/boards/:id/strands/:strand":  true,
	"DELETE /api/v1/boards/:id/strands/:strand": true,
	"POST /api/v1/post-its":                     true,
	"POST /api/v1/post-its/:id/duplicate":       true,
	"POST /api/v1/post-its/preview":             true,
	"POST /api/v1/users/:id/postit-templates":   true,
	"POST /api/v1/boards/:id/postit-templates":  true,
	"PUT /api/v1/postit-templates/:key":         true,
	"DELETE /api/v1/postit-templates/:key":      true,
	"DELETE /api/v1/post-its/:id":               true,
	"PATCH /api/v1/post-its/:id/settings":       true,
	"PATCH /api/v1/post-its/:id/position":       true,
	"PATCH /api/v1/post-its/:id/style":          true,
	"PATCH /api/v1/post-its/:id/title":          true,
}

func TestEndToEnd_AccessTokenReachesOnlyTheAllowlist(t *testing.T) {
	s := newStack(t)
	ana := s.signUp("ana")
	token := ana.tokenFor("walk").Token
	params := strings.NewReplacer(":id", uuid.NewString(), ":strand", uuid.NewString(), ":user", "u", ":name", "KEY", ":token", uuid.NewString())

	for _, route := range s.app.router.Routes() {
		// Public routes treat a token like anyone else.
		if strings.HasPrefix(route.Path, "/api/v1/auth/") || publicRoutes[route.Path] {
			continue
		}
		w := s.request(token, route.Method, params.Replace(route.Path), nil)
		allowed := w.Code != http.StatusForbidden || !strings.Contains(w.Body.String(), "token_not_allowed")
		if allowed && route.Method != http.MethodGet && !tokenWrites[route.Method+" "+route.Path] {
			t.Errorf("%s %s let a token through: %d", route.Method, route.Path, w.Code)
		}
	}
}

// An agent sets up card types for its person and for a board, which they then
// use picking only their own credential; it tries a request before saving.
// An organization's card types stay out of its reach.
func TestEndToEnd_AccessTokenSetsUpCardTypes(t *testing.T) {
	s := newStack(t)
	ana := s.signUp("ana")
	board := ana.newBoard("Repos")
	pat := &client{s, ana.id, ana.email, ana.tokenFor("agent").Token}

	cardType := map[string]any{
		"name":     "GitHub repo",
		"title":    map[string]any{"text": "{{name}} has {{stars}} stars", "vars": true},
		"resource": "https://api.github.com/repos/$repo",
		"request": map[string]any{
			"method":  "GET",
			"headers": map[string]string{"Authorization": "$credential"},
			"queries": map[string]string{},
		},
		"response": "json",
		"query":    map[string]string{"name": ".full_name", "stars": ".stargazers_count"},
		"rate":     600,
		"parameters": []map[string]any{
			{"key": "$repo", "label": "Repository (owner/name)", "type": "string"},
			{"key": "$credential", "label": "GitHub token", "type": "secret"},
		},
		"outputs": []map[string]any{{"key": "name", "label": "Name"}, {"key": "stars", "label": "Stars"}},
	}

	var mine models.PostitTemplate
	s.mustJSON(pat.do(http.MethodPost, "/api/v1/users/"+ana.id+"/postit-templates", cardType), http.StatusCreated, &mine)
	s.mustJSON(pat.do(http.MethodPost, "/api/v1/boards/"+board.Id.String()+"/postit-templates", cardType), http.StatusCreated, nil)

	var listed []models.PostitTemplate
	s.mustJSON(pat.do(http.MethodGet, "/api/v1/users/"+ana.id+"/postit-templates", nil), http.StatusOK, &listed)
	if len(listed) != 1 || listed[0].Key != mine.Key {
		t.Errorf("listed = %+v", listed)
	}
	// The person sees it as one of their card types on the board.
	s.mustJSON(ana.do(http.MethodGet, "/api/v1/postit-templates/"+mine.Key, nil), http.StatusOK, nil)

	cardType["name"] = "GitHub repository"
	s.mustJSON(pat.do(http.MethodPut, "/api/v1/postit-templates/"+mine.Key, cardType), http.StatusOK, nil)
	if w := pat.do(http.MethodDelete, "/api/v1/postit-templates/"+mine.Key, nil); w.Code != http.StatusNoContent {
		t.Errorf("delete: %d %s", w.Code, w.Body.String())
	}

	// Trying a request reaches the preview (here refused by the SSRF guard).
	w := pat.do(http.MethodPost, "/api/v1/post-its/preview", map[string]any{"board": board.Id, "resource": "http://127.0.0.1:9/x"})
	if w.Code == http.StatusForbidden {
		t.Errorf("preview with a token: %d %s", w.Code, w.Body.String())
	}

	var org models.Org
	s.mustJSON(ana.do(http.MethodPost, "/api/v1/orgs", map[string]string{"name": "acme"}), http.StatusCreated, &org)
	defer ana.do(http.MethodDelete, "/api/v1/orgs/"+org.Id.String(), nil)
	if w := pat.do(http.MethodPost, "/api/v1/orgs/"+org.Id.String()+"/postit-templates", cardType); w.Code != http.StatusForbidden && w.Code != http.StatusNotFound {
		t.Errorf("org card type with a token: %d, want refused", w.Code)
	}
}

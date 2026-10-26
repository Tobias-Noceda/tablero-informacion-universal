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

	// Read only: no writes, no secrets, no system routes, no more tokens.
	refused := []struct{ method, path string }{
		{http.MethodPost, "/api/v1/boards"},
		{http.MethodDelete, "/api/v1/boards/" + board.Id.String()},
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

// Walk every mounted private route with a token: only the read allowlist
// gets past the middleware.
func TestEndToEnd_AccessTokenReachesOnlyTheAllowlist(t *testing.T) {
	s := newStack(t)
	ana := s.signUp("ana")
	token := ana.tokenFor("walk").Token
	params := strings.NewReplacer(":id", uuid.NewString(), ":strand", uuid.NewString(), ":user", "u", ":name", "KEY", ":token", uuid.NewString())

	for _, route := range s.app.router.Routes() {
		if strings.HasPrefix(route.Path, "/api/v1/auth/") {
			continue
		}
		w := s.request(token, route.Method, params.Replace(route.Path), nil)
		allowed := w.Code != http.StatusForbidden || !strings.Contains(w.Body.String(), "token_not_allowed")
		if allowed && (route.Method != http.MethodGet) {
			t.Errorf("%s %s let a token through: %d", route.Method, route.Path, w.Code)
		}
	}
}

//go:build integration

package main

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Secreto31126/tesis/common/models"
	"github.com/Secreto31126/tesis/common/ports/safehttp"
	"github.com/google/uuid"
)

// staticCard puts a note on the board and answers its id.
func (c *client) staticCard(board models.Board, text string) uuid.UUID {
	c.s.t.Helper()
	var postit models.PostIts
	c.s.mustJSON(c.do(http.MethodPost, "/api/v1/post-its", map[string]any{
		"board": board.Id, "well_known": "static_card", "params": map[string]string{"text": text},
	}), http.StatusCreated, &postit)
	return postit.Id
}

func positionOf(t *testing.T, s *stack, board models.Board, card uuid.UUID) models.Position {
	t.Helper()
	stored, err := s.db.FindBoard(board.Id)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range stored.PostIts {
		if entry.Id == card {
			return entry.Position
		}
	}
	t.Fatalf("card %v is not on the board", card)
	return models.Position{}
}

// Editors moving different cards of one board at once lose none
// of the moves: each one changes only its own card in the board document.
func TestEndToEnd_ParallelMovesOfDifferentCardsAllLand(t *testing.T) {
	s := newStack(t)
	owner, ana := s.signUp("owner"), s.signUp("ana")
	board := owner.newBoard("conc-01")
	defer owner.do(http.MethodDelete, "/api/v1/boards/"+board.Id.String(), nil)
	owner.share(board, ana, models.BoardEditor)

	const cards = 20
	ids := make([]uuid.UUID, cards)
	for i := range ids {
		ids[i] = owner.staticCard(board, fmt.Sprint(i))
	}

	codes := inParallel(cards, func(i int) int {
		editor := owner
		if i%2 == 1 {
			editor = ana
		}
		return editor.do(http.MethodPatch, "/api/v1/post-its/"+ids[i].String()+"/position",
			map[string]float32{"x": float32(100 * i), "y": float32(-7 * i)}).Code
	})

	if codes[http.StatusNoContent] != cards {
		t.Fatalf("answers = %v", codes)
	}
	for i, id := range ids {
		if got := positionOf(t, s, board, id); got.X != float32(100*i) || got.Y != float32(-7*i) {
			t.Errorf("card %d is at %+v, want (%d, %d)", i, got, 100*i, -7*i)
		}
	}
}

// Two editors moving the same card at once: the last write wins,
// whole. The card never ends up with one editor's x and the other's y.
func TestEndToEnd_ParallelMovesOfOneCardKeepAWholePosition(t *testing.T) {
	s := newStack(t)
	owner, ana := s.signUp("owner"), s.signUp("ana")
	board := owner.newBoard("conc-02")
	defer owner.do(http.MethodDelete, "/api/v1/boards/"+board.Id.String(), nil)
	owner.share(board, ana, models.BoardEditor)
	card := owner.staticCard(board, "contested")

	const moves = 40
	sent := map[models.Position]bool{}
	positions := make([]models.Position, moves)
	for i := range positions {
		positions[i] = models.Position{X: float32(i), Y: float32(1000 + i)}
		sent[positions[i]] = true
	}

	inParallel(moves, func(i int) int {
		editor := owner
		if i%2 == 1 {
			editor = ana
		}
		return editor.do(http.MethodPatch, "/api/v1/post-its/"+card.String()+"/position",
			map[string]float32{"x": positions[i].X, "y": positions[i].Y}).Code
	})

	if got := positionOf(t, s, board, card); !sent[got] {
		t.Errorf("the card ended at %+v, which nobody sent", got)
	}
}

// Strands drawn to a card while it is deleted never outlive it.
func TestEndToEnd_StrandsToACardBeingDeletedDoNotDangle(t *testing.T) {
	s := newStack(t)
	owner := s.signUp("owner")
	board := owner.newBoard("conc-03")
	defer owner.do(http.MethodDelete, "/api/v1/boards/"+board.Id.String(), nil)

	for round := range 5 {
		source := owner.staticCard(board, fmt.Sprintf("source %d", round))
		target := owner.staticCard(board, fmt.Sprintf("target %d", round))

		inParallel(11, func(i int) int {
			if i == 5 {
				return owner.do(http.MethodDelete, "/api/v1/post-its/"+target.String(), nil).Code
			}
			return owner.do(http.MethodPost, "/api/v1/boards/"+board.Id.String()+"/strands",
				map[string]any{"source": source, "target": target}).Code
		})

		stored, err := s.db.FindBoard(board.Id)
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range stored.PostIts {
			if entry.Id == target {
				t.Fatalf("round %d: the deleted card is still on the board", round)
			}
		}
		for _, strand := range stored.Strands {
			if strand.Source == target || strand.Target == target {
				t.Errorf("round %d: strand %v points at the deleted card", round, strand.Id)
			}
		}
	}
}

// Two requests renewing a session with the same refresh cookie at
// once: one gets the new session, the other a 401, and the session lives on.
func TestEndToEnd_ParallelRenewalsWithOneCookie(t *testing.T) {
	s := newStack(t)
	cookie := s.verified()

	renew := func(c *http.Cookie) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
		req.AddCookie(c)
		req.Header.Set("Sec-Fetch-Site", "same-origin")
		w := httptest.NewRecorder()
		s.app.router.ServeHTTP(w, req)
		return w
	}

	var mu sync.Mutex
	var winners []*http.Cookie
	codes := inParallel(2, func(int) int {
		w := renew(cookie)
		if w.Code == http.StatusOK {
			mu.Lock()
			winners = append(winners, refreshCookie(w))
			mu.Unlock()
		}
		return w.Code
	})

	if codes[http.StatusOK] != 1 || codes[http.StatusUnauthorized] != 1 || len(winners) != 1 {
		t.Fatalf("answers = %v, want one 200 and one 401", codes)
	}
	// Within the grace window the spent cookie does not revoke the family.
	if w := renew(winners[0]); w.Code != http.StatusOK {
		t.Errorf("the winner's session was revoked: %d %s", w.Code, w.Body.String())
	}
}

// Two admins demoting each other at once: the organization always
// keeps an admin.
func TestEndToEnd_ParallelDemotionsKeepAnAdmin(t *testing.T) {
	s := newStack(t)

	for round := range 5 {
		ana, bob := s.signUp("ana"), s.signUp("bob")
		var org models.Org
		s.mustJSON(ana.do(http.MethodPost, "/api/v1/orgs", map[string]string{"name": fmt.Sprintf("conc-07-%d", round)}), http.StatusCreated, &org)
		members := "/api/v1/orgs/" + org.Id.String() + "/members"
		s.mustJSON(ana.do(http.MethodPut, members, map[string]string{"email": bob.email, "role": "admin"}), http.StatusOK, nil)

		inParallel(2, func(i int) int {
			if i == 0 {
				return ana.do(http.MethodPut, members, map[string]string{"email": bob.email, "role": "member"}).Code
			}
			return bob.do(http.MethodPut, members, map[string]string{"email": ana.email, "role": "member"}).Code
		})

		stored, err := s.db.FindOrg(org.Id)
		if err != nil {
			t.Fatal(err)
		}
		admins := 0
		for _, member := range stored.Members {
			if member.Role == models.OrgRoleAdmin {
				admins++
			}
		}
		if admins == 0 {
			t.Errorf("round %d: the organization has no admin left: %+v", round, stored.Members)
		}
	}
}

// Many cards needing one OAuth2 credential renewed at once spend
// the refresh once; the requests that lose the race wait for the new token
// instead of failing.
func TestEndToEnd_ParallelResolvesRenewAnOAuth2TokenOnce(t *testing.T) {
	original := safehttp.IsSafeIP
	safehttp.IsSafeIP = func(net.IP) bool { return true }
	t.Cleanup(func() { safehttp.IsSafeIP = original })

	var fetches atomic.Int32
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fetches.Add(1)
		time.Sleep(200 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"access_token":"fresh","token_type":"bearer","expires_in":3600}`)
	}))
	defer provider.Close()

	s := newStack(t)
	owner := s.signUp("owner")
	board := owner.newBoard("conc-08")
	defer owner.do(http.MethodDelete, "/api/v1/boards/"+board.Id.String(), nil)
	scope := models.BoardScope(board.Id)
	if err := s.app.secrets.PutOAuth2(scope, models.Principal{ID: owner.id}, "PROVIDER", &models.OAuth2Material{
		Flow: models.OAuth2ClientCredentials, ClientID: "id", ClientSecret: "secret", TokenURL: provider.URL + "/token",
	}); err != nil {
		t.Fatal(err)
	}

	const readers = 10
	codes := inParallel(readers, func(int) int {
		resolved, err := s.app.secrets.Resolve(scope, []string{"PROVIDER"})
		renewed := (&models.OAuth2Material{AccessToken: "fresh", TokenType: "bearer"}).Header()
		if err != nil || resolved["$PROVIDER"] != renewed {
			return http.StatusConflict
		}
		return http.StatusOK
	})

	if n := fetches.Load(); n != 1 {
		t.Errorf("the provider was asked %d times, want 1", n)
	}
	if codes[http.StatusOK] != readers {
		t.Errorf("answers = %v, want every reader to get the renewed token", codes)
	}

	// Once renewed, everyone reads the stored token without asking again.
	if _, err := s.app.secrets.Resolve(scope, []string{"PROVIDER"}); err != nil {
		t.Errorf("after the renewal: %v", err)
	}
	if n := fetches.Load(); n != 1 {
		t.Errorf("the provider was asked again after the renewal: %d", n)
	}
}

// A forced refresh (Max plan) reaches the card's API once every
// ten seconds, however many ask at once.
func TestEndToEnd_ParallelForcedRefreshesReachTheAPIOnce(t *testing.T) {
	original := safehttp.IsSafeIP
	safehttp.IsSafeIP = func(net.IP) bool { return true }
	t.Cleanup(func() { safehttp.IsSafeIP = original })

	var calls atomic.Int32
	stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"compra":1400,"venta":1450}`)
	}))
	defer stub.Close()

	s := newStack(t)
	owner := s.signUp("owner")
	board := owner.newBoard("conc-09")
	defer owner.do(http.MethodDelete, "/api/v1/boards/"+board.Id.String(), nil)
	var postit models.PostIts
	s.mustJSON(owner.do(http.MethodPost, "/api/v1/post-its", map[string]any{"board": board.Id, "well_known": "dolar_oficial"}), http.StatusCreated, &postit)
	resource, _ := url.Parse(stub.URL)
	if err := s.db.UpdatePostIt(postit.Id, map[string]any{"resource": resource}); err != nil {
		t.Fatalf("retarget: %v", err)
	}

	codes := inParallel(10, func(int) int {
		return owner.do(http.MethodGet, "/api/v1/post-its/"+postit.Id.String()+"?fresh=1", nil).Code
	})

	if codes[http.StatusOK] != 1 || codes[http.StatusTooManyRequests] != 9 {
		t.Errorf("answers = %v, want one 200 and nine 429", codes)
	}
	if n := calls.Load(); n != 1 {
		t.Errorf("the card's API was asked %d times, want 1", n)
	}
}

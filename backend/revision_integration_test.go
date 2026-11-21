//go:build integration

package main

import (
	"net/http"
	"testing"

	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

// Saving a card's settings changes the board document too, which is what
// the realtime service watches: the others run the card again at once
// instead of at their next refresh.
func TestEndToEnd_SavingSettingsRaisesTheCardsRevision(t *testing.T) {
	s := newStack(t)
	owner := s.signUp("owner")
	board := owner.newBoard("revisions")
	defer owner.do(http.MethodDelete, "/api/v1/boards/"+board.Id.String(), nil)

	var postit models.PostIts
	s.mustJSON(owner.do(http.MethodPost, "/api/v1/post-its",
		map[string]any{"board": board.Id, "well_known": "dolar_oficial"}), http.StatusCreated, &postit)

	revision := func() int {
		t.Helper()
		stored, err := s.db.FindBoard(board.Id)
		if err != nil {
			t.Fatalf("find board: %v", err)
		}
		for _, entry := range stored.PostIts {
			if entry.Id == postit.Id {
				return entry.Rev
			}
		}
		t.Fatalf("card %v is not on the board", postit.Id)
		return 0
	}

	if rev := revision(); rev != 0 {
		t.Fatalf("a new card starts at revision %d, want 0", rev)
	}
	settings := "/api/v1/post-its/" + postit.Id.String() + "/settings"
	for want := 1; want <= 2; want++ {
		s.mustJSON(owner.do(http.MethodPatch, settings, map[string]any{"rate": 300 + want}), http.StatusNoContent, nil)
		if rev := revision(); rev != want {
			t.Errorf("after save %d revision = %d, want %d", want, rev, want)
		}
	}

	// Moving it changes the board already; it is not a settings change.
	s.mustJSON(owner.do(http.MethodPatch, "/api/v1/post-its/"+postit.Id.String()+"/position", map[string]float32{"x": 10, "y": 20}), http.StatusNoContent, nil)
	if rev := revision(); rev != 2 {
		t.Errorf("after a move revision = %d, want it unchanged", rev)
	}

	// A board that is not the card's is never touched.
	if err := s.db.BumpPostIt(uuid.New(), postit.Id); err == nil {
		t.Error("bumping the card on another board succeeded")
	}
}

//go:build integration

package main

import (
	"net/http"
	"testing"

	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

// A card keeps its style on the board, a patch changes only what it sends
// and an empty value (zero for a size) goes back to the default, which is
// stored as nothing.
func TestEndToEnd_CardStyles(t *testing.T) {
	s := newStack(t)
	owner, viewer := s.signUp("owner"), s.signUp("viewer")

	board := owner.newBoard("cards")
	defer owner.do(http.MethodDelete, "/api/v1/boards/"+board.Id.String(), nil)
	owner.share(board, viewer, models.BoardViewer)

	card := func() uuid.UUID {
		var postit models.PostIts
		s.mustJSON(owner.do(http.MethodPost, "/api/v1/post-its", map[string]any{"board": board.Id, "well_known": "dolar_oficial"}), http.StatusCreated, &postit)
		return postit.Id
	}
	styled, plain := card(), card()
	stylePath := "/api/v1/post-its/" + styled.String() + "/style"

	var style models.CardStyle
	s.mustJSON(owner.do(http.MethodPatch, stylePath, map[string]any{
		"color": "purple", "border": "dotted", "width": 320.4, "height": 180,
	}), http.StatusOK, &style)

	want := models.CardStyle{Color: models.StrandColorPurple, Border: models.CardBorderDotted, Width: 320, Height: 180}
	if style != want {
		t.Fatalf("style = %+v, want %+v", style, want)
	}

	s.mustJSON(owner.do(http.MethodPatch, stylePath, map[string]any{"border": "", "height": 0}), http.StatusOK, &style)

	want = models.CardStyle{Color: models.StrandColorPurple, Width: 320}
	if style != want {
		t.Errorf("patched = %+v, want %+v", style, want)
	}

	stored, err := s.db.FindBoard(board.Id)
	if err != nil {
		t.Fatalf("find board: %v", err)
	}
	for _, entry := range stored.PostIts {
		switch entry.Id {
		case styled:
			if entry.CardStyle != want {
				t.Errorf("stored style = %+v, want %+v", entry.CardStyle, want)
			}
		case plain:
			if entry.CardStyle != (models.CardStyle{}) {
				t.Errorf("the other card changed: %+v", entry.CardStyle)
			}
		}
	}

	if w := owner.do(http.MethodPatch, stylePath, map[string]any{"width": 5000}); w.Code != http.StatusBadRequest {
		t.Errorf("too wide: status = %d, want 400", w.Code)
	}
	if w := owner.do(http.MethodPatch, "/api/v1/post-its/"+uuid.NewString()+"/style", map[string]any{"border": "solid"}); w.Code != http.StatusNotFound {
		t.Errorf("unknown card: status = %d, want 404", w.Code)
	}
	if w := viewer.do(http.MethodPatch, stylePath, map[string]any{"border": "solid"}); w.Code != http.StatusNotFound {
		t.Errorf("viewer: status = %d, want 404", w.Code)
	}
}

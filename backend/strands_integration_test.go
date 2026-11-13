//go:build integration

package main

import (
	"net/http"
	"testing"

	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

// A strand keeps its style on the board, a patch changes only what it sends
// and an empty value goes back to the default, which is stored as nothing.
func TestEndToEnd_StrandStyles(t *testing.T) {
	s := newStack(t)
	owner, viewer := s.signUp("owner"), s.signUp("viewer")

	board := owner.newBoard("strands")
	defer owner.do(http.MethodDelete, "/api/v1/boards/"+board.Id.String(), nil)
	owner.share(board, viewer, models.BoardViewer)
	base := "/api/v1/boards/" + board.Id.String()

	card := func() uuid.UUID {
		var postit models.PostIts
		s.mustJSON(owner.do(http.MethodPost, "/api/v1/post-its", map[string]any{"board": board.Id, "well_known": "dolar_oficial"}), http.StatusCreated, &postit)
		return postit.Id
	}
	source, target, other := card(), card(), card()

	var plain, styled models.Strand
	s.mustJSON(owner.do(http.MethodPost, base+"/strands", map[string]any{"source": other, "target": target}), http.StatusCreated, &plain)
	s.mustJSON(owner.do(http.MethodPost, base+"/strands", map[string]any{
		"source": source, "target": target, "label": " feeds ", "arrow": "end", "width": "thick", "color": "blue",
	}), http.StatusCreated, &styled)

	want := models.StrandStyle{Label: "feeds", Arrow: models.StrandArrowEnd, Width: models.StrandWidthThick, Color: models.StrandColorBlue}
	if styled.StrandStyle != want {
		t.Fatalf("created style = %+v, want %+v", styled.StrandStyle, want)
	}

	strandPath := base + "/strands/" + styled.Id.String()
	var patched models.Strand
	s.mustJSON(owner.do(http.MethodPatch, strandPath, map[string]any{"label": "", "arrow": "both"}), http.StatusOK, &patched)

	want = models.StrandStyle{Arrow: models.StrandArrowBoth, Width: models.StrandWidthThick, Color: models.StrandColorBlue}
	if patched.Id != styled.Id || patched.Source != source || patched.StrandStyle != want {
		t.Errorf("patched = %+v, want %+v", patched, want)
	}

	stored, err := s.db.FindBoard(board.Id)
	if err != nil {
		t.Fatalf("find board: %v", err)
	}
	for _, strand := range stored.Strands {
		switch strand.Id {
		case styled.Id:
			if strand.StrandStyle != want {
				t.Errorf("stored style = %+v, want %+v", strand.StrandStyle, want)
			}
		case plain.Id:
			if strand.StrandStyle != (models.StrandStyle{}) {
				t.Errorf("the other strand changed: %+v", strand.StrandStyle)
			}
		}
	}

	if w := owner.do(http.MethodPatch, strandPath, map[string]any{"color": "#ff0000"}); w.Code != http.StatusBadRequest {
		t.Errorf("invalid colour: status = %d, want 400", w.Code)
	}
	if w := owner.do(http.MethodPatch, base+"/strands/"+uuid.NewString(), map[string]any{"label": "x"}); w.Code != http.StatusNotFound {
		t.Errorf("unknown strand: status = %d, want 404", w.Code)
	}
	if w := viewer.do(http.MethodPatch, strandPath, map[string]any{"label": "x"}); w.Code != http.StatusNotFound {
		t.Errorf("viewer: status = %d, want 404", w.Code)
	}
}

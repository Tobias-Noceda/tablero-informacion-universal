package postits

import (
	"errors"
	"testing"

	"github.com/Secreto31126/tesis/common/mocks"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

// Saving a card's settings touches its entry on the board, so the board's
// change stream tells everyone looking at it to run the card again. The
// cached result goes first, or they would read the old one back.
func TestUpdatePostIt_BumpsTheCardOnTheBoardAfterDroppingItsResult(t *testing.T) {
	id, board := uuid.New(), uuid.New()
	var steps []string
	db := &mocks.MockDB{
		FindPostItFn: func(uuid.UUID) (*models.PostIts, error) {
			return &models.PostIts{Id: id, Board: board, RunAs: boardOwner.ID}, nil
		},
		UpdatePostItFn: func(uuid.UUID, map[string]any) error {
			steps = append(steps, "save")
			return nil
		},
		BumpPostItFn: func(gotBoard, gotCard uuid.UUID) error {
			if gotBoard != board || gotCard != id {
				t.Errorf("bumped %v on %v, want %v on %v", gotCard, gotBoard, id, board)
			}
			steps = append(steps, "bump")
			return nil
		},
	}
	cache := &mocks.MockCache{DropPostItResultFn: func(uuid.UUID) error {
		steps = append(steps, "drop")
		return nil
	}}

	if err := newService(db, cache, nil).UpdatePostIt(boardOwner, id, map[string]any{"rate": 60}); err != nil {
		t.Fatal(err)
	}

	if len(steps) != 3 || steps[0] != "save" || steps[1] != "drop" || steps[2] != "bump" {
		t.Errorf("steps = %v, want save, drop, bump", steps)
	}
}

// The settings are saved either way: a missed bump only means the others
// see the new result at their next refresh, as before.
func TestUpdatePostIt_AFailedBumpStillSaves(t *testing.T) {
	id := uuid.New()
	db := &mocks.MockDB{
		FindPostItFn:   existingCard(id),
		UpdatePostItFn: func(uuid.UUID, map[string]any) error { return nil },
		BumpPostItFn:   func(uuid.UUID, uuid.UUID) error { return errors.New("mongo down") },
	}

	if err := newService(db, nil, nil).UpdatePostIt(boardOwner, id, map[string]any{"rate": 60}); err != nil {
		t.Errorf("err = %v, want the update to succeed", err)
	}
}

func TestUpdatePostIt_NothingToSaveBumpsNothing(t *testing.T) {
	bumped := false
	db := &mocks.MockDB{BumpPostItFn: func(uuid.UUID, uuid.UUID) error {
		bumped = true
		return nil
	}}

	if err := newService(db, nil, nil).UpdatePostIt(boardOwner, uuid.New(), map[string]any{}); err != nil {
		t.Fatal(err)
	}
	if bumped {
		t.Error("an empty update touched the board")
	}
}

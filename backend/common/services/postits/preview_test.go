package postits

import (
	"errors"
	"reflect"
	"testing"

	"github.com/Secreto31126/tesis/common/mocks"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/Secreto31126/tesis/common/services/access"
	"github.com/google/uuid"
)

func previewService(db *mocks.MockDB, run *mocks.MockExecuter, secrets *mocks.MockSecretResolver, limiter *mocks.CountingLimiter) *PostItsService {
	if db == nil {
		db = boardOf(boardOwner, member)
	}
	if secrets == nil {
		secrets = &mocks.MockSecretResolver{}
	}
	if limiter == nil {
		limiter = &mocks.CountingLimiter{}
	}
	return New(db, &mocks.MockCache{}, run, secrets, access.New(nil), builtins(), nil, mocks.StaticEntitlements{}, limiter)
}

func TestPreviewPostIt_RunsTheOutputsWithoutSavingOrCaching(t *testing.T) {
	db := boardOf(boardOwner, member)
	db.CreatePostItFn = func(*models.PostIts, models.BoardPostItTitle, string, models.Position) (*models.PostIts, error) {
		t.Fatal("a preview stored the card")
		return nil, nil
	}
	var ran *models.PostIts
	run := &mocks.MockExecuter{ExecuteFn: func(p *models.PostIts) (any, error) {
		ran = p
		return map[string]any{"fact": "cats sleep"}, nil
	}}
	svc := previewService(db, run, nil, nil)
	svc.cache = &mocks.MockCache{AddPostItResultFn: func(*models.PostIts, any) error {
		t.Error("a preview cached its result")
		return nil
	}}

	got, err := svc.PreviewPostIt(member, oneOff(uuid.New(), nil))
	if err != nil {
		t.Fatalf("PreviewPostIt: %v", err)
	}
	want := Preview{Outputs: map[string]any{"fact": "cats sleep"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}
	if ran == nil || ran.RunAs != member.ID {
		t.Errorf("the request ran as %+v, want the editor who asked", ran)
	}
}

func TestPreviewPostIt_WithoutOutputsAnswersTheWholeResponse(t *testing.T) {
	var query map[string]string
	run := &mocks.MockExecuter{ExecuteFn: func(p *models.PostIts) (any, error) {
		query = p.Query
		return map[string]any{previewWhole: map[string]any{"data": []any{1.0, 2.0}}}, nil
	}}
	svc := previewService(nil, run, nil, nil)

	card := oneOff(uuid.New(), nil)
	card.Query = nil
	got, err := svc.PreviewPostIt(boardOwner, card)
	if err != nil {
		t.Fatalf("PreviewPostIt: %v", err)
	}
	if !reflect.DeepEqual(query, map[string]string{previewWhole: "."}) {
		t.Errorf("ran with query %v, want the whole document", query)
	}
	want := Preview{Response: map[string]any{"data": []any{1.0, 2.0}}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}
}

func TestPreviewPostIt_WholeResponseNeedsJSON(t *testing.T) {
	svc := previewService(nil, &mocks.MockExecuter{}, nil, nil)
	card := oneOff(uuid.New(), nil)
	card.Query = nil
	card.Response = "html"

	if _, err := svc.PreviewPostIt(boardOwner, card); !errors.Is(err, ErrInvalidOneOff) {
		t.Errorf("got %v, want ErrInvalidOneOff", err)
	}
}

func TestPreviewPostIt_OnlyEditors(t *testing.T) {
	viewer := models.Principal{ID: "viewer-id"}
	db := boardOf(boardOwner)
	find := db.FindBoardFn
	db.FindBoardFn = func(id uuid.UUID) (*models.Board, error) {
		board, err := find(id)
		board.Members = append(board.Members, models.BoardMember{User: viewer.ID, Role: models.BoardViewer})
		return board, err
	}
	run := &mocks.MockExecuter{ExecuteFn: func(*models.PostIts) (any, error) {
		t.Error("a request ran for someone who may not edit the board")
		return nil, nil
	}}
	svc := previewService(db, run, nil, nil)

	for _, who := range []models.Principal{viewer, outsider} {
		if _, err := svc.PreviewPostIt(who, oneOff(uuid.New(), nil)); !errors.Is(err, ErrNotAMember) {
			t.Errorf("%s: got %v, want ErrNotAMember", who.ID, err)
		}
	}
}

func TestPreviewPostIt_ChecksTheCredentialsItBinds(t *testing.T) {
	refused := errors.New("not yours")
	secrets := &mocks.MockSecretResolver{CanBindFn: func(models.Principal, uuid.UUID, []models.SecretRef) error { return refused }}
	run := &mocks.MockExecuter{ExecuteFn: func(*models.PostIts) (any, error) {
		t.Error("the request ran with a credential the editor may not use")
		return nil, nil
	}}
	svc := previewService(nil, run, secrets, nil)

	ref := models.SecretRef{Scope: models.UserScope("someone-else"), Name: "KEY"}
	card := oneOff(uuid.New(), map[string]models.SecretRef{"KEY": ref})
	card.Request.Headers = map[string]string{"Authorization": "$KEY"}
	if _, err := svc.PreviewPostIt(member, card); !errors.Is(err, refused) {
		t.Errorf("got %v, want the binding refused", err)
	}
}

func TestPreviewPostIt_IsRateLimitedPerUser(t *testing.T) {
	limiter := &mocks.CountingLimiter{Refuse: map[string]bool{"preview:" + member.ID: true}}
	run := &mocks.MockExecuter{ExecuteFn: func(*models.PostIts) (any, error) {
		t.Error("a refused preview still ran")
		return nil, nil
	}}
	svc := previewService(nil, run, nil, limiter)

	if _, err := svc.PreviewPostIt(member, oneOff(uuid.New(), nil)); !errors.Is(err, ErrRateLimited) {
		t.Errorf("got %v, want ErrRateLimited", err)
	}
}

func TestPreviewPostIt_RefusesAnInvalidRequest(t *testing.T) {
	svc := previewService(nil, &mocks.MockExecuter{}, nil, nil)
	card := oneOff(uuid.New(), nil)
	card.Resource = nil

	if _, err := svc.PreviewPostIt(member, card); !errors.Is(err, ErrInvalidOneOff) {
		t.Errorf("got %v, want ErrInvalidOneOff", err)
	}
}

package postits

import (
	"errors"
	"net/url"
	"testing"

	"github.com/Secreto31126/tesis/common/mocks"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/Secreto31126/tesis/common/services/access"
	"github.com/google/uuid"
)

// oneOff is a valid card that carries its own request.
func oneOff(board uuid.UUID, bindings map[string]models.SecretRef) *models.PostIts {
	return &models.PostIts{
		Board:    board,
		Resource: &url.URL{Scheme: "https", Host: "example.com", Path: "/fact"},
		Request:  models.Request{Method: "GET"},
		Response: "json",
		Query:    map[string]string{"fact": ".fact"},
		Rate:     600,
		Bindings: bindings,
	}
}

func TestValidateOneOff(t *testing.T) {
	cases := map[string]struct {
		edit func(*models.PostIts)
		ok   bool
	}{
		"valid":            {func(*models.PostIts) {}, true},
		"empty method":     {func(p *models.PostIts) { p.Request.Method = "" }, true},
		"lowercase get":    {func(p *models.PostIts) { p.Request.Method = "get" }, true},
		"no resource":      {func(p *models.PostIts) { p.Resource = nil }, false},
		"file scheme":      {func(p *models.PostIts) { p.Resource = &url.URL{Scheme: "file", Path: "/etc/passwd"} }, false},
		"no host":          {func(p *models.PostIts) { p.Resource = &url.URL{Scheme: "https"} }, false},
		"post":             {func(p *models.PostIts) { p.Request.Method = "POST" }, false},
		"xml response":     {func(p *models.PostIts) { p.Response = "xml" }, false},
		"no query":         {func(p *models.PostIts) { p.Query = nil }, false},
		"empty expression": {func(p *models.PostIts) { p.Query = map[string]string{"fact": " "} }, false},
		"empty header":     {func(p *models.PostIts) { p.Request.Headers = map[string]string{"": "x"} }, false},
		"negative rate":    {func(p *models.PostIts) { p.Rate = -1 }, false},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			p := oneOff(uuid.New(), nil)
			p.Request.Body = "dropped"
			c.edit(p)
			err := validateOneOff(p)
			if c.ok && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !c.ok && !errors.Is(err, ErrInvalidOneOff) {
				t.Fatalf("got %v, want ErrInvalidOneOff", err)
			}
			if c.ok && (p.Request.Method != "GET" || p.Request.Body != "") {
				t.Errorf("request = %+v, want GET without a body", p.Request)
			}
		})
	}
}

func TestCreatePostIt_RefusesAnInvalidOneOff(t *testing.T) {
	created := false
	db := boardOf(boardOwner)
	db.CreatePostItFn = func(p *models.PostIts, _ models.BoardPostItTitle, _ string, _ models.Position) (*models.PostIts, error) {
		created = true
		return p, nil
	}
	svc := newService(db, nil, nil)

	card := oneOff(uuid.New(), nil)
	card.Request.Method = "DELETE"
	_, err := svc.CreatePostIt(boardOwner, &models.BoardPostItTitle{Text: "Hi"}, card)
	if !errors.Is(err, ErrInvalidOneOff) {
		t.Errorf("got %v, want ErrInvalidOneOff", err)
	}
	if created {
		t.Error("the post-it was stored anyway")
	}
}

func TestUpdatePostIt_EditsAOneOffRequest(t *testing.T) {
	board := uuid.New()
	db := boardOf(boardOwner, member)
	db.FindPostItFn = func(id uuid.UUID) (*models.PostIts, error) {
		card := oneOff(board, nil)
		card.Id = id
		return card, nil
	}
	var got map[string]any
	db.UpdatePostItFn = func(_ uuid.UUID, set map[string]any) error {
		got = set
		return nil
	}
	var bound bool
	resolver := &mocks.MockSecretResolver{
		CanBindFn: func(models.Principal, uuid.UUID, []models.SecretRef) error {
			bound = true
			return nil
		},
	}
	svc := New(db, &mocks.MockCache{}, &mocks.MockExecuter{}, resolver, access.New(nil), builtins(), nil)

	err := svc.UpdatePostIt(member, uuid.New(), map[string]any{
		"resource": "https://api.example.com/v2",
		"request":  models.Request{Headers: map[string]string{"x-api-key": "$KEY"}},
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	resource, ok := got["resource"].(*url.URL)
	if !ok || resource.String() != "https://api.example.com/v2" {
		t.Errorf("resource = %#v, want the parsed URL", got["resource"])
	}
	if request := got["request"].(models.Request); request.Method != "GET" {
		t.Errorf("method = %q, want GET", request.Method)
	}
	if !bound || got["runas"] != member.ID {
		t.Errorf("bound = %v, runas = %v: a new request must be bound as the editor", bound, got["runas"])
	}
}

func TestUpdatePostIt_RefusesAnInvalidOneOffEdit(t *testing.T) {
	db := boardOf(boardOwner)
	db.FindPostItFn = func(uuid.UUID) (*models.PostIts, error) { return oneOff(uuid.New(), nil), nil }
	db.UpdatePostItFn = func(uuid.UUID, map[string]any) error {
		t.Error("stored anyway")
		return nil
	}
	svc := newService(db, nil, nil)

	err := svc.UpdatePostIt(boardOwner, uuid.New(), map[string]any{"resource": "ftp://example.com"})
	if !errors.Is(err, ErrInvalidOneOff) {
		t.Errorf("got %v, want ErrInvalidOneOff", err)
	}
}

func TestUpdatePostIt_KeepsACardTypesRequest(t *testing.T) {
	db := boardOf(boardOwner)
	db.FindPostItFn = func(uuid.UUID) (*models.PostIts, error) {
		card := oneOff(uuid.New(), nil)
		card.WellKnown = "dog_facts"
		return card, nil
	}
	db.UpdatePostItFn = func(uuid.UUID, map[string]any) error {
		t.Error("stored anyway")
		return nil
	}
	svc := newService(db, nil, nil)

	err := svc.UpdatePostIt(boardOwner, uuid.New(), map[string]any{"resource": "https://evil.example"})
	if !errors.Is(err, ErrTemplateFixed) {
		t.Errorf("got %v, want ErrTemplateFixed", err)
	}
}

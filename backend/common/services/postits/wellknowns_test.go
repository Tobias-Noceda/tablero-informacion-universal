package postits

import (
	"errors"
	"testing"

	"github.com/Secreto31126/tesis/common/mocks"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/Secreto31126/tesis/common/services/access"
	"github.com/google/uuid"
)

func TestCreatePostIt_RefusesUnknownAndDeletedWellKnowns(t *testing.T) {
	store := builtins()
	if err := store.DeleteWellKnown("dog_facts"); err != nil {
		t.Fatal(err)
	}

	db := boardOf(boardOwner)
	db.CreatePostItFn = func(p *models.PostIts, _ models.BoardPostItTitle, _ string, _ models.Position) (*models.PostIts, error) {
		return p, nil
	}
	svc := New(db, &mocks.MockCache{}, &mocks.MockExecuter{}, &mocks.MockSecretResolver{}, access.New(nil), store, nil)

	for _, key := range []string{"dog_facts", "nope"} {
		_, err := svc.CreatePostIt(boardOwner, nil, &models.PostIts{Board: uuid.New(), WellKnown: key})
		if !errors.Is(err, ErrUnknownWellKnown) {
			t.Errorf("%s: err = %v, want ErrUnknownWellKnown", key, err)
		}
	}

	if _, err := svc.CreatePostIt(boardOwner, nil, &models.PostIts{Board: uuid.New(), WellKnown: "dolar_oficial"}); err != nil {
		t.Errorf("live well-known refused: %v", err)
	}
}

// Deleting a well-known stops new cards, not the ones already on boards.
func TestSystemParams_StillResolvedForADeletedWellKnown(t *testing.T) {
	store := builtins()
	if err := store.DeleteWellKnown("nasa_apod"); err != nil {
		t.Fatal(err)
	}

	resolver := &mocks.MockSecretResolver{
		ResolveFn: func(scope models.SecretScope, names []string) (map[string]string, error) {
			return map[string]string{"$" + string(models.SystemNasaApiKey): "platform-key"}, nil
		},
	}
	svc := New(&mocks.MockDB{}, &mocks.MockCache{}, &mocks.MockExecuter{}, resolver, access.New(nil), store, nil)

	params, err := svc.systemParams(&models.PostIts{WellKnown: "nasa_apod"})
	if err != nil {
		t.Fatal(err)
	}
	if params["$api_key"] != "platform-key" {
		t.Errorf("params = %v, want the platform key", params)
	}

	// A card whose well-known vanished entirely runs without platform secrets.
	if params, err := svc.systemParams(&models.PostIts{WellKnown: "gone"}); err != nil || params != nil {
		t.Errorf("params = %v, err = %v", params, err)
	}
}

// templateStubs answers Usable from a fixed map of key → definition.
type templateStubs map[string]models.PostitTemplate

func (c templateStubs) Usable(_ models.Principal, _ uuid.UUID, key string) (*models.WellKnown, error) {
	def, ok := c[key]
	if !ok {
		return nil, errors.New("not_found")
	}
	wk := def.AsWellKnown()
	return &wk, nil
}

func TestCreatePostIt_FromPostitTemplate(t *testing.T) {
	key := models.NewPostitTemplateKey()
	templates := templateStubs{key: {
		Key:      key,
		Scope:    models.PostitTemplateScope{Kind: models.TemplateScopeUser, Owner: boardOwner.ID},
		Name:     "Cat fact",
		Title:    models.BoardPostItTitle{Text: "{{fact}}", Vars: true},
		Resource: "https://catfact.ninja/fact",
		Request:  models.Request{Method: "GET", Queries: map[string]string{"max_length": "$max"}},
		Response: "json",
		Query:    map[string]string{"fact": ".fact"},
		Parameters: []models.PostitTemplateParameter{
			{Key: "$max", Label: "Max", Type: models.ParameterNumber},
		},
		Outputs: []models.PostitTemplateOutput{{Key: "fact", Label: "Fact"}},
	}}

	db := boardOf(boardOwner)
	var ptype string
	db.CreatePostItFn = func(p *models.PostIts, _ models.BoardPostItTitle, kind string, _ models.Position) (*models.PostIts, error) {
		ptype = kind
		return p, nil
	}
	svc := New(db, &mocks.MockCache{}, &mocks.MockExecuter{}, &mocks.MockSecretResolver{}, access.New(nil), builtins(), templates)

	postit, err := svc.CreatePostIt(boardOwner, nil, &models.PostIts{Board: uuid.New(), WellKnown: key, Params: map[string]string{"$max": "80"}})
	if err != nil {
		t.Fatal(err)
	}
	if ptype != key || postit.Resource.String() != "https://catfact.ninja/fact" || postit.Params["$max"] != "80" {
		t.Errorf("card: type %q, %+v", ptype, postit)
	}

	// Missing, not usable here, or no templates at all: unknown.
	if _, err := svc.CreatePostIt(boardOwner, nil, &models.PostIts{Board: uuid.New(), WellKnown: models.NewPostitTemplateKey()}); !errors.Is(err, ErrUnknownWellKnown) {
		t.Errorf("unusable: err = %v", err)
	}
	none := New(db, &mocks.MockCache{}, &mocks.MockExecuter{}, &mocks.MockSecretResolver{}, access.New(nil), builtins(), nil)
	if _, err := none.CreatePostIt(boardOwner, nil, &models.PostIts{Board: uuid.New(), WellKnown: key}); !errors.Is(err, ErrUnknownWellKnown) {
		t.Errorf("without templates: err = %v", err)
	}
}

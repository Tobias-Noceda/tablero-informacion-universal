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
	svc := New(db, &mocks.MockCache{}, &mocks.MockExecuter{}, &mocks.MockSecretResolver{}, access.New(nil), store)

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
	svc := New(&mocks.MockDB{}, &mocks.MockCache{}, &mocks.MockExecuter{}, resolver, access.New(nil), store)

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

package wellknowns

import (
	"errors"
	"fmt"
	"time"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/models"
)

var (
	ErrNotFound   = errors.New("not_found")
	ErrNotDeleted = errors.New("not_deleted")
	ErrInvalid    = errors.New("invalid_well_known")
	ErrForbidden  = errors.New("forbidden")
)

// Service is the catalogue of well-known cards: anyone signed in reads it,
// only platform admins change it.
type Service struct {
	store infrastructure.WellKnownStore
	now   func() time.Time
}

func New(store infrastructure.WellKnownStore) *Service {
	return &Service{store: store, now: time.Now}
}

// Seed stores every built-in definition not stored yet. Safe on every boot.
func (srv *Service) Seed() error {
	return srv.store.SeedWellKnowns(Builtin())
}

func (srv *Service) List(principal models.Principal) ([]models.WellKnown, error) {
	wks, err := srv.store.ListWellKnowns()
	if err != nil {
		return nil, err
	}

	for i := range wks {
		redact(principal, &wks[i])
	}
	return wks, nil
}

func (srv *Service) Get(principal models.Principal, key string) (*models.WellKnown, error) {
	wk, err := srv.find(key)
	if err != nil {
		return nil, err
	}
	if wk.Deleted {
		return nil, ErrNotFound
	}

	redact(principal, wk)
	return wk, nil
}

// Put creates or replaces the definition at key, reviving it if deleted.
// Existing cards keep the copy of the template they were created with.
func (srv *Service) Put(principal models.Principal, key string, wk models.WellKnown) (*models.WellKnown, error) {
	if !principal.Admin {
		return nil, ErrForbidden
	}

	wk.Key = key
	if err := wk.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalid, err)
	}

	existing, err := srv.find(key)
	switch {
	case err == nil:
		wk.Builtin = existing.Builtin
	case errors.Is(err, ErrNotFound):
		wk.Builtin = false
	default:
		return nil, err
	}

	wk.Deleted = false
	wk.UpdatedAt = srv.now().UTC()
	wk.UpdatedBy = principal.ID

	if err := srv.store.PutWellKnown(&wk); err != nil {
		return nil, err
	}
	return &wk, nil
}

// Delete hides the definition from the picker and new cards. It stays stored,
// so the seed does not bring it back and cards made from it keep running.
func (srv *Service) Delete(principal models.Principal, key string) error {
	if !principal.Admin {
		return ErrForbidden
	}

	if err := srv.store.DeleteWellKnown(key); err != nil {
		if errors.Is(err, infrastructure.ErrWellKnownNotFound) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

// Restore brings a deleted definition back as it was.
func (srv *Service) Restore(principal models.Principal, key string) (*models.WellKnown, error) {
	if !principal.Admin {
		return nil, ErrForbidden
	}

	switch err := srv.store.RestoreWellKnown(key); {
	case errors.Is(err, infrastructure.ErrWellKnownNotFound):
		return nil, ErrNotFound
	case errors.Is(err, infrastructure.ErrWellKnownNotDeleted):
		return nil, ErrNotDeleted
	case err != nil:
		return nil, err
	}

	return srv.Get(principal, key)
}

func (srv *Service) find(key string) (*models.WellKnown, error) {
	wk, err := srv.store.FindWellKnown(key)
	if errors.Is(err, infrastructure.ErrWellKnownNotFound) {
		return nil, ErrNotFound
	}
	return wk, err
}

// redact hides from non-admins which platform credentials a card uses, as
// the post-its service does when one is missing.
func redact(principal models.Principal, wk *models.WellKnown) {
	if principal.Admin {
		return
	}
	wk.SystemSecrets = nil
	wk.UpdatedBy = ""
}

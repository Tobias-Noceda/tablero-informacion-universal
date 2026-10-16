package users

import (
	"errors"
	"strings"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

var (
	ErrForbidden       = infrastructure.ErrForbidden
	ErrInvalidName     = errors.New("invalid name")
	ErrInvalidLocale   = errors.New("invalid_locale")
	ErrNothingToUpdate = errors.New("no fields to update")
)

type UserService struct {
	users infrastructure.UserStore
}

func New(users infrastructure.UserStore) *UserService {
	return &UserService{users}
}

func (srv *UserService) Profile(principal models.Principal, id uuid.UUID) (*models.Profile, error) {
	user, err := srv.self(principal, id)
	if err != nil {
		return nil, err
	}

	profile := user.Profile()
	return &profile, nil
}

func (srv *UserService) Summary(principal models.Principal, id uuid.UUID) (*models.UserSummary, error) {
	if principal.Anonymous() {
		return nil, ErrForbidden
	}

	user, err := srv.users.FindUser(id)
	if err != nil {
		return nil, err
	}

	summary := user.Summary()
	return &summary, nil
}

// Changes is what a user may edit about themselves; nil leaves a field as
// it is.
type Changes struct {
	Name   *string
	Locale *string
}

// Update validates every change before applying any.
func (srv *UserService) Update(principal models.Principal, id uuid.UUID, changes Changes) (*models.Profile, error) {
	set := map[string]any{}
	if changes.Name != nil {
		name := strings.TrimSpace(*changes.Name)
		if name == "" {
			return nil, ErrInvalidName
		}
		set["name"] = name
	}
	if changes.Locale != nil {
		locale, ok := models.SupportedLocale(*changes.Locale)
		if !ok {
			return nil, ErrInvalidLocale
		}
		set["locale"] = locale
	}
	if len(set) == 0 {
		return nil, ErrNothingToUpdate
	}

	user, err := srv.self(principal, id)
	if err != nil {
		return nil, err
	}
	if err := srv.users.UpdateUser(user.Id, set); err != nil {
		return nil, err
	}
	if name, ok := set["name"].(string); ok {
		user.Name = name
	}
	if locale, ok := set["locale"].(models.Locale); ok {
		user.Locale = locale
	}

	profile := user.Profile()
	return &profile, nil
}

// self loads the caller's own record; anyone else's is Forbidden, and a
// token whose user is gone is treated the same way.
func (srv *UserService) self(principal models.Principal, id uuid.UUID) (*models.User, error) {
	if principal.Anonymous() || principal.ID != id.String() {
		return nil, ErrForbidden
	}

	user, err := srv.users.FindUser(id)
	if errors.Is(err, infrastructure.ErrUserNotFound) {
		return nil, ErrForbidden
	}
	if err != nil {
		return nil, err
	}

	return user, nil
}

package users

import (
	"errors"
	"strings"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

var (
	ErrForbidden   = infrastructure.ErrForbidden
	ErrInvalidName = errors.New("invalid name")
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

func (srv *UserService) Rename(principal models.Principal, id uuid.UUID, name string) (*models.Profile, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrInvalidName
	}

	user, err := srv.self(principal, id)
	if err != nil {
		return nil, err
	}
	if err := srv.users.UpdateUser(user.Id, map[string]any{"name": name}); err != nil {
		return nil, err
	}
	user.Name = name

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

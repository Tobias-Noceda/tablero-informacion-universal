package entitlements

import (
	"errors"
	"time"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

// Counter counts what a user holds of the limits that are theirs alone.
type Counter interface {
	CountOwnedOrgs(userID string) (int64, error)
	CountOwnedGroups(userID string) (int64, error)
	CountSecretsBilledTo(userID string) (int64, error)
	CountPostitTemplates(scope models.PostitTemplateScope) (int64, error)
}

// Service resolves who pays for something and what their plan allows. The
// plan is read from the user on every call, so a change (a payment, an
// expiry, an admin's override) applies to the next request.
type Service struct {
	users  infrastructure.UserReader
	orgs   infrastructure.OrgReader
	boards infrastructure.BoardReader
	groups infrastructure.GroupReader
	counts Counter
	now    func() time.Time
}

var _ infrastructure.Entitlements = (*Service)(nil)

func New(users infrastructure.UserReader, orgs infrastructure.OrgReader, boards infrastructure.BoardReader, groups infrastructure.GroupReader, counts Counter, now func() time.Time) *Service {
	return &Service{users, orgs, boards, groups, counts, now}
}

// free is what an account that cannot be found is on: nothing it owned is
// removed, it just cannot grow.
func free(payer string) models.Entitlement {
	return models.EntitlementOf(payer, models.PlanFree)
}

func (srv *Service) ForUser(id string) (models.Entitlement, error) {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return free(id), nil
	}

	user, err := srv.users.FindUser(parsed)
	if errors.Is(err, infrastructure.ErrUserNotFound) {
		return free(id), nil
	}
	if err != nil {
		return models.Entitlement{}, err
	}
	return user.Entitlement(srv.now()), nil
}

// ForBoard bills an organization's board to the organization's payer, and
// to the board's owner when the organization is gone.
func (srv *Service) ForBoard(board *models.Board) (models.Entitlement, error) {
	if board.Org == nil {
		return srv.ForUser(board.Owner)
	}

	org, err := srv.orgs.FindOrg(*board.Org)
	if errors.Is(err, infrastructure.ErrOrgNotFound) {
		return srv.ForUser(board.Owner)
	}
	if err != nil {
		return models.Entitlement{}, err
	}
	if org.Payer() == "" {
		// Nobody to bill on an organization with no owner or admin left: the
		// board's owner pays, as the backfill assigns it.
		return srv.ForUser(board.Owner)
	}
	return srv.ForOrg(org)
}

func (srv *Service) ForOrg(org *models.Org) (models.Entitlement, error) {
	return srv.ForUser(org.Payer())
}

func (srv *Service) ForSecretScope(scope models.SecretScope) (models.Entitlement, error) {
	switch scope.Kind {
	case models.ScopeUser:
		return srv.ForUser(scope.Owner)
	case models.ScopeMember:
		_, user, _ := scope.Member()
		return srv.ForUser(user)
	case models.ScopeBoard:
		id, err := uuid.Parse(scope.Owner)
		if err != nil {
			return free(""), nil
		}
		board, err := srv.boards.FindBoard(id)
		if err != nil || board == nil {
			return free(""), err
		}
		return srv.ForBoard(board)
	case models.ScopeGroup:
		id, err := uuid.Parse(scope.Owner)
		if err != nil {
			return free(""), nil
		}
		group, err := srv.groups.FindGroup(id)
		if errors.Is(err, infrastructure.ErrGroupNotFound) {
			return free(""), nil
		}
		if err != nil {
			return models.Entitlement{}, err
		}
		return srv.ForUser(group.Owner)
	default:
		// The platform's own secrets: nobody pays, nothing limits them.
		return models.Entitlement{}, nil
	}
}

func (srv *Service) ForTemplateScope(scope models.PostitTemplateScope) (models.Entitlement, error) {
	switch scope.Kind {
	case models.TemplateScopeBoard:
		id, err := uuid.Parse(scope.Owner)
		if err != nil {
			return free(""), nil
		}
		board, err := srv.boards.FindBoard(id)
		if err != nil || board == nil {
			return free(""), err
		}
		return srv.ForBoard(board)
	case models.TemplateScopeOrg:
		id, err := uuid.Parse(scope.Owner)
		if err != nil {
			return free(""), nil
		}
		org, err := srv.orgs.FindOrg(id)
		if errors.Is(err, infrastructure.ErrOrgNotFound) {
			return free(""), nil
		}
		if err != nil {
			return models.Entitlement{}, err
		}
		return srv.ForOrg(org)
	default:
		return srv.ForUser(scope.Owner)
	}
}

func (srv *Service) SecretsLeft(payer models.Entitlement) (int, error) {
	used, err := srv.counts.CountSecretsBilledTo(payer.Payer)
	if err != nil {
		return 0, err
	}
	return max(payer.Limits.Secrets-int(used), 0), nil
}

func (srv *Service) Usage(user string) (models.Usage, error) {
	var usage models.Usage

	templates := func(userID string) (int64, error) {
		return srv.counts.CountPostitTemplates(models.PostitTemplateScope{Kind: models.TemplateScopeUser, Owner: userID})
	}
	for _, count := range []struct {
		of   func(string) (int64, error)
		into *int
	}{
		{srv.counts.CountOwnedOrgs, &usage.Orgs},
		{srv.counts.CountOwnedGroups, &usage.Groups},
		{srv.counts.CountSecretsBilledTo, &usage.Secrets},
		{templates, &usage.UserTemplates},
	} {
		n, err := count.of(user)
		if err != nil {
			return models.Usage{}, err
		}
		*count.into = int(n)
	}

	return usage, nil
}

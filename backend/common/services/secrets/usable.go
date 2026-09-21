package secrets

import (
	"errors"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

var (
	ErrUnknownCredential = infrastructure.ErrUnknownCredential
	ErrNotShareable      = errors.New("The platform's secrets cannot be shared")
	ErrInvalidGrant      = errors.New("Invalid grant")
)

// ResolveAs decrypts every ref principal may use on board. A ref nobody
// stored is absent from the result; one the principal may not use is an
// error, so a card whose owner lost access stops instead of running blind.
func (srv *SecretsService) ResolveAs(principal models.Principal, board uuid.UUID, refs []models.SecretRef) (map[models.SecretRef]string, error) {
	stored, err := srv.findRefs(refs)
	if err != nil {
		return nil, err
	}

	resolved := make(map[models.SecretRef]string, len(stored))
	for _, s := range stored {
		if err := srv.policy.CanUse(principal, board, &s); err != nil {
			return nil, err
		}

		value, err := srv.present(&s)
		if err != nil {
			return nil, err
		}
		resolved[models.SecretRef{Scope: s.Scope, Name: s.Name}] = value
	}

	return resolved, nil
}

// CanBind checks, without decrypting anything, that every ref exists and
// that principal may use it on board.
func (srv *SecretsService) CanBind(principal models.Principal, board uuid.UUID, refs []models.SecretRef) error {
	stored, err := srv.findRefs(refs)
	if err != nil {
		return err
	}

	found := make(map[models.SecretRef]*models.Secret, len(stored))
	for i := range stored {
		found[models.SecretRef{Scope: stored[i].Scope, Name: stored[i].Name}] = &stored[i]
	}

	for _, ref := range refs {
		s, ok := found[ref]
		if !ok {
			return ErrUnknownCredential
		}
		if err := srv.policy.CanUse(principal, board, s); err != nil {
			return err
		}
	}

	return nil
}

// ListUsable is everything principal could bind into a card on board: what
// the board shares, what they keep there for themselves, their own profile,
// the groups they belong to, and whatever was shared with any of those.
func (srv *SecretsService) ListUsable(principal models.Principal, board uuid.UUID) ([]models.SecretMeta, error) {
	if err := srv.policy.CanView(principal, models.BoardScope(board)); err != nil {
		return nil, err
	}

	scopes := []models.SecretScope{
		models.BoardScope(board),
		models.MemberScope(board, principal.ID),
		models.UserScope(principal.ID),
	}
	audiences := []models.Audience{
		{Kind: models.AudienceUser, ID: principal.ID},
		{Kind: models.AudienceBoard, ID: board.String()},
	}

	groups, err := srv.groups.FindUserGroups(principal.ID)
	if err != nil {
		return nil, err
	}
	for _, group := range groups {
		scopes = append(scopes, models.GroupScope(group.Id))
		audiences = append(audiences, models.Audience{Kind: models.AudienceGroup, ID: group.Id.String()})
	}

	var candidates []models.Secret
	for _, scope := range scopes {
		stored, err := srv.store.ListSecrets(scope)
		if err != nil {
			return nil, err
		}
		candidates = append(candidates, stored...)
	}

	granted, err := srv.store.FindGranted(audiences)
	if err != nil {
		return nil, err
	}
	candidates = append(candidates, granted...)

	usable := make([]models.SecretMeta, 0, len(candidates))
	seen := make(map[models.SecretRef]struct{}, len(candidates))
	for _, s := range candidates {
		ref := models.SecretRef{Scope: s.Scope, Name: s.Name}
		if _, listed := seen[ref]; listed || srv.policy.CanUse(principal, board, &s) != nil {
			continue
		}
		seen[ref] = struct{}{}
		usable = append(usable, s.Meta())
	}

	return usable, nil
}

// SetGrants replaces who else may bind a secret. Only whoever manages the
// scope may share it, and the platform's own secrets are never shared.
func (srv *SecretsService) SetGrants(scope models.SecretScope, principal models.Principal, name string, grants []models.Grant) error {
	if err := srv.policy.CanManage(principal, scope); err != nil {
		return err
	}

	if scope.Kind == models.ScopeSystem {
		return ErrNotShareable
	}

	for _, grant := range grants {
		if !grant.Valid() {
			return ErrInvalidGrant
		}
	}

	return srv.store.SetGrants(scope, name, grants)
}

func (srv *SecretsService) findRefs(refs []models.SecretRef) ([]models.Secret, error) {
	namesByScope := make(map[models.SecretScope][]string)
	for _, ref := range refs {
		namesByScope[ref.Scope] = append(namesByScope[ref.Scope], ref.Name)
	}

	var stored []models.Secret
	for scope, names := range namesByScope {
		found, err := srv.store.FindSecrets(scope, names)
		if err != nil {
			return nil, err
		}
		stored = append(stored, found...)
	}

	return stored, nil
}

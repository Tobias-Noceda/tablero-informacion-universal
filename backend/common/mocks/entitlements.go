package mocks

import (
	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/models"
)

// StaticEntitlements answers the same plan for everything, or the one in
// Plans for a given payer. Unset, everyone is on max, so a test that is not
// about limits never meets one. Secrets and Used are what SecretsLeft and
// Usage count per payer.
type StaticEntitlements struct {
	Plan    models.Plan
	Plans   map[string]models.Plan
	Secrets map[string]int
	Used    map[string]models.Usage
	// HoldFn sees every Hold; unset, Hold holds nothing.
	HoldFn func(limit, holder string) (func(), error)
}

func (s StaticEntitlements) Hold(limit, holder string) (func(), error) {
	if s.HoldFn != nil {
		return s.HoldFn(limit, holder)
	}
	return func() {}, nil
}

var _ infrastructure.Entitlements = StaticEntitlements{}

func (s StaticEntitlements) of(payer string) models.Entitlement {
	if plan, ok := s.Plans[payer]; ok {
		return models.EntitlementOf(payer, plan)
	}
	if s.Plan == "" {
		return models.EntitlementOf(payer, models.PlanMax)
	}
	return models.EntitlementOf(payer, s.Plan)
}

func (s StaticEntitlements) ForUser(id string) (models.Entitlement, error) {
	return s.of(id), nil
}

// ForBoard bills the board's owner: the fake knows no organizations.
func (s StaticEntitlements) ForBoard(board *models.Board) (models.Entitlement, error) {
	return s.of(board.Owner), nil
}

func (s StaticEntitlements) ForOrg(org *models.Org) (models.Entitlement, error) {
	return s.of(org.Payer()), nil
}

// ForSecretScope bills the scope's owner as written, except a member scope
// (its user) and the system scope (nobody).
func (s StaticEntitlements) ForSecretScope(scope models.SecretScope) (models.Entitlement, error) {
	if scope.Kind == models.ScopeSystem {
		return models.Entitlement{}, nil
	}
	if _, user, ok := scope.Member(); ok {
		return s.of(user), nil
	}
	return s.of(scope.Owner), nil
}

func (s StaticEntitlements) ForTemplateScope(scope models.PostitTemplateScope) (models.Entitlement, error) {
	return s.of(scope.Owner), nil
}

func (s StaticEntitlements) SecretsLeft(payer models.Entitlement) (int, error) {
	return max(payer.Limits.Secrets-s.Secrets[payer.Payer], 0), nil
}

func (s StaticEntitlements) Usage(user string) (models.Usage, error) {
	usage := s.Used[user]
	if usage.Secrets == 0 {
		usage.Secrets = s.Secrets[user]
	}
	return usage, nil
}

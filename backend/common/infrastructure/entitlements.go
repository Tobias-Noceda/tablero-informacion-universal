package infrastructure

import "github.com/Secreto31126/tesis/common/models"

// Entitlements answers what a plan allows, for whoever pays for the thing
// being changed. Nothing is cached and no token carries a plan: every check
// sits on a write, a forced refresh, a cache miss or a personal access
// token, where one lookup is cheap and always current.
type Entitlements interface {
	// ForUser is the user's own plan. An account that is gone is on free.
	ForUser(id string) (models.Entitlement, error)
	// ForBoard is the plan a board runs on: its owner's, or the payer's of
	// the organization it belongs to.
	ForBoard(board *models.Board) (models.Entitlement, error)
	// ForOrg is the plan of the organization's payer.
	ForOrg(org *models.Org) (models.Entitlement, error)
	// ForSecretScope is the plan a secret of scope is billed to: the user's
	// for their own and what they keep on a board, the board's payer for a
	// board's, the owner's for a group's. The system scope has no payer and
	// is never limited.
	ForSecretScope(scope models.SecretScope) (models.Entitlement, error)
	// ForTemplateScope is the plan a post-it template of scope counts on.
	ForTemplateScope(scope models.PostitTemplateScope) (models.Entitlement, error)
	// SecretsLeft is how many more secrets the payer may store, never
	// negative.
	SecretsLeft(payer models.Entitlement) (int, error)
	// Usage counts what user holds of the limits that are theirs alone.
	Usage(user string) (models.Usage, error)
}

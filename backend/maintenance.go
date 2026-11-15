package main

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/Secreto31126/tesis/common/models"
	"github.com/Secreto31126/tesis/common/services/wellknowns"
	"github.com/google/uuid"
)

// keyMaintainer is the slice of the secrets service the maintenance commands
// need, kept as an interface so the dispatch can be tested without a database.
type keyMaintainer interface {
	Rewrap() (int, error)
	RotateScope(scope models.SecretScope) error
	RotateAll() error
}

// billingReconciler is the slice of the billing service the reconcile command
// needs.
type billingReconciler interface {
	Reconcile() (synced, failed int, err error)
}

type maintenanceFlags struct {
	migrate          bool
	rewrapKeys       bool
	rotateKey        string
	rotateAllKeys    bool
	reconcileBilling bool
}

// schema is what migrate needs from the database: its collections, their
// indexes and the well-known catalogue to seed.
type schema interface {
	RenameLegacyCollections() error
	EnsureIndexes() error
	BackfillBilling() error
	infrastructure.WellKnownStore
}

// migrate moves renamed collections to their current name, creates the
// indexes the stores rely on, gives the organizations and secrets stored
// before plans existed someone to bill, and inserts the built-in well-knowns
// not stored yet (edited or deleted ones are left alone). All four are
// idempotent, and the renames go first so the indexes land on the moved
// collections.
func migrate(db schema) error {
	if err := db.RenameLegacyCollections(); err != nil {
		return fmt.Errorf("rename collections: %w", err)
	}
	if err := db.EnsureIndexes(); err != nil {
		return fmt.Errorf("ensure indexes: %w", err)
	}
	if err := db.BackfillBilling(); err != nil {
		return fmt.Errorf("backfill billing: %w", err)
	}
	if err := wellknowns.New(db).Seed(); err != nil {
		return fmt.Errorf("seed well-knowns: %w", err)
	}
	return nil
}

// migrateOnBoot reports whether a plain start should migrate before serving.
// Vercel sets VERCEL=1 on every instance, which starts on each cold start and
// scale-out, so there the build runs `-migrate` once per production deploy
// instead (see vercel.json). Docker and local runs keep migrating at boot.
func migrateOnBoot(getenv func(string) string) bool {
	return getenv("VERCEL") == ""
}

func parseMaintenance(args []string) (*maintenanceFlags, error) {
	flags := &maintenanceFlags{}
	set := flag.NewFlagSet("tesis", flag.ContinueOnError)
	set.SetOutput(io.Discard)
	set.BoolVar(&flags.migrate, "migrate", false, "create the indexes and seed the built-in well-knowns, then exit")
	set.BoolVar(&flags.rewrapKeys, "rewrap-keys", false, "re-wrap every data key under the current master key, then exit")
	set.StringVar(&flags.rotateKey, "rotate-key", "", "rotate the data key of one scope (<kind>:<owner>, or 'system'), then exit")
	set.BoolVar(&flags.rotateAllKeys, "rotate-all-keys", false, "rotate the data key of every scope, then exit")

	set.BoolVar(&flags.reconcileBilling, "reconcile-billing", false, "sync every subscription that is not canceled with Mercado Pago, then exit")

	if err := set.Parse(args); err != nil {
		return nil, err
	}

	return flags, nil
}

// requested reports whether a maintenance command that needs the services was asked for. -migrate
// is not one: it only needs the database and runs before the rest boots.
func (f *maintenanceFlags) requested() bool {
	return f.rewrapKeys || f.rotateKey != "" || f.rotateAllKeys || f.reconcileBilling
}

// run executes the requested maintenance and reports what it did.
func (f *maintenanceFlags) run(secrets keyMaintainer, billing billingReconciler, out io.Writer) error {
	if f.rewrapKeys {
		rewrapped, err := secrets.Rewrap()
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "rewrapped %d data keys\n", rewrapped)
	}

	if f.rotateKey != "" {
		scope, err := parseScope(f.rotateKey)
		if err != nil {
			return err
		}
		if err := secrets.RotateScope(scope); err != nil {
			return err
		}
		fmt.Fprintf(out, "rotated the data key of %s\n", scope.Key())
	}

	if f.rotateAllKeys {
		if err := secrets.RotateAll(); err != nil {
			return err
		}
		fmt.Fprintln(out, "rotated every data key")
	}

	if f.reconcileBilling {
		synced, failed, err := billing.Reconcile()
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "reconciled %d subscriptions, %d failed\n", synced, failed)
		if failed > 0 {
			return fmt.Errorf("%d subscriptions could not be reconciled", failed)
		}
	}

	return nil
}

func parseScope(raw string) (models.SecretScope, error) {
	kind, owner, _ := strings.Cut(raw, ":")

	var scope models.SecretScope
	switch models.ScopeKind(kind) {
	case models.ScopeBoard:
		id, err := uuid.Parse(owner)
		if err != nil {
			return scope, fmt.Errorf("board scope wants a uuid owner, got %q", owner)
		}
		scope = models.BoardScope(id)
	case models.ScopeUser:
		scope = models.UserScope(owner)
	case models.ScopeSystem:
		scope = models.SecretScope{Kind: models.ScopeSystem, Owner: owner}
	default:
		return scope, fmt.Errorf("unknown scope kind %q", kind)
	}

	if !scope.Valid() {
		return scope, fmt.Errorf("invalid scope %q", raw)
	}

	return scope, nil
}

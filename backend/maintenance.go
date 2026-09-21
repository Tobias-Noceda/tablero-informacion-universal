package main

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

// keyMaintainer is the slice of the secrets service the maintenance commands
// need, kept as an interface so the dispatch can be tested without a database.
type keyMaintainer interface {
	Rewrap() (int, error)
	RotateScope(scope models.SecretScope) error
	RotateAll() error
}

type maintenanceFlags struct {
	rewrapKeys    bool
	rotateKey     string
	rotateAllKeys bool
}

func parseMaintenance(args []string) (*maintenanceFlags, error) {
	flags := &maintenanceFlags{}
	set := flag.NewFlagSet("tesis", flag.ContinueOnError)
	set.SetOutput(io.Discard)
	set.BoolVar(&flags.rewrapKeys, "rewrap-keys", false, "re-wrap every data key under the current master key, then exit")
	set.StringVar(&flags.rotateKey, "rotate-key", "", "rotate the data key of one scope (<kind>:<owner>, or 'system'), then exit")
	set.BoolVar(&flags.rotateAllKeys, "rotate-all-keys", false, "rotate the data key of every scope, then exit")

	if err := set.Parse(args); err != nil {
		return nil, err
	}

	return flags, nil
}

func (f *maintenanceFlags) requested() bool {
	return f.rewrapKeys || f.rotateKey != "" || f.rotateAllKeys
}

// run executes the requested maintenance and reports what it did.
func (f *maintenanceFlags) run(secrets keyMaintainer, out io.Writer) error {
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

package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Secreto31126/tesis/common/models"
	"github.com/google/uuid"
)

type maintainerSpy struct {
	rewrapped  bool
	rotated    []models.SecretScope
	rotatedAll bool
}

func (m *maintainerSpy) Rewrap() (int, error) {
	m.rewrapped = true
	return 3, nil
}

func (m *maintainerSpy) RotateScope(scope models.SecretScope) error {
	m.rotated = append(m.rotated, scope)
	return nil
}

func (m *maintainerSpy) RotateAll() error {
	m.rotatedAll = true
	return nil
}

func TestMaintenance_NothingRequestedByDefault(t *testing.T) {
	flags, err := parseMaintenance(nil)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if flags.requested() {
		t.Error("a plain start was taken for a maintenance run")
	}
}

func TestMaintenance_Rewrap(t *testing.T) {
	flags, _ := parseMaintenance([]string{"-rewrap-keys"})
	spy := &maintainerSpy{}
	var out bytes.Buffer

	if err := flags.run(spy, &out); err != nil {
		t.Fatalf("run: %v", err)
	}
	if !spy.rewrapped || !strings.Contains(out.String(), "3") {
		t.Errorf("rewrapped=%v out=%q", spy.rewrapped, out.String())
	}
}

func TestMaintenance_RotateOneScope(t *testing.T) {
	id := uuid.New()
	cases := map[string]models.SecretScope{
		"board:" + id.String(): models.BoardScope(id),
		"user:alice":           models.UserScope("alice"),
		"system":               models.SystemScope,
	}

	for raw, want := range cases {
		flags, _ := parseMaintenance([]string{"-rotate-key", raw})
		spy := &maintainerSpy{}

		if err := flags.run(spy, &bytes.Buffer{}); err != nil {
			t.Fatalf("%s: %v", raw, err)
		}
		if len(spy.rotated) != 1 || spy.rotated[0] != want {
			t.Errorf("%s rotated %v, want %v", raw, spy.rotated, want)
		}
	}
}

func TestMaintenance_RejectsBadScopes(t *testing.T) {
	for _, raw := range []string{"board:nope", "team:x", "user:", "system:x", ""} {
		flags, _ := parseMaintenance([]string{"-rotate-key", raw})
		spy := &maintainerSpy{}

		if err := flags.run(spy, &bytes.Buffer{}); err == nil && raw != "" {
			t.Errorf("%q was accepted", raw)
		}
		if len(spy.rotated) != 0 {
			t.Errorf("%q rotated something", raw)
		}
	}
}

func TestMaintenance_RotateAll(t *testing.T) {
	flags, _ := parseMaintenance([]string{"-rotate-all-keys"})
	spy := &maintainerSpy{}

	if err := flags.run(spy, &bytes.Buffer{}); err != nil {
		t.Fatalf("run: %v", err)
	}
	if !spy.rotatedAll {
		t.Error("RotateAll was not called")
	}
}

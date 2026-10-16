package password

import (
	"strings"
	"testing"
)

func TestArgon2_RoundTrip(t *testing.T) {
	hasher := NewWithParams(Fast)

	encoded, err := hasher.Hash("correct horse")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if !strings.HasPrefix(encoded, "$argon2id$") || strings.Contains(encoded, "correct horse") {
		t.Errorf("encoded = %q", encoded)
	}

	if ok, err := hasher.Compare(encoded, "correct horse"); err != nil || !ok {
		t.Errorf("right password: ok = %v, err = %v", ok, err)
	}
	if ok, err := hasher.Compare(encoded, "wrong"); err != nil || ok {
		t.Errorf("wrong password: ok = %v, err = %v", ok, err)
	}
}

func TestArgon2_SameInputDifferentSalt(t *testing.T) {
	hasher := NewWithParams(Fast)

	a, _ := hasher.Hash("pw")
	b, _ := hasher.Hash("pw")
	if a == b {
		t.Error("two hashes of the same password must differ by salt")
	}
}

func TestArgon2_CompareRejectsGarbage(t *testing.T) {
	hasher := NewWithParams(Fast)

	if ok, err := hasher.Compare("not a hash", "pw"); err == nil || ok {
		t.Errorf("garbage: ok = %v, err = %v", ok, err)
	}
}

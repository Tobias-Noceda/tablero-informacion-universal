package crypto

import (
	"bytes"
	"encoding/base64"
	"strings"
	"testing"
)

func key(b byte) string {
	return base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{b}, 32))
}

func sealer(t *testing.T, raw string) *Sealer {
	t.Helper()
	s, err := NewFromKeys(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return s
}

func TestSeal_RoundTrip(t *testing.T) {
	s := sealer(t, "1:"+key(0xA1))

	sealed, err := s.Seal("board|API_KEY", []byte("super-secret"))
	if err != nil {
		t.Fatalf("seal: %v", err)
	}

	if bytes.Contains(sealed.Ciphertext, []byte("super-secret")) {
		t.Error("plaintext is recoverable from the ciphertext")
	}

	got, err := s.Open("board|API_KEY", sealed)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if string(got) != "super-secret" {
		t.Errorf("got %q, want super-secret", got)
	}
}

func TestOpen_RejectsForeignAAD(t *testing.T) {
	s := sealer(t, "1:"+key(0xA1))

	sealed, err := s.Seal("board-a|API_KEY", []byte("value"))
	if err != nil {
		t.Fatalf("seal: %v", err)
	}

	for _, aad := range []string{"board-b|API_KEY", "board-a|OTHER_NAME", ""} {
		if _, err := s.Open(aad, sealed); err == nil {
			t.Errorf("aad %q opened a ciphertext it does not own", aad)
		}
	}
}

func TestOpen_RejectsTampering(t *testing.T) {
	s := sealer(t, "1:"+key(0xA1))

	sealed, _ := s.Seal("board|K", []byte("value"))
	sealed.Ciphertext[0] ^= 0xFF

	if _, err := s.Open("board|K", sealed); err == nil {
		t.Error("a modified ciphertext was accepted")
	}
}

func TestOpen_ErrorDoesNotLeakCause(t *testing.T) {
	s := sealer(t, "1:"+key(0xA1))
	sealed, _ := s.Seal("board|K", []byte("value"))

	_, err := s.Open("wrong", sealed)
	if err == nil {
		t.Fatal("expected an error")
	}
	for _, leak := range []string{"aad", "nonce", "authentication", "cipher"} {
		if strings.Contains(strings.ToLower(err.Error()), leak) {
			t.Errorf("error mentions %q, which distinguishes failure modes: %v", leak, err)
		}
	}
}

func TestSeal_NonceIsFreshPerCall(t *testing.T) {
	s := sealer(t, "1:"+key(0xA1))

	seen := make(map[string]bool)
	for range 200 {
		sealed, err := s.Seal("board|K", []byte("same plaintext every time"))
		if err != nil {
			t.Fatalf("seal: %v", err)
		}
		n := string(sealed.Nonce)
		if seen[n] {
			t.Fatal("nonce reused, which breaks GCM")
		}
		seen[n] = true
	}
}

func TestRotation_OldVersionsStayReadable(t *testing.T) {
	old := sealer(t, "1:"+key(0xA1))
	sealed, _ := old.Seal("board|K", []byte("written-before-rotation"))

	rotated := sealer(t, "1:"+key(0xA1)+",2:"+key(0xB2))

	got, err := rotated.Open("board|K", sealed)
	if err != nil {
		t.Fatalf("a secret sealed with v1 must still open after adding v2: %v", err)
	}
	if string(got) != "written-before-rotation" {
		t.Errorf("got %q", got)
	}

	if !rotated.NeedsRotation(sealed) {
		t.Error("a v1 secret should be reported as needing rotation")
	}

	fresh, _ := rotated.Seal("board|K", []byte("new"))
	if fresh.KeyVersion != 2 {
		t.Errorf("new secrets sealed with v%d, want the highest version", fresh.KeyVersion)
	}
	if rotated.NeedsRotation(fresh) {
		t.Error("a freshly sealed secret should not need rotation")
	}
}

func TestOpen_UnknownKeyVersion(t *testing.T) {
	s := sealer(t, "2:"+key(0xB2))
	sealed, _ := s.Seal("board|K", []byte("v"))

	retired := sealer(t, "3:"+key(0xC3))
	if _, err := retired.Open("board|K", sealed); err == nil {
		t.Error("expected an error when the sealing key is no longer loaded")
	}
}

func TestParse_Rejects(t *testing.T) {
	short := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{1}, 16))

	cases := map[string]string{
		"empty":            "",
		"blank":            "   ",
		"no version":       key(0xA1),
		"version not int":  "one:" + key(0xA1),
		"not base64":       "1:!!!!",
		"wrong key length": "1:" + short,
		"duplicate":        "1:" + key(0xA1) + ",1:" + key(0xB2),
	}

	for name, raw := range cases {
		if _, err := NewFromKeys(raw); err == nil {
			t.Errorf("%s: expected a parse error, got none", name)
		}
	}
}

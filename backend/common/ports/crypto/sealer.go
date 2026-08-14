package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
	"strings"
)

const MASTER_KEYS_ENV = "SECRETS_MASTER_KEYS"

type Sealed struct {
	Ciphertext []byte
	Nonce      []byte
	KeyVersion int
}

type Sealer struct {
	keys    map[int]cipher.AEAD
	current int
}

func New() (*Sealer, error) {
	return NewFromKeys(os.Getenv(MASTER_KEYS_ENV))
}

func NewFromKeys(raw string) (*Sealer, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, fmt.Errorf("%s is not set", MASTER_KEYS_ENV)
	}

	s := &Sealer{keys: make(map[int]cipher.AEAD)}

	for entry := range strings.SplitSeq(raw, ",") {
		version, key, found := strings.Cut(strings.TrimSpace(entry), ":")
		if !found {
			return nil, fmt.Errorf("malformed key entry, want <version>:<base64>")
		}

		v, err := strconv.Atoi(version)
		if err != nil {
			return nil, fmt.Errorf("key version is not a number: %s", version)
		}

		if _, exists := s.keys[v]; exists {
			return nil, fmt.Errorf("duplicate key version %d", v)
		}

		material, err := base64.StdEncoding.DecodeString(key)
		if err != nil {
			return nil, fmt.Errorf("key %d is not valid base64", v)
		}

		if len(material) != 32 {
			return nil, fmt.Errorf("key %d is %d bytes, want 32", v, len(material))
		}

		block, err := aes.NewCipher(material)
		if err != nil {
			return nil, err
		}

		aead, err := cipher.NewGCM(block)
		if err != nil {
			return nil, err
		}

		s.keys[v] = aead
		if v > s.current {
			s.current = v
		}
	}

	return s, nil
}

func (s *Sealer) Seal(aad string, plaintext []byte) (*Sealed, error) {
	aead, ok := s.keys[s.current]
	if !ok {
		return nil, fmt.Errorf("no key available to seal with")
	}

	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	return &Sealed{
		Ciphertext: aead.Seal(nil, nonce, plaintext, []byte(aad)),
		Nonce:      nonce,
		KeyVersion: s.current,
	}, nil
}

func (s *Sealer) Open(aad string, sealed *Sealed) ([]byte, error) {
	aead, ok := s.keys[sealed.KeyVersion]
	if !ok {
		return nil, fmt.Errorf("no key for version %d, cannot decrypt", sealed.KeyVersion)
	}

	plaintext, err := aead.Open(nil, sealed.Nonce, sealed.Ciphertext, []byte(aad))
	if err != nil {
		return nil, fmt.Errorf("could not decrypt secret")
	}

	return plaintext, nil
}

func (s *Sealer) NeedsRotation(sealed *Sealed) bool {
	return sealed.KeyVersion < s.current
}

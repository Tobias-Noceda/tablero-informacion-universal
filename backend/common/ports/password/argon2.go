package password

import (
	"github.com/Secreto31126/tesis/common/infrastructure"
	"github.com/alexedwards/argon2id"
)

// Default follows the OWASP argon2id baseline; Fast keeps unit tests quick.
var (
	Default = &argon2id.Params{Memory: 64 * 1024, Iterations: 1, Parallelism: 4, SaltLength: 16, KeyLength: 32}
	Fast    = &argon2id.Params{Memory: 8 * 1024, Iterations: 1, Parallelism: 1, SaltLength: 16, KeyLength: 32}
)

type Argon2 struct {
	params *argon2id.Params
}

var _ infrastructure.PasswordHasher = (*Argon2)(nil)

func New() *Argon2 {
	return NewWithParams(Default)
}

func NewWithParams(params *argon2id.Params) *Argon2 {
	return &Argon2{params: params}
}

func (a *Argon2) Hash(password string) (string, error) {
	return argon2id.CreateHash(password, a.params)
}

func (a *Argon2) Compare(encoded, password string) (bool, error) {
	return argon2id.ComparePasswordAndHash(password, encoded)
}

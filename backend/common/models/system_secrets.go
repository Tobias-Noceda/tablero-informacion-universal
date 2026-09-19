package models

// SystemSecretName is a platform-owned secret, referenced from code by name
// and injected into well-known post-its at execution time. Create the secret
// in the system scope first, then add its constant here.
type SystemSecretName string

const (
	SystemNasaApiKey SystemSecretName = "NASA_API_KEY"
)

var KnownSystemSecrets = []SystemSecretName{
	SystemNasaApiKey,
}

// SystemSecretStatus is what an operator sees: whether each name the code
// expects has been provisioned, plus anything stored the code does not know.
type SystemSecretStatus struct {
	Name       string     `json:"name"`
	Kind       SecretKind `json:"kind,omitempty"`
	Configured bool       `json:"configured"`
	Known      bool       `json:"known"`
}

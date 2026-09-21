package secrets

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Secreto31126/tesis/common/models"
)

var ErrProviderNotConfigured = errors.New("This provider is not available yet")

func providerConfig(provider models.OAuthProvider) (models.OAuthProviderConfig, error) {
	config, ok := models.OAuthProviders[provider]
	if !ok {
		return config, fmt.Errorf("Unknown provider")
	}
	return config, nil
}

// PutOAuth2Client stores the platform's own application credential for a
// provider. It is the only path that writes the oauth2_client kind.
func (srv *SecretsService) PutOAuth2Client(principal models.Principal, provider models.OAuthProvider, clientID, clientSecret string) error {
	if err := srv.policy.CanManage(principal, models.SystemScope); err != nil {
		return err
	}

	config, err := providerConfig(provider)
	if err != nil {
		return err
	}

	if clientID == "" || clientSecret == "" {
		return fmt.Errorf("Client id and secret are required")
	}

	plaintext, err := json.Marshal(&models.OAuth2Client{ClientID: clientID, ClientSecret: clientSecret})
	if err != nil {
		return err
	}

	return srv.seal(models.SystemScope, string(config.Credential), models.SecretOAuth2Client, plaintext, clear{})
}

// platformClient loads the application credential for a provider, read
// directly rather than through Resolve so it never shapes into a header.
func (srv *SecretsService) platformClient(config models.OAuthProviderConfig) (*models.OAuth2Client, error) {
	stored, err := srv.store.FindSecrets(models.SystemScope, []string{string(config.Credential)})
	if err != nil {
		return nil, err
	}
	if len(stored) == 0 || stored[0].Kind != models.SecretOAuth2Client {
		return nil, ErrProviderNotConfigured
	}

	plaintext, err := srv.unseal(&stored[0])
	if err != nil {
		return nil, err
	}

	var client models.OAuth2Client
	if err := json.Unmarshal(plaintext, &client); err != nil {
		return nil, err
	}

	return &client, nil
}

// hydrate completes a provider grant with the platform's side of the
// credential. A self-managed credential is returned untouched.
func (srv *SecretsService) hydrate(material *models.OAuth2Material) error {
	if material.Provider == "" {
		return nil
	}

	config, err := providerConfig(material.Provider)
	if err != nil {
		return err
	}

	client, err := srv.platformClient(config)
	if err != nil {
		return err
	}

	material.Hydrate(config, client)
	return nil
}

// Connect starts a consent handshake against the platform's application:
// stores an empty grant for the scope and returns where to send the user.
func (srv *SecretsService) Connect(scope models.SecretScope, principal models.Principal, provider models.OAuthProvider, name, redirectURI string) (string, error) {
	if err := srv.policy.CanManage(principal, scope); err != nil {
		return "", err
	}

	if scope.Kind == models.ScopeSystem {
		return "", fmt.Errorf("The platform itself cannot consent to a provider")
	}

	if !models.ValidSecretName(name) {
		return "", fmt.Errorf("Secret name must match [A-Z][A-Z0-9_]*")
	}

	config, err := providerConfig(provider)
	if err != nil {
		return "", err
	}

	if _, err := srv.platformClient(config); err != nil {
		return "", err
	}

	grant := &models.OAuth2Material{Flow: models.OAuth2AuthorizationCode, Provider: provider}
	if err := srv.sealMaterial(scope, name, grant, false); err != nil {
		return "", err
	}

	return srv.Authorize(scope, principal, name, redirectURI)
}

func (srv *SecretsService) Providers() ([]models.OAuthProviderStatus, error) {
	statuses := make([]models.OAuthProviderStatus, 0, len(models.OAuthProviders))
	for provider, config := range models.OAuthProviders {
		_, err := srv.platformClient(config)
		if err != nil && !errors.Is(err, ErrProviderNotConfigured) {
			return nil, err
		}
		statuses = append(statuses, models.OAuthProviderStatus{Provider: provider, Configured: err == nil})
	}

	return statuses, nil
}

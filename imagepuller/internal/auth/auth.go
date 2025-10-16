package auth

import (
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/edgelesssys/contrast/imagepuller/internal/api"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
)

// InsecureConfig represents the imagepuller's registry authentication configuration.
type InsecureConfig struct {
	Auths              map[string]authn.AuthConfig
	CA                 map[string][]tls.Certificate
	InsecureSkipVerify bool
	ExtraEnv           map[string]string
}

// ReadAuthConfig reads the auth config from the TOML file.
func ReadAuthConfig(log *slog.Logger) (*InsecureConfig, error) {
	data, err := os.ReadFile(api.InsecureConfigPath)
	if errors.Is(err, os.ErrNotExist) {
		log.Info("Imagepuller auth config file does not exist. Authenticated pulls are not available")
		return &InsecureConfig{}, nil
	} else if err != nil {
		return nil, fmt.Errorf("reading insecure config file %s: %w", api.InsecureConfigPath, err)
	}

	var cfg InsecureConfig
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing insecure config TOML: %w", err)
	}
	log.Info("Found and parsed imagepuller auth config")

	return &cfg, nil
}

// AuthenticatorFor returns the configured authenticator for the given image's registry.
func (c *InsecureConfig) AuthenticatorFor(imageRef string) (authn.Authenticator, *http.Transport, error) {
	authenticator := authn.Anonymous
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: c.InsecureSkipVerify},
	}

	ref, err := name.ParseReference(imageRef)
	if err != nil {
		return nil, nil, err
	}
	registry := ref.Context().RegistryStr()

	if authConfig, ok := c.Auths[registry]; ok {
		authenticator = authn.FromConfig(authConfig)
	}

	if certs, ok := c.CA[registry]; ok {
		transport.TLSClientConfig.Certificates = certs
	}

	return authenticator, transport, nil
}

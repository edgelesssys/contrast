// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package auth

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/edgelesssys/contrast/imagepuller/internal/imagepullapi"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/pelletier/go-toml/v2"
)

// Config represents the imagepuller's registry authentication configurations.
type Config struct {
	Registries map[string]Registry `toml:"registries"`
	ExtraEnv   map[string]string   `toml:"extra-env"`
}

// Registry represents authentication configuration for a single registry.
type Registry struct {
	authn.AuthConfig
	CACerts            string `toml:"ca-certs"`
	InsecureSkipVerify bool   `toml:"insecure-skip-verify"`
}

// ReadInsecureConfig reads the auth config from the specified TOML file.
// No integrity checks are performed on the device from which the initdata-processor originally read this config.
// An attacker with k8s admin privileges could thus change the config.
func ReadInsecureConfig(path string, log *slog.Logger) (*Config, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) || len(data) == 0 {
		log.Info("Imagepuller auth config file does not exist or is empty. Authenticated pulls are not available")
		return &Config{}, nil
	} else if err != nil {
		return nil, fmt.Errorf("reading insecure config file %q: %w", imagepullapi.InsecureConfigPath, err)
	}

	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing insecure config TOML: %w", err)
	}
	log.Info("Found and parsed imagepuller auth config")

	return &cfg, nil
}

var errUnparseableRef = errors.New("could not parse image ref")

// AuthTransportFor constructs the appropriate http.Transport and authn.Authenticator for the given image's registry.
func (c *Config) AuthTransportFor(imageRef string) (authn.Authenticator, *http.Transport, error) {
	// Note: this does no check image pinning.
	ref, err := name.ParseReference(imageRef)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %w", errUnparseableRef, err)
	}
	registry := c.registryFor(ref.Context().RegistryStr())

	authenticator := authn.Anonymous
	if registry.AuthConfig != (authn.AuthConfig{}) {
		authenticator = authn.FromConfig(registry.AuthConfig)
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: registry.InsecureSkipVerify},
		Proxy:           http.ProxyFromEnvironment,
	}
	if registry.CACerts != "" {
		certpool := x509.NewCertPool()
		certpool.AppendCertsFromPEM([]byte(registry.CACerts))
		transport.TLSClientConfig.RootCAs = certpool
	}

	return authenticator, transport, nil
}

// ApplyEnvVars applies the envvar-based proxy configuration in ExtraEnv.
func (c *Config) ApplyEnvVars() {
	allowedEnvVars := []string{"HTTP_PROXY", "HTTPS_PROXY", "NO_PROXY"}
	for _, env := range allowedEnvVars {
		if value, ok := c.ExtraEnv[env]; ok {
			os.Setenv(env, value)
		}
	}
}

// registryFor returns the registry, if any, for the given registry name.
func (c *Config) registryFor(name string) Registry {
	var registry Registry
	maxMatchingLabels := -1 // "." has 0 matching labels.
	for fqdn, registryCandidate := range c.Registries {
		name = strings.TrimSuffix(name, ".")
		fqdn = strings.TrimSuffix(fqdn, ".")
		matchingLabels := strings.Count(fqdn, ".")
		if matchingLabels > maxMatchingLabels && strings.HasSuffix(name, fqdn) {
			maxMatchingLabels = matchingLabels
			registry = registryCandidate
		}
	}
	return registry
}

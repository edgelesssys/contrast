package runners

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/edgelesssys/contrast/internal/atls"
	"github.com/edgelesssys/contrast/internal/attestation/certcache"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/sdk"
)

type ATLS struct {
	addr       string
	validators []atls.Validator
}

func NewATLS(addr string) *ATLS {
	return &ATLS{addr: addr, validators: nil}
}

func NewATLSWithValidators(addr string, manifestPath string) (*ATLS, error) {
	validators, err := validatorFromManifest(manifestPath)
	if err != nil {
		return nil, err
	}
	return &ATLS{addr: addr, validators: validators}, nil
}

func (a *ATLS) Run(ctx context.Context) error {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("creating ephemeral private key: %w", err)
	}
	cfg, err := atls.CreateAttestationClientTLSConfig(ctx, nil, a.validators, key)
	if err != nil {
		return fmt.Errorf("creating aTLS config: %w", err)
	}
	dialer := tls.Dialer{Config: cfg}
	conn, err := dialer.DialContext(ctx, "tcp4", a.addr)
	if err != nil {
		return fmt.Errorf("dialing %q: %w", a.addr, err)
	}
	return conn.Close()
}

func validatorFromManifest(p string) ([]atls.Validator, error) {
	manifestBytes, err := os.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("reading manifest file %q: %w", p, err)
	}

	var m manifest.Manifest
	if err := json.Unmarshal(manifestBytes, &m); err != nil {
		return nil, fmt.Errorf("parsing manifest: %w", err)
	}

	logger := slog.New(slog.DiscardHandler)
	getter := certcache.NewCachedHTTPSGetter(&noStore{}, certcache.NeverGCTicker, logger)

	return sdk.ValidatorsFromManifest(getter, &m, logger)
}

type noStore struct{}

func (*noStore) Get(key string) ([]byte, bool) {
	return nil, false
}

func (*noStore) Set(key string, value []byte) {
}

func (*noStore) Clear() {
}

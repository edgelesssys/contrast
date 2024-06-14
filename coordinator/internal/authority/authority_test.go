// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package authority

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"github.com/edgelesssys/contrast/coordinator/history"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/google/go-sev-guest/proto/sevsnp"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	manifestGenerationExpected = `
# HELP contrast_coordinator_manifest_generation Current manifest generation.
# TYPE contrast_coordinator_manifest_generation gauge
contrast_coordinator_manifest_generation %d
`
)

var keyDigest = manifest.HexString("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")

func TestEmptyAuthority(t *testing.T) {
	a, reg := newAuthority(t)

	// A fresh authority does not have a signing key, so this should fail.
	manifests, ca, err := a.getManifestsAndLatestCA()
	assert.Error(t, err)
	assert.Nil(t, ca)
	assert.Empty(t, manifests)

	manifest, err := a.latestManifest()
	assert.Error(t, err)
	assert.Nil(t, manifest)

	requireGauge(t, reg, 0)
}

func TestSetManifest(t *testing.T) {
	require := require.New(t)
	a, reg := newAuthority(t)
	expected, mnfstBytes, policies := newManifest(t)

	mnfst, err := a.latestManifest()
	require.ErrorIs(err, ErrNoManifest)
	require.Nil(mnfst)

	ca, err := a.setManifest(mnfstBytes, policies)
	require.NoError(err)
	require.NotNil(ca)
	requireGauge(t, reg, 1)

	actual, err := a.latestManifest()
	require.NoError(err)
	require.NotNil(actual)

	require.Equal(expected, actual)

	// Simulate manifest updates that this instance is not aware of by deleting its state.
	a.state.Store(nil)

	_, err = a.setManifest(mnfstBytes, policies)
	require.NoError(err)
	requireGauge(t, reg, 2)
}

func TestSetManifest_TooFewPolicies(t *testing.T) {
	require := require.New(t)
	a, reg := newAuthority(t)
	_, mnfstBytes, _ := newManifest(t)

	ca, err := a.setManifest(mnfstBytes, [][]byte{})
	require.Error(err)
	require.Nil(ca)
	requireGauge(t, reg, 0)
}

func TestSetManifest_BadManifest(t *testing.T) {
	require := require.New(t)
	a, reg := newAuthority(t)
	_, _, policies := newManifest(t)

	ca, err := a.setManifest([]byte(`{ "policies": 1 }`), policies)
	require.Error(err)
	require.Nil(ca)
	requireGauge(t, reg, 0)
}

func TestGetManifestsAndLatestCA(t *testing.T) {
	require := require.New(t)
	a, reg := newAuthority(t)
	originalManifest, mnfstBytes, policies := newManifest(t)

	manifests, ca, err := a.getManifestsAndLatestCA()
	require.ErrorIs(err, ErrNoManifest)
	require.Empty(manifests)
	require.Nil(ca)

	oldCA, err := a.setManifest(mnfstBytes, policies)
	require.NoError(err)
	require.NotNil(oldCA)
	requireGauge(t, reg, 1)

	alteredManifest := *originalManifest
	alteredManifest.WorkloadOwnerKeyDigests = nil
	alteredManifestBytes, err := json.Marshal(alteredManifest)
	require.NoError(err)

	expectedCA, err := a.setManifest(alteredManifestBytes, policies)
	require.NoError(err)
	require.NotNil(expectedCA)
	requireGauge(t, reg, 2)

	require.NotEqual(expectedCA.GetMeshCACert(), oldCA.GetMeshCACert())

	expectedManifests := []*manifest.Manifest{originalManifest, &alteredManifest}

	manifests, currentCA, err := a.getManifestsAndLatestCA()
	require.NoError(err)
	require.Equal(expectedCA.GetMeshCACert(), currentCA.GetMeshCACert())
	require.Equal(expectedCA.GetRootCACert(), currentCA.GetRootCACert())
	require.Equal(expectedManifests, manifests)

	// Simulate manifest updates that this instance is not aware of by deleting its state.
	a.state.Store(nil)

	manifests, _, err = a.getManifestsAndLatestCA()
	require.NoError(err)
	require.Equal(expectedManifests, manifests)
	requireGauge(t, reg, len(expectedManifests))
}

func TestSNPValidateOpts(t *testing.T) {
	require := require.New(t)
	a, _ := newAuthority(t)
	_, mnfstBytes, policies := newManifest(t)
	policyHash := sha256.Sum256(policies[0])
	report := &sevsnp.Report{HostData: policyHash[:]}

	opts, err := a.SNPValidateOpts(report)
	require.Error(err)
	require.Nil(opts)

	_, err = a.setManifest(mnfstBytes, policies)
	require.NoError(err)

	opts, err = a.SNPValidateOpts(report)
	require.NoError(err)
	require.NotNil(opts)

	// Change to unknown policy hash in HostData.
	report.HostData[0]++

	opts, err = a.SNPValidateOpts(report)
	require.Error(err)
	require.Nil(opts)
}

// TODO(burgerdev): test ValidateCallback and GetCertBundle

func newAuthority(t *testing.T) (*Authority, *prometheus.Registry) {
	t.Helper()
	fs := afero.NewBasePathFs(afero.NewOsFs(), t.TempDir())
	store := history.NewAferoStore(&afero.Afero{Fs: fs})
	hist := history.NewWithStore(store)
	reg := prometheus.NewRegistry()
	return New(hist, reg, slog.Default()), reg
}

func newManifest(t *testing.T) (*manifest.Manifest, []byte, [][]byte) {
	t.Helper()
	policy := []byte("=== SOME REGO HERE ===")
	policyHash := sha256.Sum256(policy)
	policyHashHex := manifest.NewHexString(policyHash[:])
	mnfst := &manifest.Manifest{
		Policies:                map[manifest.HexString][]string{policyHashHex: {"test"}},
		WorkloadOwnerKeyDigests: []manifest.HexString{keyDigest},
	}
	mnfstBytes, err := json.Marshal(mnfst)
	require.NoError(t, err)
	return mnfst, mnfstBytes, [][]byte{policy}
}

func requireGauge(t *testing.T, reg *prometheus.Registry, val int) {
	t.Helper()

	expected := fmt.Sprintf(manifestGenerationExpected, val)
	require.NoError(t, testutil.GatherAndCompare(reg, strings.NewReader(expected), "contrast_coordinator_manifest_generation"))
}

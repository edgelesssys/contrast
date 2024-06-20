// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package authority

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"github.com/edgelesssys/contrast/coordinator/history"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/userapi"
	"github.com/google/go-sev-guest/proto/sevsnp"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/spf13/afero"
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

func TestSNPValidateOpts(t *testing.T) {
	require := require.New(t)
	a, _ := newAuthority(t)
	_, mnfstBytes, policies := newManifest(t)
	policyHash := sha256.Sum256(policies[0])
	report := &sevsnp.Report{HostData: policyHash[:]}

	opts, err := a.SNPValidateOpts(report)
	require.Error(err)
	require.Nil(opts)

	req := &userapi.SetManifestRequest{
		Manifest: mnfstBytes,
		Policies: policies,
	}
	_, err = a.SetManifest(context.Background(), req)
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

	mnfst := manifest.DefaultAKS()
	mnfst.Policies = map[manifest.HexString][]string{policyHashHex: {"test"}}
	mnfst.WorkloadOwnerKeyDigests = []manifest.HexString{keyDigest}
	mnfstBytes, err := json.Marshal(mnfst)
	require.NoError(t, err)
	return &mnfst, mnfstBytes, [][]byte{policy}
}

func requireGauge(t *testing.T, reg *prometheus.Registry, val int) {
	t.Helper()

	expected := fmt.Sprintf(manifestGenerationExpected, val)
	require.NoError(t, testutil.GatherAndCompare(reg, strings.NewReader(expected), "contrast_coordinator_manifest_generation"))
}

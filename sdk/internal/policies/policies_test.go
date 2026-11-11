// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package policies

import (
	"encoding/json"
	"testing"

	"github.com/edgelesssys/contrast/internal/initdata"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckReferenced(t *testing.T) {
	newPolicy := func(algorithm, rego string) []byte {
		i, err := initdata.New(algorithm, map[string]string{"policy.rego": rego})
		require.NoError(t, err)
		raw, err := i.Encode()
		require.NoError(t, err)
		return raw
	}
	newManifest := func(policies ...[]byte) []byte {
		m := manifest.Manifest{Policies: make(map[manifest.HexString]manifest.PolicyEntry)}
		for _, p := range policies {
			digest, err := initdata.Raw(p).Digest()
			require.NoError(t, err)
			m.Policies[manifest.NewHexString(digest)] = manifest.PolicyEntry{}
		}
		out, err := json.Marshal(m)
		require.NoError(t, err)
		return out
	}

	policyA := newPolicy("sha256", "package a")
	policyB := newPolicy("sha384", "package b")
	policyC := newPolicy("sha512", "package c")

	for name, tc := range map[string]struct {
		manifests [][]byte
		policies  [][]byte
		wantErr   string
	}{
		"single manifest": {
			manifests: [][]byte{newManifest(policyA, policyB)},
			policies:  [][]byte{policyB, policyA},
		},
		"policies across the history": {
			manifests: [][]byte{newManifest(policyA, policyB), newManifest(policyB, policyC)},
			policies:  [][]byte{policyA, policyB, policyC},
		},
		"no policies": {
			manifests: [][]byte{newManifest()},
		},
		"policy from an old manifest missing": {
			manifests: [][]byte{newManifest(policyA), newManifest(policyB)},
			policies:  [][]byte{policyB},
			wantErr:   "missing",
		},
		"unreferenced policy": {
			manifests: [][]byte{newManifest(policyA)},
			policies:  [][]byte{policyA, policyB},
			wantErr:   "isn't referenced by any manifest",
		},
		"duplicate policy": {
			manifests: [][]byte{newManifest(policyA)},
			policies:  [][]byte{policyA, policyA},
			wantErr:   "more than once",
		},
		"policy that isn't initdata": {
			manifests: [][]byte{newManifest(policyA)},
			policies:  [][]byte{[]byte("package a")},
			wantErr:   "digesting policy 0",
		},
		"malformed manifest": {
			manifests: [][]byte{[]byte("{")},
			wantErr:   "unmarshalling manifest 0",
		},
	} {
		t.Run(name, func(t *testing.T) {
			err := CheckReferenced(tc.manifests, tc.policies)
			if tc.wantErr != "" {
				assert.ErrorContains(t, err, tc.wantErr)
				return
			}
			assert.NoError(t, err)
		})
	}
}

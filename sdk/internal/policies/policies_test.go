// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package policies

import (
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	"github.com/edgelesssys/contrast/internal/history"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckReferenced(t *testing.T) {
	newManifest := func(policies ...[]byte) []byte {
		m := manifest.Manifest{Policies: make(map[manifest.HexString]manifest.PolicyEntry)}
		for _, p := range policies {
			digest := history.Digest(p)
			m.Policies[manifest.NewHexString(digest[:])] = manifest.PolicyEntry{}
		}
		out, err := json.Marshal(m)
		require.NoError(t, err)
		return out
	}

	newUppercaseManifest := func(policy []byte) []byte {
		digest := history.Digest(policy)
		m := manifest.Manifest{Policies: map[manifest.HexString]manifest.PolicyEntry{
			manifest.HexString(strings.ToUpper(hex.EncodeToString(digest[:]))): {},
		}}
		out, err := json.Marshal(m)
		require.NoError(t, err)
		return out
	}

	policyA := []byte("policy a")
	policyB := []byte("policy b")
	policyC := []byte("policy c")

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
		"duplicate policy from mixed-case references": {
			manifests: [][]byte{newUppercaseManifest(policyB), newManifest(policyB)},
			policies:  [][]byte{policyB, policyB},
		},
		"uppercase policy reference": {
			manifests: [][]byte{newUppercaseManifest(policyB)},
			policies:  [][]byte{policyB},
		},
		"malformed policy reference": {
			manifests: [][]byte{[]byte(`{"Policies":{"zz":{}}}`)},
			wantErr:   "decoding policy hash",
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

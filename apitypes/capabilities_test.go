// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package apitypes

import (
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCapabilitiesResponseFieldName pins the wire field name, which is part of the API contract and can't be changed once clients rely on it.
func TestCapabilitiesResponseFieldName(t *testing.T) {
	data, err := json.Marshal(CapabilitiesResponse{APIVersions: []string{APIVersionV1}})
	require.NoError(t, err)

	assert.JSONEq(t, `{"api_versions":["v1"]}`, string(data))
}

// TestCapabilitiesResponseIgnoresUnknownFields pins the rule that lets a Coordinator extend this response without breaking older clients.
func TestCapabilitiesResponseIgnoresUnknownFields(t *testing.T) {
	var resp CapabilitiesResponse
	require.NoError(t, json.Unmarshal([]byte(`{"api_versions":["v1"],"something_new":42}`), &resp))

	assert.Equal(t, []string{APIVersionV1}, resp.APIVersions)
}

// TestCapabilitiesResponseDigestGolden fixes the layout of the capabilities digest.
func TestCapabilitiesResponseDigestGolden(t *testing.T) {
	for name, tc := range map[string]struct {
		versions []string
		want     string
	}{
		"v1":         {versions: []string{"v1"}, want: "6624bc030b20abe7ccc1597dce71e692dc3137df1054e61ca0bb8b4cd93ecb44"},
		"v2 and v1":  {versions: []string{"v2", "v1"}, want: "414bcbf55ae9ac8bbbae826326cad6d21f8113a2b0266aea884c6db47669d8d3"},
		"v1 and v2":  {versions: []string{"v1", "v2"}, want: "ba71720d5a96b866490487aa67cc566a15fb60b48ffa2d2f3f3af1287286693c"},
		"joined":     {versions: []string{"v2v1"}, want: "fc0f59b4e8874702c2c5bdee593667828ef9f77545146bfbae53538cdd107c24"},
		"no version": {want: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},
	} {
		t.Run(name, func(t *testing.T) {
			got := CapabilitiesResponse{APIVersions: tc.versions}.Digest()
			assert.Equal(t, tc.want, hex.EncodeToString(got[:]))
		})
	}
}

// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package apitypes

import (
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

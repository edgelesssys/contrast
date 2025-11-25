// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package httpapi

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	fullResponse = `
{
	"raw_attestation_doc": "Y29udGVudA==",
	"manifests": [
		"bWFuaWZlc3Qx",
		"bWFuaWZlc3Qy"
	],
	"policies": ["cG9saWN5MQ=="],
	"root_ca": "Uk9PVFBFTQ==",
	"mesh_ca": "TUVTSFBFTQ=="
}
`
	badField = `
{
	"policies": "incompatible field type"
}
`
	badFieldWithVersion = `
{
	"version": "999.0.1",
	"policies": "incompatible field type"
}
`
)

func TestResponseDecoding(t *testing.T) {
	for name, tc := range map[string]struct {
		resp           string
		want           *AttestationResponse
		wantErr        bool
		wantErrVersion string
	}{
		"empty response parses correctly": {
			resp: `{}`,
			want: &AttestationResponse{},
		},
		"populated response parses correctly": {
			resp: fullResponse,
			want: &AttestationResponse{
				RawAttestationDoc: []byte("content"),
				CoordinatorState: CoordinatorState{
					Manifests: [][]byte{
						[]byte("manifest1"),
						[]byte("manifest2"),
					},
					Policies: [][]byte{[]byte("policy1")},
					RootCA:   []byte("ROOTPEM"),
					MeshCA:   []byte("MESHPEM"),
				},
			},
		},
		"parsing error contains version, if available": {
			resp:           badFieldWithVersion,
			wantErr:        true,
			wantErrVersion: "999.0.1",
		},
		"parsing error does not contain a version if none is available": {
			resp:    badField,
			wantErr: true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			assert := assert.New(t)

			resp, err := UnmarshalAttestationResponse([]byte(tc.resp))

			if !tc.wantErr {
				assert.NoError(err)
				assert.Equal(tc.want, resp)
				return
			}

			assert.Nil(resp)
			assert.Error(err)
			var targetErr *unmarshalError
			require.ErrorAs(err, &targetErr)
			assert.Equal(tc.wantErrVersion, targetErr.version)
			var jsonErr *json.UnmarshalTypeError
			require.ErrorAs(err, &jsonErr)
			assert.Equal("CoordinatorState.policies", jsonErr.Field)
		})
	}
}

var _ = error(&unmarshalError{})

// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package httpapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
	"version": "999.0.1",
	"policies": "incompatible field type"
}
`
)

func TestResponseDecoding(t *testing.T) {
	for name, tc := range map[string]struct {
		resp    string
		want    *AttestationResponse
		wantErr string
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
		"parsing error contains version": {
			resp:    badField,
			wantErr: "999.0.1",
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			resp, err := UnmarshalAttestationResponse([]byte(tc.resp))

			if tc.wantErr != "" {
				assert.ErrorContains(err, tc.wantErr)
				assert.Nil(resp)
				return
			}

			assert.NoError(err)
			assert.Equal(tc.want, resp)
		})
	}
}

// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package kuberesource

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEncodeDecode(t *testing.T) {
	testCases := []struct {
		name    string
		fixture string
		wantErr bool
	}{
		{
			name: "valid",
			fixture: `apiVersion: v1
kind: Pod
metadata:
  name: foo
spec:
  containers:
  - image: image
    name: bar
`,
			wantErr: false,
		},
		{
			name: "unknown field",
			fixture: `apiVersion: v1
kind: Pod
unknown: field
metadata:
  name: foo
spec:
  containers:
  - image: image
    name: bar
`,
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require := require.New(t)

			resources, err := UnmarshalApplyConfigurations([]byte(tc.fixture))
			if tc.wantErr {
				require.Error(err)
				return
			}
			require.NoError(err)

			got, err := EncodeResources(resources...)
			require.NoError(err)

			require.Equal(tc.fixture, string(got))
		})
	}
}

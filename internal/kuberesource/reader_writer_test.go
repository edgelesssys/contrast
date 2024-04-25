// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package kuberesource

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEncodeDecode(t *testing.T) {
	require := require.New(t)

	fixture := `apiVersion: v1
kind: Pod
metadata:
  name: foo
spec:
  containers:
  - image: image
    name: bar
`

	resources, err := UnmarshalApplyConfigurations([]byte(fixture))
	require.NoError(err)

	got, err := EncodeResources(resources...)
	require.NoError(err)

	require.Equal(fixture, string(got))
}

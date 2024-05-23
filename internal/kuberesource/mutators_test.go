// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package kuberesource

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPatchNamespaces(t *testing.T) {
	coordinator := CoordinatorBundle()
	openssl, err := OpenSSL()
	require.NoError(t, err)
	emojivoto, err := Emojivoto(ServiceMeshIngressEgress)
	require.NoError(t, err)

	for _, tc := range []struct {
		name string
		set  []any
	}{
		{
			name: "coordinator",
			set:  coordinator,
		},
		{
			name: "openssl",
			set:  openssl,
		},
		{
			name: "emojivoto",
			set:  emojivoto,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require := require.New(t)
			expectedNamespace := "right-namespace"
			set := PatchNamespaces(tc.set, expectedNamespace)
			u, err := ResourcesToUnstructured(set)
			require.NoError(err)
			require.NotEmpty(u)
			for _, obj := range u {
				require.Equal(expectedNamespace, obj.GetNamespace())
			}
		})
		t.Run(tc.name+"-empty-namespace", func(t *testing.T) {
			require := require.New(t)
			set := PatchNamespaces(tc.set, "some-namespace")
			set = PatchNamespaces(set, "")
			u, err := ResourcesToUnstructured(set)
			require.NoError(err)
			require.NotEmpty(u)
			for _, obj := range u {
				meta := obj.Object["metadata"].(map[string]any)
				_, ok := meta["namespace"]
				require.False(ok, "namespace should have been deleted")
			}
		})
	}
}

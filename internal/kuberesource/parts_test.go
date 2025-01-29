// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package kuberesource

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewPortForwarder(t *testing.T) {
	require := require.New(t)

	config := PortForwarder("coordinator", "default").
		WithListenPorts([]int32{1313, 7777}).
		WithForwardTarget("coordinator")

	b, err := EncodeResources(config)
	require.NoError(err)
	t.Log("\n" + string(b))
}

func TestCoordinator(t *testing.T) {
	require := require.New(t)

	b, err := EncodeResources(Coordinator("default"))
	require.NoError(err)
	t.Log("\n" + string(b))
}

func TestNoNamespaces(t *testing.T) {
	coordinator := CoordinatorBundle()
	openssl := OpenSSL()
	emojivoto := Emojivoto(ServiceMeshIngressEgress)
	volumeStatefulSet := VolumeStatefulSet()
	mysql := MySQL()

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
		{
			name: "volume-stateful-set",
			set:  volumeStatefulSet,
		},
		{
			name: "mysql",
			set:  mysql,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require := require.New(t)
			u, err := ResourcesToUnstructured(tc.set)
			require.NoError(err)
			require.NotEmpty(u)
			for _, obj := range u {
				metaAny, ok := obj.Object["metadata"]
				require.True(ok)
				meta, ok := metaAny.(map[string]any)
				require.True(ok)
				_, ok = meta["namespace"]
				require.False(ok)
			}
		})
	}
}

package kuberesource

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewPortForwarder(t *testing.T) {
	require := require.New(t)

	config := PortForwarder("coordinator", "default").
		WithListenPort(1313).
		WithForwardTarget("coordinator", 1313)

	b, err := EncodeResources(config)
	require.NoError(err)
	t.Log("\n" + string(b))
}

func TestCoordinator(t *testing.T) {
	require := require.New(t)

	config := Coordinator("default")

	b, err := EncodeResources(config)
	require.NoError(err)
	t.Log("\n" + string(b))
}

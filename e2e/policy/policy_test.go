package policy

import (
	"testing"

	"github.com/edgelesssys/contrast/e2e/internal/contrasttest"
	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/stretchr/testify/require"
)

var imageReplacementsFile, namespaceFile string
var skipUndeploy bool

func TestPolicy(t *testing.T) {
	ct := contrasttest.New(t, imageReplacementsFile, namespaceFile, skipUndeploy)

	// curently this probably is overkill, but its a helpful start
	resources := kuberesource.Emojivoto(kuberesource.ServiceMeshIngressEgress)
	resources = append(resources, kuberesource.CoordinatorBundle()...)
	resources = kuberesource.AddPortForwarders(resources)

	ct.Init(t, resources)

	require.True(t, t.Run("generate", ct.Generate), "contrast generate needs to succeed for subsequent tests")

	require.True(t, t.Run("apply", ct.Apply), "Kubernetes resources need to be applied for subsequent tests")

	require.True(t, t.Run("set", ct.Set), "contrast set needs to succeed for subsequent tests")
	require.True(t, t.Run("contrast verify", ct.Verify), "contrast verify needs to succeed for subsequent tests")
}

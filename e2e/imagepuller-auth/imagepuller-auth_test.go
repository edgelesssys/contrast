// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build e2e

package imagepullerauth

import (
	"context"
	"encoding/base64"
	"flag"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/edgelesssys/contrast/e2e/internal/contrasttest"
	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/platforms"
	"github.com/pelletier/go-toml/v2"
	"github.com/stretchr/testify/require"
)

func TestImagepullerAuth(t *testing.T) {
	platform, err := platforms.FromString(contrasttest.Flags.PlatformStr)
	require.NoError(t, err)
	ct := contrasttest.New(t)

	token := os.Getenv("CONTRAST_GHCR_READ")
	require.NotEmpty(t, token, "environment variable CONTRAST_GHCR_READ must be set with a ghcr token")
	cfg := map[string]any{
		"registries": map[string]any{
			"ghcr.io.": map[string]string{
				"auth": base64.StdEncoding.EncodeToString(fmt.Appendf(nil, "user-not-required-here:%s", token)),
			},
		},
	}
	imagePullerConfig, err := toml.Marshal(cfg)
	require.NoError(t, err)
	ct.NodeInstallerImagePullerConfig = imagePullerConfig

	runtimeHandler, err := manifest.RuntimeHandler(platform)
	require.NoError(t, err)

	resources := kuberesource.CoordinatorBundle()
	deploymentName := "auth-test"
	authTester := kuberesource.AuthenticatedPullTester(deploymentName)
	resources = append(resources, authTester)
	resources = kuberesource.PatchRuntimeHandlers(resources, runtimeHandler)
	resources = kuberesource.AddPortForwarders(resources)

	ct.Init(t, resources)

	require.NoError(t, err)
	require.True(t, t.Run("generate", ct.Generate), "contrast generate needs to succeed for subsequent tests")
	require.True(t, t.Run("apply", ct.Apply), "Kubernetes resources need to be applied for subsequent tests")
	require.True(t, t.Run("set", ct.Set), "contrast set needs to succeed for subsequent tests")
	require.True(t, t.Run("contrast verify", ct.Verify), "contrast verify needs to succeed for subsequent tests")

	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	t.Cleanup(cancel)
	require.NoError(t, ct.Kubeclient.WaitForDeployment(ctx, ct.Namespace, deploymentName))
}

func TestMain(m *testing.M) {
	contrasttest.RegisterFlags()
	flag.Parse()

	os.Exit(m.Run())
}

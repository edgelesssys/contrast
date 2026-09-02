// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

// test-if: path:apitypes
// test-if: path:sdk
// test-if: path:coordinator/internal/httpapi

//go:build e2e

package httpapi

import (
	"context"
	"flag"
	"os"
	"testing"
	"time"

	"github.com/edgelesssys/contrast/e2e/internal/contrasttest"
	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/platforms"
	"github.com/edgelesssys/contrast/sdk/apiv1"
	"github.com/stretchr/testify/require"
)

type apiMode struct {
	name       string
	apiVersion string
	useHTTP    bool
}

var apiModes = []apiMode{
	{name: "grpc"},
	{name: "http-negotiated", useHTTP: true},
	{name: "http-pinned-" + apiv1.Version, useHTTP: true, apiVersion: apiv1.Version},
}

func (a apiMode) run(ctx context.Context, ct *contrasttest.ContrastTest, f func(flags []string) error) error {
	if !a.useHTTP {
		return f(nil)
	}
	return ct.WithForwardedHTTPAPI(ctx, a.apiVersion, f)
}

func TestHTTPAPI(t *testing.T) {
	platform, err := platforms.FromString(contrasttest.Flags.PlatformStr)
	require.NoError(t, err)
	ct := contrasttest.New(t)

	runtimeHandler, err := manifest.RuntimeHandler(platform)
	require.NoError(t, err)

	resources := kuberesource.OpenSSL()
	resources = append(resources, kuberesource.CoordinatorBundle()...)
	resources = kuberesource.PatchRuntimeHandlers(resources, runtimeHandler)
	resources = kuberesource.AddPortForwarders(resources)

	ct.Init(t, resources)

	require.True(t, t.Run("generate", ct.Generate), "contrast generate needs to succeed for subsequent tests")
	require.True(t, t.Run("apply", ct.Apply), "Kubernetes resources need to be applied for subsequent tests")
	require.True(t, t.Run("initial set", ct.Set), "contrast set needs to succeed for subsequent tests")

	for _, mode := range apiModes {
		t.Run(mode.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), ct.FactorPlatformTimeout(5*time.Minute))
			t.Cleanup(cancel)

			// Each subtest re-sets the same manifest, which the Coordinator records as a new transition.
			require.True(t, t.Run("set", func(t *testing.T) {
				require.NoError(t, mode.run(ctx, ct, func(flags []string) error {
					return ct.RunSet(ctx, flags...)
				}))
			}), "set needs to succeed for the remaining subtests of this mode")

			t.Run("verify", ct.Verify)
		})
	}

	t.Run("workload still runs", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), ct.FactorPlatformTimeout(3*time.Minute))
		t.Cleanup(cancel)

		require.NoError(t, ct.Kubeclient.WaitForDeployment(ctx, ct.Namespace, "openssl-backend"))
	})
}

func TestMain(m *testing.M) {
	contrasttest.RegisterFlags()
	flag.Parse()

	os.Exit(m.Run())
}

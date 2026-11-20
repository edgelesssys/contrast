// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

// test-if: path:sdk
// test-if: path:apitypes
// test-if: path:coordinator/internal/httpapi

//go:build e2e

package apiversion

import (
	"context"
	"flag"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/edgelesssys/contrast/apitypes"
	apitypesv1 "github.com/edgelesssys/contrast/apitypes/apiv1"
	"github.com/edgelesssys/contrast/e2e/internal/contrasttest"
	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/platforms"
	"github.com/edgelesssys/contrast/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAPIVersion ensures the SDK speaks both the unversioned API and the versioned API.
func TestAPIVersion(t *testing.T) {
	platform, err := platforms.FromString(contrasttest.Flags.PlatformStr)
	require.NoError(t, err)
	runtimeHandler, err := manifest.RuntimeHandler(platform)
	require.NoError(t, err)

	ct := contrasttest.New(t)

	resources := kuberesource.CoordinatorBundle()
	resources = kuberesource.PatchRuntimeHandlers(resources, runtimeHandler)
	resources = kuberesource.AddPortForwarders(resources)
	ct.Init(t, resources)

	require.True(t, t.Run("generate", ct.Generate), "contrast generate needs to succeed for subsequent tests")
	require.True(t, t.Run("apply", ct.Apply), "Kubernetes resources need to be applied for subsequent tests")
	require.True(t, t.Run("set", ct.Set), "contrast set needs to succeed for subsequent tests")

	for name, tc := range map[string]struct {
		apiVersion string
		configure  func(*sdk.Client) *sdk.Client
		// wantPaths are the HTTP API endpoints the SDK must request, in order.
		wantPaths []string
		// wantTransitionSignature is set if the endpoint reports the latest transition signature.
		wantTransitionSignature bool
	}{
		"unversioned API": {
			wantPaths: []string{apitypesv1.LegacyAttestPath},
		},
		"v1 with a minimum API version": {
			apiVersion:              apitypesv1.Version,
			configure:               func(c *sdk.Client) *sdk.Client { return c.WithMinimumAPIVersion(apitypesv1.Version) },
			wantPaths:               []string{apitypes.CapabilitiesPath, apitypesv1.AttestPath},
			wantTransitionSignature: true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)

			ctx, cancel := context.WithTimeout(t.Context(), ct.FactorPlatformTimeout(2*time.Minute))
			t.Cleanup(cancel)

			recorder := &pathRecorder{next: http.DefaultTransport}
			state, err := ct.AttestViaSDK(ctx, tc.apiVersion, func(c *sdk.Client) *sdk.Client {
				c = c.WithHTTPClient(&http.Client{Transport: recorder})
				if tc.configure != nil {
					c = tc.configure(c)
				}
				return c
			})
			require.NoError(err)

			assert.Equal(t, tc.wantPaths, recorder.get())
			require.NotEmpty(state.Manifests)
			assert.NotEmpty(t, state.RootCA)
			assert.NotEmpty(t, state.MeshCA)
			assert.NotEmpty(t, state.LatestTransitionHash)
			assert.Equal(t, tc.wantTransitionSignature, len(state.LatestTransitionSignature) != 0)
		})
	}
}

// pathRecorder records the paths of the requests sent through it.
type pathRecorder struct {
	next http.RoundTripper

	mu    sync.Mutex
	paths []string
}

func (r *pathRecorder) RoundTrip(req *http.Request) (*http.Response, error) {
	r.mu.Lock()
	r.paths = append(r.paths, req.URL.Path)
	r.mu.Unlock()
	return r.next.RoundTrip(req)
}

func (r *pathRecorder) get() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.paths
}

func TestMain(m *testing.M) {
	contrasttest.RegisterFlags()
	flag.Parse()

	os.Exit(m.Run())
}

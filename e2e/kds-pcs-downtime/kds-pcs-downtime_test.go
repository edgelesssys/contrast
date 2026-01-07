// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build e2e

package kdspcsdowntime

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/edgelesssys/contrast/e2e/internal/contrasttest"
	"github.com/edgelesssys/contrast/e2e/internal/kubeclient"
	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/platforms"
	"github.com/elazarl/goproxy"
	"github.com/stretchr/testify/require"
)

const (
	kdsAddr = "kdsintf.amd.com:443"
	pcsAddr = "api.trustedservices.intel.com:443"
)

func TestKDSPCSDowntime(t *testing.T) {
	platform, err := platforms.FromString(contrasttest.Flags.PlatformStr)
	require.NoError(t, err)
	ct := contrasttest.New(t)

	runtimeHandler, err := manifest.RuntimeHandler(platform)
	require.NoError(t, err)
	resources := kuberesource.CoordinatorBundle()
	resources = kuberesource.PatchRuntimeHandlers(resources, runtimeHandler)
	resources = kuberesource.AddPortForwarders(resources)
	ct.Init(t, resources)

	proxy := goproxy.NewProxyHttpServer()
	server := http.Server{Handler: proxy}
	errCh := make(chan error)

	// If set to true, connections to KDS and PCS will be blocked by the proxy.
	var blockKDSPCS atomic.Bool
	// connectionProxied will be set to true if the proxy performs an HTTP CONNECT to the address of KDS or PCS.
	var connectionProxied atomic.Bool
	proxy.ConnectDial = func(network string, addr string) (net.Conn, error) {
		t.Logf("Proxying connection: %q", addr)
		if (addr == kdsAddr || addr == pcsAddr) && blockKDSPCS.Load() {
			t.Logf("Blocking connection to KDS/PCS %q", addr)
			connectionProxied.Store(true)
			return nil, fmt.Errorf("connection to KDS/PCS %q blocked by test proxy", addr)
		}
		return (&net.Dialer{}).DialContext(t.Context(), network, addr)
	}

	proxyListener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:")
	require.NoError(t, err)

	t.Cleanup(func() {
		require.NoError(t, server.Close())
		err := <-errCh
		require.ErrorIs(t, err, http.ErrServerClosed)
	})

	go func() {
		errCh <- server.Serve(proxyListener)
	}()

	t.Setenv("https_proxy", proxyListener.Addr().String())

	require.True(t, t.Run("generate", ct.Generate), "contrast generate needs to succeed for subsequent tests")
	require.True(t, t.Run("apply", ct.Apply), "Kubernetes resources need to be applied for subsequent tests")

	t.Run("kds downtime", func(t *testing.T) {
		if !platforms.IsSNP(platform) {
			t.Skip("KDS downtime test is only applicable to SEV-SNP workloads")
		}

		require := require.New(t)

		ctx, cancel := context.WithTimeout(t.Context(), ct.FactorPlatformTimeout(3*time.Minute))
		defer cancel()

		require.NoError(ct.Kubeclient.WaitForCoordinator(ctx, ct.Namespace))

		//
		// Look at dev-docs/endorsement-caching.md for table of different cases.
		//

		// Coordinator and CLI cache are empty at the beginning.

		coordinatorPods, err := ct.Kubeclient.PodsFromOwner(ctx, ct.Namespace, "StatefulSet", "coordinator")
		require.NoError(err)
		require.NotEmpty(coordinatorPods, "pod not found: %s/%s", ct.Namespace, "coordinator")

		// Block coordinator access to KDS.
		etcHosts, stderr, err := ct.Kubeclient.Exec(ctx, ct.Namespace, coordinatorPods[0].Name, []string{"/bin/sh", "-c", "cat /etc/hosts"})
		require.NoError(err, "stderr: %q", stderr)
		_, stderr, err = ct.Kubeclient.Exec(ctx, ct.Namespace, coordinatorPods[0].Name, []string{"/bin/sh", "-c", "echo 127.0.0.1 kdsintf.amd.com >> /etc/hosts"})
		require.NoError(err, "stderr: %q", stderr)

		// Block CLI access to KDS.
		blockKDSPCS.Store(true)

		// Set should fail because neither coordinator nor CLI can reach KDS and there is no cached data.
		// Set loop considers context deadline exceeded from KDS as a retriable error.
		// Lower the timeout so the set loop doesn't exceed the test timeout.
		setCtx, setCancel := context.WithTimeout(ctx, ct.FactorPlatformTimeout(1*time.Minute))
		defer setCancel()
		err = ct.RunSet(setCtx)
		t.Logf("Set error: %v", err)
		require.ErrorContains(err, "transport: authentication handshake failed: context deadline exceeded")
		require.True(connectionProxied.Load(), "expected connection to KDS to be proxied")
		connectionProxied.Store(false)

		// Unblock coordinator access to KDS.
		_, stderr, err = ct.Kubeclient.Exec(ctx, ct.Namespace, coordinatorPods[0].Name, []string{"/bin/sh", "-c", fmt.Sprintf("echo '%s' > /etc/hosts", etcHosts)})
		require.NoError(err, "updating /etc/hosts: stderr: %q", stderr)

		// Set should succeed because coordinator can reach KDS.
		require.NoError(ct.RunSet(ctx))

		// Block coordinator access to KDS again.
		_, stderr, err = ct.Kubeclient.Exec(ctx, ct.Namespace, coordinatorPods[0].Name, []string{"/bin/sh", "-c", "echo 127.0.0.1 kdsintf.amd.com >> /etc/hosts"})
		require.NoError(err, "updating /etc/hosts: stderr: %q", stderr)

		// Verify should succeed because certs are now cached by coordinator.
		require.NoError(ct.RunVerify(ctx))

		// Clear coordinator cache by restarting it.
		require.NoError(ct.Kubeclient.Restart(ctx, kubeclient.StatefulSet{}, ct.Namespace, "coordinator"))
		require.NoError(ct.Kubeclient.WaitForCoordinator(ctx, ct.Namespace))

		coordinatorPods, err = ct.Kubeclient.PodsFromOwner(ctx, ct.Namespace, "StatefulSet", "coordinator")
		require.NoError(err)
		require.NotEmpty(coordinatorPods, "pod not found: %s/%s", ct.Namespace, "coordinator")

		// Block coordinator access to KDS.
		_, stderr, err = ct.Kubeclient.Exec(ctx, ct.Namespace, coordinatorPods[0].Name, []string{"/bin/sh", "-c", "echo 127.0.0.1 kdsintf.amd.com >> /etc/hosts"})
		require.NoError(err, "updating /etc/hosts: stderr: %q", stderr)

		// Unblock CLI access to KDS.
		blockKDSPCS.Store(false)

		// Recover should succeed because CLI can reach KDS.
		require.NoError(ct.RunRecover(ctx))

		// Block CLI access to KDS again.
		blockKDSPCS.Store(true)

		// Verify should succeed because CLI has now cached the certs.
		require.NoError(ct.RunVerify(ctx))
	})

	t.Run("pcs downtime", func(t *testing.T) {
		if !platforms.IsTDX(platform) {
			t.Skip("PCS downtime test is only applicable to TDX workloads")
		}

		require := require.New(t)

		ctx, cancel := context.WithTimeout(t.Context(), ct.FactorPlatformTimeout(2*time.Minute))
		defer cancel()

		c := kubeclient.NewForTest(t)

		require.NoError(c.WaitForCoordinator(ctx, ct.Namespace))

		//
		// We can't test PCS downtime on the issuer side, since PCS/PCCS are accessed from the host.
		// Look at dev-docs/endorsement-caching.md for table of different cases.
		//

		// CLI cache is empty at the beginning. Block CLI access to PCS.
		blockKDSPCS.Store(true)

		// Set should fail because the CLI can't reach the PCS and there is no cached data.
		// Set loop considers context deadline exceeded from PCS as a retriable error.
		// Lower the timeout so the set loop doesn't exceed the test timeout.
		setCtx, setCancel := context.WithTimeout(ctx, ct.FactorPlatformTimeout(1*time.Minute))
		defer setCancel()
		err = ct.RunSet(setCtx)
		t.Logf("Set error: %v", err)
		require.ErrorContains(err, "transport: authentication handshake failed: context deadline exceeded")
		require.True(connectionProxied.Load(), "expected connection to PCS to be proxied")
		connectionProxied.Store(false)

		// Unblock CLI access to PCS.
		blockKDSPCS.Store(false)

		// Set should succeed because the CLI can reach PCS.
		require.NoError(ct.RunSet(ctx))

		// Block CLI access to PCS again.
		blockKDSPCS.Store(true)

		// Verify should succeed because collateral is now cached by CLI.
		require.NoError(ct.RunVerify(ctx))
	})
}

func TestMain(m *testing.M) {
	contrasttest.RegisterFlags()
	flag.Parse()

	os.Exit(m.Run())
}

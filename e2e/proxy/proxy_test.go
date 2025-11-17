// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build e2e

// proxy ensures that HTTP_PROXY environment variables are respected by the Contrast CLI.
package proxy

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/edgelesssys/contrast/e2e/internal/contrasttest"
	"github.com/edgelesssys/contrast/e2e/internal/kubeclient"
	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/platforms"
	"github.com/elazarl/goproxy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHTTPProxy ensures that environment variables like HTTP_PROXY are respected by the Contrast
// CLI. It starts a HTTP proxy server and executes the entire Contrast lifecycle, but in a separate
// process configured with the environment variables under test. This is necessary because the
// proxy detection mechanism caches its result [1] and is hard to override with the gRPC API.
//
// [1]: https://cs.opensource.google/go/go/+/refs/tags/go1.25.4:src/net/http/transport.go;l=961-966
func TestHTTPProxy(t *testing.T) {
	platform, err := platforms.FromString(contrasttest.Flags.PlatformStr)
	require.NoError(t, err)

	runtimeHandler, err := manifest.RuntimeHandler(platform)
	require.NoError(t, err)

	ct := contrasttest.New(t)

	resources := kuberesource.CoordinatorBundle()
	resources = kuberesource.PatchRuntimeHandlers(resources, runtimeHandler)
	resources = kuberesource.AddPortForwarders(resources)

	ct.Init(t, resources)

	// Start a proxy server

	proxy := goproxy.NewProxyHttpServer()
	server := http.Server{Handler: proxy}
	errCh := make(chan error)

	// coordinatorConnectionProxied will be set to true if the proxy performs an HTTP CONNECT to the address of the Coordinator.
	var coordinatorConnectionProxied atomic.Bool
	// Similarly, this will be set to true if the proxy handles a connection for a container registry (e.g. ghcr.io).
	var registryConnectionProxied atomic.Bool
	proxy.ConnectDial = func(network string, addr string) (net.Conn, error) {
		// For future reference: addr is in host:port format, not a URI.
		t.Logf("Proxying connection: %q", addr)
		if strings.HasPrefix(addr, "0.0.0.0:") { // we use this address in ContrastTest.runAgainstCoordinator
			coordinatorConnectionProxied.Store(true)
		}
		// While we could parse the expected registries from the ImageReplacementsFile, we know
		// that the pause container image will come from MCR, so we use that as an indicator for
		// registry requests being proxied.
		if addr == "mcr.microsoft.com:443" {
			registryConnectionProxied.Store(true)
		}
		ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
		defer cancel()
		return (&net.Dialer{}).DialContext(ctx, network, addr)
	}

	proxyListener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:")
	require.NoError(t, err)
	proxyAddr := proxyListener.Addr().String()
	const invalidAddr = "127.0.0.1:0"

	t.Cleanup(func() {
		require.NoError(t, server.Close())
		err := <-errCh
		require.ErrorIs(t, err, http.ErrServerClosed)
	})

	go func() {
		errCh <- server.Serve(proxyListener)
	}()

	testCases := map[string]struct {
		env         map[string]string
		wantProxied bool
		wantErrMsg  string
	}{
		"proxy env not set": {
			wantProxied: false,
		},
		"https_proxy valid": {
			env:         map[string]string{"https_proxy": proxyAddr},
			wantProxied: true,
		},
		"https_proxy invalid": {
			env:        map[string]string{"https_proxy": invalidAddr},
			wantErrMsg: "transport: Error while dialing: dial tcp " + invalidAddr + ": connect: connection refused",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			for k, v := range tc.env {
				t.Setenv(k, v)
			}

			if tc.wantErrMsg != "" {
				// only try verify because set uses a retry loop
				assert.ErrorContains(runCommand(t.Context(), ct, "verify"), tc.wantErrMsg)
				return
			}

			require.NoError(runCommand(t.Context(), ct, "generate"))
			assert.False(coordinatorConnectionProxied.Swap(false))
			assert.Equal(tc.wantProxied, registryConnectionProxied.Swap(false))

			require.True(t.Run("apply", ct.Apply), "Kubernetes resources need to be applied for subsequent tests")

			require.NoError(runCommand(t.Context(), ct, "set"))
			assert.Equal(tc.wantProxied, coordinatorConnectionProxied.Swap(false))

			require.NoError(runCommand(t.Context(), ct, "verify"))
			assert.Equal(tc.wantProxied, coordinatorConnectionProxied.Swap(false))
		})
	}
}

func TestMain(m *testing.M) {
	contrasttest.RegisterFlags()
	runCommandStr := flag.String("run-command", "", "")
	runNamespace := flag.String("run-namespace", "", "")
	runWorkdir := flag.String("run-workdir", "", "")
	flag.Parse()

	ctx := context.Background()

	if *runCommandStr != "" {
		if err := runCommandImpl(ctx, *runCommandStr, *runNamespace, *runWorkdir, contrasttest.Flags.PlatformStr); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	os.Exit(m.Run())
}

// runCommand runs a CLI command in a new process so that the proxy env vars are re-read.
// Go caches the env vars, so we can't run the commands in the same process as usual.
func runCommand(ctx context.Context, ct *contrasttest.ContrastTest, cmd string) error {
	argv := append([]string{}, os.Args[1:]...)
	argv = append(argv, "-run-command="+cmd, "-run-namespace="+ct.Namespace, "-run-workdir="+ct.WorkDir)
	out, err := exec.CommandContext(ctx, os.Args[0], argv...).CombinedOutput()
	if err != nil {
		return errors.New(string(out))
	}
	return nil
}

func runCommandImpl(ctx context.Context, cmd, namespace, workDir, platformStr string) error {
	kclient, err := kubeclient.NewForTestWithoutT()
	if err != nil {
		return err
	}

	platform, err := platforms.FromString(platformStr)
	if err != nil {
		return err
	}
	ct := &contrasttest.ContrastTest{
		Namespace:             namespace,
		WorkDir:               workDir,
		Kubeclient:            kclient,
		Platform:              platform,
		ImageReplacementsFile: contrasttest.Flags.ImageReplacementsFile,
	}

	switch cmd {
	case "generate":
		return ct.RunGenerate(ctx)
	case "set":
		return ct.RunSet(ctx)
	case "verify":
		return ct.RunVerify(ctx)
	}

	return errors.New("unknown command: " + cmd)
}

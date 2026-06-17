// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build e2e

package ipsec

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/edgelesssys/contrast/e2e/internal/contrasttest"
	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/platforms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	initiatorVIP = "192.0.2.1"
	responderVIP = "192.0.2.2"
)

// TestIPSec ensures that Contrast guests can use kernel-managed IPSec with mesh certificates.
//
// The test spawns two mostly symmetric pods, both using strongSwan. Since the IP addresses are
// dynamic, the test first fetches these and then continues with the IPSec configuration. One pod,
// the initiator, initiates an IKEv2 handshake, while the responder just waits until the handshake
// arrives. Afterwards, both pods are connected via IPSec tunnel and can reach each other on their
// respective tunnel IPs (192.0.2.0/24, which would normally not be routed).
func TestIPSec(t *testing.T) {
	platform, err := platforms.FromString(contrasttest.Flags.PlatformStr)
	require.NoError(t, err)
	ct := contrasttest.New(t)

	runtimeHandler, err := manifest.RuntimeHandler(platform)
	require.NoError(t, err)

	resources := kuberesource.IPSec()
	coordinator := kuberesource.CoordinatorBundle()

	resources = append(resources, coordinator...)

	resources = kuberesource.PatchRuntimeHandlers(resources, runtimeHandler)

	resources = kuberesource.AddPortForwarders(resources)

	ct.Init(t, resources)
	require.True(t, t.Run("generate", ct.Generate), "contrast generate needs to succeed for subsequent tests")

	require.True(t, t.Run("apply", ct.Apply), "Kubernetes resources need to be applied for subsequent tests")

	require.True(t, t.Run("set", ct.Set), "contrast set needs to succeed for subsequent tests")

	require.True(t, t.Run("contrast verify", ct.Verify), "contrast verify needs to succeed for subsequent tests")

	t.Run("ipsec tunnel", func(t *testing.T) {
		require := require.New(t)

		ctx, cancel := context.WithTimeout(t.Context(), ct.FactorPlatformTimeout(5*time.Minute))
		defer cancel()

		require.NoError(ct.Kubeclient.WaitForDeployment(ctx, ct.Namespace, "ipsec"))

		pods, err := ct.Kubeclient.PodsFromDeployment(ctx, ct.Namespace, "ipsec")
		require.NoError(err)
		require.Len(pods, 2)

		initiator := pods[0]
		responder := pods[1]

		// Configure the responder first so that it's ready when the initiator initiates.
		configureCmd := configurationScript(responder.Status.PodIP, initiator.Status.PodIP, responderVIP, initiatorVIP, "none")
		stdout, stderr, err := ct.Kubeclient.Exec(ctx, ct.Namespace, pods[1].Name, []string{"/bin/sh", "-c", configureCmd})
		t.Logf("responder configure stdout:\n%s", stdout)
		require.NoError(err, "responder configure stderr:\n%s", stderr)

		configureCmd = configurationScript(initiator.Status.PodIP, responder.Status.PodIP, initiatorVIP, responderVIP, "start")
		stdout, stderr, err = ct.Kubeclient.Exec(ctx, ct.Namespace, pods[0].Name, []string{"/bin/sh", "-c", configureCmd})
		t.Logf("initiator configure stdout:\n%s", stdout)
		require.NoError(err, "initiator configure stderr:\n%s", stderr)

		// Wait until the tunnel is ready.
		require.EventuallyWithT(func(t *assert.CollectT) {
			// ping will return 0 if at least one reply was received. Let's be generous and try
			// three times, with 5s timeout.
			pingCmd := fmt.Sprintf("ping -c 3 -W 5 %s", responderVIP)
			stdout, stderr, err = ct.Kubeclient.Exec(ctx, ct.Namespace, initiator.Name, []string{"/bin/sh", "-c", pingCmd})
			assert.NoError(t, err, "ping failed:\nstdout:%s\nstderr:%s", stdout, stderr)
		}, time.Minute, 3*time.Second)
	})
}

const swanctlConfTemplate = `
connections {
    ipsec {
        version = 2

        local_addrs = %s
        remote_addrs = %s

        encap = yes

        local {
            auth = pubkey
        }

        remote {
            auth = pubkey
            id = %%any
        }

        children {
            ipsec {
                local_ts = %s/32
                remote_ts = %s/32
                start_action = %s
            }
        }
    }
}`

func configurationScript(localAddr, remoteAddr, localVIP, remoteVIP string, startAction string) string {
	return strings.Join([]string{
		fmt.Sprintf("ip addr add %s/32 dev lo", localVIP),
		"cat > /etc/swanctl/swanctl.conf << 'EOF'",
		fmt.Sprintf(swanctlConfTemplate, localAddr, remoteAddr, localVIP, remoteVIP, startAction),
		"EOF",
		"swanctl --load-all",
	}, "\n")
}

func TestMain(m *testing.M) {
	contrasttest.RegisterFlags()
	flag.Parse()

	os.Exit(m.Run())
}

// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

//go:build e2e

package trustedstore

import (
	"context"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/edgelesssys/contrast/e2e/internal/contrasttest"
	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/edgelesssys/contrast/internal/platforms"

	"github.com/stretchr/testify/require"
	applyappsv1 "k8s.io/client-go/applyconfigurations/apps/v1"
)

func TestSecureImageStorage(t *testing.T) {
	tests := map[string]struct {
		size       float64
		annotation string
	}{
		"enabled by default": {
			size: 10.0,
		},
		"block device size configurable": {
			size:       2.0,
			annotation: "2Gi",
		},
		"disabled through annotation": {
			size:       1.0,
			annotation: "0",
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)

			platform, err := platforms.FromString(contrasttest.Flags.PlatformStr)
			require.NoError(err)
			ct := contrasttest.New(t)

			runtimeHandler, err := manifest.RuntimeHandler(platform)
			require.NoError(err)

			resources := kuberesource.Vault(ct.Namespace)
			coordinator := kuberesource.CoordinatorBundle()

			for _, obj := range coordinator {
				switch v := obj.(type) {
				case *applyappsv1.StatefulSetApplyConfiguration:
					if v.Annotations == nil {
						v.Annotations = map[string]string{}
					}
					v.Annotations["contrast.edgeless.systems/secure-image-storage-size"] = tc.annotation
				}
			}

			resources = append(resources, coordinator...)
			resources = kuberesource.PatchRuntimeHandlers(resources, runtimeHandler)
			resources = kuberesource.AddPortForwarders(resources)

			ct.Init(t, resources)

			require.True(t.Run("generate", ct.Generate), "contrast generate needs to succeed for subsequent tests")
			require.True(t.Run("apply", ct.Apply), "Kubernetes resources need to be applied for subsequent tests")
			require.True(t.Run("set", ct.Set), "contrast set needs to succeed for subsequent tests")
			require.True(t.Run("contrast verify", ct.Verify), "contrast verify needs to succeed for subsequent tests")

			ctx, cancel := context.WithTimeout(t.Context(), ct.FactorPlatformTimeout(2*time.Minute))
			defer cancel()
			require.NoError(ct.Kubeclient.WaitForStatefulSet(ctx, ct.Namespace, "coordinator"))
			pods, err := ct.Kubeclient.PodsFromOwner(ctx, ct.Namespace, "StatefulSet", "coordinator")
			require.NoError(err)

			for _, c := range pods[0].Spec.Containers {

				stdOut, stdErr, err := ct.Kubeclient.ExecContainer(
					ctx,
					ct.Namespace,
					pods[0].Name,
					c.Name,
					[]string{
						"sh",
						"-c",
						"df -h /",
					},
				)

				require.NoError(err, "stdout: %s, stderr: %s", stdOut, stdErr)
				diskSize, err := extractDiskSize(stdOut)
				require.NoError(err, "failed to extract root disk size from container:\n%s", stdOut)

				require.NoError(checkDiskSize(tc.size, diskSize, c.Name))
			}
		})
	}
}

// extractDiskSize capture the "Size" field from df -h / output.
func extractDiskSize(logs string) (float64, error) {
	re := regexp.MustCompile(`(?m)^.*\s+([\d.]+)[MG]\s+([\d.]+)[KMGTP]?\s+([\d.]+)[KMGTP]?\s+([\d.]+)%\s+/$`)
	matches := re.FindStringSubmatch(logs)
	if len(matches) < 2 {
		return 0, fmt.Errorf("disk size not found")
	}
	return strconv.ParseFloat(matches[1], 64)
}

// checkDiskSize checks whether the captured size is roughly the same as what we have set.
// The output from df -h will never match exactly what is set in the deployment.
func checkDiskSize(expected, size float64, name string) error {
	if size < expected-1.0 || size > expected+1.0 {
		return fmt.Errorf("unexpected disk size in container %s: %.1fG (expected about %.1fGi)", name, size, expected)
	}
	return nil
}

func TestMain(m *testing.M) {
	contrasttest.RegisterFlags()
	flag.Parse()

	os.Exit(m.Run())
}

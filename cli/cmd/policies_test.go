// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/edgelesssys/contrast/internal/initdata"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	podYAML = `
apiVersion: v1
kind: Pod
metadata:
  name: my-pod
`
	invalidPolicyYAML = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test
spec:
  template:
    metadata:
      annotations:
        io.katacontainers.config.hypervisor.cc_init_data: 'invalid-base64'
`
)

var coordinatorDeploymentTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test
spec:
  template:
    metadata:
      annotations:
        io.katacontainers.config.hypervisor.cc_init_data: %q
        contrast.edgeless.systems/pod-role: coordinator
`

var podPolicyTemplate = `
apiVersion: v1
kind: Pod
metadata:
  name: another-pod
  annotations:
    io.katacontainers.config.hypervisor.cc_init_data: %q
`

func TestPoliciesFromKubeResources(t *testing.T) {
	i, err := initdata.New("sha256", nil)
	require.NoError(t, err)
	serialized, err := i.Encode()
	require.NoError(t, err)
	anno, err := serialized.EncodeKataAnnotation()
	require.NoError(t, err)
	testCases := []struct {
		name           string
		files          map[string]string
		expectedOutput []deployment
		expectedErr    string
	}{
		{
			name: "valid input",
			files: map[string]string{
				"deployment.yml": fmt.Sprintf(coordinatorDeploymentTemplate, anno),
			},
			expectedOutput: []deployment{
				{
					name:             "test",
					initdata:         serialized,
					role:             "coordinator",
					workloadSecretID: "apps/v1/Deployment/default/test",
				},
			},
		},
		{
			name: "missing annotation",
			files: map[string]string{
				"pod.yml": podYAML,
			},
		},
		{
			name: "invalid policy annotation",
			files: map[string]string{
				"deployment.yml": invalidPolicyYAML,
			},
			expectedErr: "illegal base64 data",
		},
		{
			name: "multiple files",
			files: map[string]string{
				"deployment.yml": fmt.Sprintf(coordinatorDeploymentTemplate, anno),
				"pod.yml":        fmt.Sprintf(podPolicyTemplate, anno),
			},
			expectedOutput: []deployment{
				{
					name:             "test",
					initdata:         serialized,
					role:             manifest.RoleCoordinator,
					workloadSecretID: "apps/v1/Deployment/default/test",
				},
				{
					name:             "another-pod",
					initdata:         serialized,
					role:             manifest.RoleNone,
					workloadSecretID: "core/v1/Pod/default/another-pod",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()

			var paths []string
			for filename, content := range tc.files {
				path := filepath.Join(tempDir, filename)
				err := os.WriteFile(path, []byte(content), 0o644)
				require.NoError(t, err)
				paths = append(paths, path)
			}

			deployments, err := policiesFromKubeResources(paths)
			sort.Slice(deployments, func(i, j int) bool {
				return deployments[i].name < deployments[j].name
			})
			sort.Slice(tc.expectedOutput, func(i, j int) bool {
				return tc.expectedOutput[i].name < tc.expectedOutput[j].name
			})
			if tc.expectedErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErr)
				assert.Nil(t, deployments, "deployments should be nil when an error is returned")
			} else {
				require.NoError(t, err)
				if tc.expectedOutput == nil {
					assert.Nil(t, deployments, "deployments should be nil")
				} else {
					if deployments == nil {
						t.Fatal("deployments should be non-nil")
					}
					assert.Equal(t, tc.expectedOutput, deployments)
				}
			}
		})
	}
}

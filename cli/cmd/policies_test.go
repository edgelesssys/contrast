// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package cmd

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"sort"
	"testing"

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
        io.katacontainers.config.agent.policy: 'invalid-base64'
`
)

var encodedValidPolicy = base64.StdEncoding.EncodeToString([]byte(`valid-agent-policy`))

var validDeploymentYAML = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test
spec:
  template:
    metadata:
      annotations:
        io.katacontainers.config.agent.policy: '` + encodedValidPolicy + `'
        contrast.edgeless.systems/pod-role: coordinator
`

var anotherValidPodYAML = `
apiVersion: v1
kind: Pod
metadata:
  name: another-pod
  annotations:
    io.katacontainers.config.agent.policy: '` + encodedValidPolicy + `'
`

func TestPoliciesFromKubeResources(t *testing.T) {
	testCases := []struct {
		name           string
		files          map[string]string
		expectedOutput []deployment
		expectedErr    string
	}{
		{
			name: "valid input",
			files: map[string]string{
				"deployment.yml": validDeploymentYAML,
			},
			expectedOutput: []deployment{
				{
					name:             "test",
					policy:           manifest.Policy([]byte(`valid-agent-policy`)),
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
			expectedErr: "failed to parse policy test",
		},
		{
			name: "multiple files",
			files: map[string]string{
				"deployment.yml": validDeploymentYAML,
				"pod.yml":        anotherValidPodYAML,
			},
			expectedOutput: []deployment{
				{
					name:             "test",
					policy:           manifest.Policy([]byte(`valid-agent-policy`)),
					role:             manifest.RoleCoordinator,
					workloadSecretID: "apps/v1/Deployment/default/test",
				},
				{
					name:             "another-pod",
					policy:           manifest.Policy([]byte(`valid-agent-policy`)),
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

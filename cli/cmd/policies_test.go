// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"encoding/base64"
	"sort"
	"testing"

	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var encodedValidPolicy = base64.StdEncoding.EncodeToString([]byte(`valid-agent-policy`))

func TestPoliciesFromKubeResources(t *testing.T) {
	testCases := []struct {
		name           string
		resources      []any
		expectedOutput []deployment
		expectedErr    string
	}{
		{
			name: "valid input",
			resources: []any{
				kuberesource.Deployment("test", "").
					WithSpec(kuberesource.DeploymentSpec().
						WithTemplate(kuberesource.PodTemplateSpec().
							WithAnnotations(map[string]string{
								kataPolicyAnnotationKey:   encodedValidPolicy,
								contrastRoleAnnotationKey: "coordinator",
							}))),
			},
			expectedOutput: []deployment{
				{
					name:             "test",
					policy:           manifest.Policy([]byte(`valid-agent-policy`)),
					role:             manifest.RoleCoordinator,
					workloadSecretID: "apps/v1/Deployment/default/test",
				},
			},
		},
		{
			name: "missing annotation",
			resources: []any{
				kuberesource.Deployment("test", "").DeploymentApplyConfiguration,
			},
		},
		{
			name: "invalid policy annotation",
			resources: []any{
				kuberesource.Pod("test", "").
					WithAnnotations(map[string]string{
						kataPolicyAnnotationKey: "invalid-base64",
					}),
			},
			expectedErr: "failed to parse policy test",
		},
		{
			name: "multiple files",
			resources: []any{
				kuberesource.Deployment("test", "").
					WithSpec(kuberesource.DeploymentSpec().
						WithTemplate(kuberesource.PodTemplateSpec().
							WithAnnotations(map[string]string{
								kataPolicyAnnotationKey:   encodedValidPolicy,
								contrastRoleAnnotationKey: "coordinator",
							}))),
				kuberesource.Pod("another-pod", "").
					WithAnnotations(map[string]string{
						kataPolicyAnnotationKey: encodedValidPolicy,
					}),
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
			deployments, err := policiesFromKubeResources(tc.resources)
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

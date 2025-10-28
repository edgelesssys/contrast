// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package cmd

import (
	"sort"
	"testing"

	"github.com/edgelesssys/contrast/internal/initdata"
	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/edgelesssys/contrast/internal/manifest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestPoliciesFromKubeResources(t *testing.T) {
	i, err := initdata.New("sha256", nil)
	require.NoError(t, err)
	serialized, err := i.Encode()
	require.NoError(t, err)
	anno, err := serialized.EncodeKataAnnotation()
	require.NoError(t, err)
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
								initdata.InitdataAnnotationKey: anno,
								contrastRoleAnnotationKey:      "coordinator",
							}).
							WithSpec(kuberesource.PodSpec().
								WithRuntimeClassName("contrast-cc"),
							))),
			},
			expectedOutput: []deployment{
				{
					name:             "test",
					initdata:         serialized,
					role:             manifest.RoleCoordinator,
					workloadSecretID: "apps/v1/Deployment/default/test",
				},
			},
		},
		{
			name: "missing annotation",
			resources: []any{
				kuberesource.Deployment("test", "").
					WithSpec(kuberesource.DeploymentSpec().
						WithTemplate(kuberesource.PodTemplateSpec().
							WithSpec(kuberesource.PodSpec().
								WithRuntimeClassName("contrast-cc"),
							))),
			},
			expectedErr: "missing initdata annotation",
		},
		{
			name: "invalid policy annotation",
			resources: []any{
				kuberesource.Pod("test", "").
					WithAnnotations(map[string]string{
						initdata.InitdataAnnotationKey: "invalid-base64",
					}).
					WithSpec(kuberesource.PodSpec().
						WithRuntimeClassName("contrast-cc"),
					),
			},
			expectedErr: "illegal base64 data",
		},
		{
			name: "multiple files",
			resources: []any{
				kuberesource.Deployment("test", "").
					WithSpec(kuberesource.DeploymentSpec().
						WithTemplate(kuberesource.PodTemplateSpec().
							WithAnnotations(map[string]string{
								initdata.InitdataAnnotationKey: anno,
								contrastRoleAnnotationKey:      "coordinator",
							}).
							WithSpec(kuberesource.PodSpec().
								WithRuntimeClassName("contrast-cc"),
							))),
				kuberesource.Pod("another-pod", "").
					WithAnnotations(map[string]string{
						initdata.InitdataAnnotationKey: anno,
					}).
					WithSpec(kuberesource.PodSpec().
						WithRuntimeClassName("contrast-cc"),
					),
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
			fileMap := make(map[string][]*unstructured.Unstructured)
			u, err := kuberesource.ResourcesToUnstructured(tc.resources)
			require.NoError(t, err)
			fileMap["testfile"] = append(fileMap["testfile"], u...)
			deployments, err := policiesFromKubeResources(fileMap)
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

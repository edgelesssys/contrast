// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package verifier_test

import (
	"testing"

	"github.com/edgelesssys/contrast/cli/verifier"
	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/stretchr/testify/require"
)

const deploymentWithoutGPU = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web
spec:
  replicas: 1
  template:
    spec:
      runtimeClassName: contrast-cc
      containers:
        - name: app
          resources:
            limits:
              cpu: "1"
`

const deploymentWithEnoughCPUsForGPUs = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web
spec:
  replicas: 1
  template:
    spec:
      runtimeClassName: contrast-cc
      containers:
        - name: app
          resources:
            limits:
              cpu: "2"
              nvidia.com/gpu: "2"
`

const deploymentWithTooFewCPUsForGPUs = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web
spec:
  replicas: 1
  template:
    metadata:
      name: web-pod
      namespace: test
    spec:
      runtimeClassName: contrast-cc
      containers:
        - name: app
          resources:
            limits:
              nvidia.com/gpu: "2" # only 1 vCPU (hypervisor overhead) for 2 GPUs
`

func TestGPUCountValid(t *testing.T) {
	testCases := map[string]struct {
		k8sYaml string
		wantErr bool
	}{
		"no gpu": {
			k8sYaml: deploymentWithoutGPU,
		},
		"enough cpus for gpus": {
			k8sYaml: deploymentWithEnoughCPUsForGPUs,
		},
		"too few cpus for gpus": {
			k8sYaml: deploymentWithTooFewCPUsForGPUs,
			wantErr: true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			toVerifySlice, err := kuberesource.UnmarshalApplyConfigurations([]byte(tc.k8sYaml))
			require.NoError(err)

			verifier := verifier.GPUCountValid{}

			for _, toVerify := range toVerifySlice {
				err := verifier.Verify(toVerify)
				if tc.wantErr {
					require.Error(err)
				} else {
					require.NoError(err)
				}
			}
		})
	}
}

// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package verifier_test

import (
	"testing"

	"github.com/edgelesssys/contrast/cli/verifier"
	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/stretchr/testify/require"
)

const deploymentWithoutExplicitCPUCount = `
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
              cpu: 2000m
`

const deploymentWithValidCPUCount = `
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
              cpu: 2000m
`

const deploymentWithTooManyCPUs = `
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
              cpu: "8" # invalid because of always-added CPU
`

func TestCPUCountValid(t *testing.T) {
	testCases := map[string]struct {
		k8sYaml string
		wantErr bool
	}{
		"no explicit cpu count": {
			k8sYaml: deploymentWithoutExplicitCPUCount,
		},
		"valid cpu count": {
			k8sYaml: deploymentWithValidCPUCount,
		},
		"too many cpus": {
			k8sYaml: deploymentWithTooManyCPUs,
			wantErr: true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			toVerifySlice, err := kuberesource.UnmarshalApplyConfigurations([]byte(tc.k8sYaml))
			require.NoError(err)

			verifier := verifier.CPUCountValid{}

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

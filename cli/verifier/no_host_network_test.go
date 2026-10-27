// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package verifier_test

import (
	"testing"

	"github.com/edgelesssys/contrast/cli/verifier"
	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/stretchr/testify/require"
)

const podWithHostNetwork = `
apiVersion: v1
kind: Pod
metadata:
  name: test
spec:
  runtimeClassName: contrast-cc
  hostNetwork: true
  containers:
    - name: test
      image: bash
`

const podWithHostNetworkFalse = `
apiVersion: v1
kind: Pod
metadata:
  name: test
spec:
  runtimeClassName: contrast-cc
  hostNetwork: false
  containers:
    - name: test
      image: bash
`

const podWithoutHostNetwork = `
apiVersion: v1
kind: Pod
metadata:
  name: test
spec:
  runtimeClassName: contrast-cc
  containers:
    - name: test
      image: bash
`

const deploymentWithHostNetwork = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test
spec:
  template:
    spec:
      runtimeClassName: contrast-cc
      hostNetwork: true
      containers:
        - name: test
          image: bash
`

const nonCCPodWithHostNetwork = `
apiVersion: v1
kind: Pod
metadata:
  name: test
spec:
  hostNetwork: true
  containers:
    - name: test
      image: bash
`

func TestVerifyNoHostNetwork(t *testing.T) {
	testCases := map[string]struct {
		k8sObjectYAML string
		wantErr       bool
	}{
		"no hostNetwork":              {k8sObjectYAML: podWithoutHostNetwork},
		"hostNetwork false":           {k8sObjectYAML: podWithHostNetworkFalse},
		"hostNetwork true":            {k8sObjectYAML: podWithHostNetwork, wantErr: true},
		"deployment hostNetwork":      {k8sObjectYAML: deploymentWithHostNetwork, wantErr: true},
		"non-cc pod with hostNetwork": {k8sObjectYAML: nonCCPodWithHostNetwork},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			toVerifySlice, err := kuberesource.UnmarshalApplyConfigurations([]byte(tc.k8sObjectYAML))
			require.NoError(err)

			verifier := verifier.ContrastPodSpecVerifier(verifier.NoHostNetwork)

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

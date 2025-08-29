// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package verifier_test

import (
	"testing"

	"github.com/edgelesssys/contrast/cli/verifier"
	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/stretchr/testify/require"
)

const deploymentNoAnnotations = `
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
        - name: currency-conversion
          image: ghcr.io/edgelesssys/conversion:v1.2.3@...
`

const deploymentWithEmptyEgressAnnotationInSpec = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web
spec:
  replicas: 1
  template:
    metadata:
      annotations:
        contrast.edgeless.systems/servicemesh-egress: ""
    spec:
      runtimeClassName: contrast-cc
      containers:
        - name: currency-conversion
          image: ghcr.io/edgelesssys/conversion:v1.2.3@...
`

const deploymentWithGoodEgressAnnotationInSpec = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web
spec:
  replicas: 1
  template:
    metadata:
      annotations:
        contrast.edgeless.systems/servicemesh-egress: "asdf"
    spec:
      runtimeClassName: contrast-cc
      containers:
        - name: currency-conversion
          image: ghcr.io/edgelesssys/conversion:v1.2.3@...
`

func TestServiceMeshEgress(t *testing.T) {
	testCases := map[string]struct {
		k8sYaml string
		wantErr bool
	}{
		"no annotations": {
			k8sYaml: deploymentNoAnnotations,
		},
		"bad spec": {
			k8sYaml: deploymentWithEmptyEgressAnnotationInSpec,
			wantErr: true,
		},
		"good spec": {
			k8sYaml: deploymentWithGoodEgressAnnotationInSpec,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			toVerifySlice, err := kuberesource.UnmarshalApplyConfigurations([]byte(tc.k8sYaml))
			require.NoError(err)

			verifier := verifier.ServiceMeshEgressNotEmpty{}

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

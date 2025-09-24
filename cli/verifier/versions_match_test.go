// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package verifier_test

import (
	"fmt"
	"testing"

	"github.com/edgelesssys/contrast/cli/verifier"
	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/stretchr/testify/require"
)

const (
	coordinatorWithVersionTemplate = `
apiVersion: v1
kind: Pod
metadata:
  name: test
  annotations:
    contrast.edgeless.systems/pod-role: %s
spec:
  runtimeClassName: contrast-cc
  containers:
    - name: coordinator
      image: ghcr.io/edgelesssys/contrast/coordinator:%s@sha256:65e7832acb46d952d5c96c824d6f370999b8f3d547b3c2c449e82d8dc4b4816c
`

	statefulSetWithVersionTemplate = `
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: test
spec:
  template:
    spec:
      runtimeClassName: contrast-cc
      containers:
        - name: contrast-initializer
          image: ghcr.io/edgelesssys/contrast/coordinator:%s@sha256:65e7832acb46d952d5c96c824d6f370999b8f3d547b3c2c449e82d8dc4b4816c
`

	podWithoutVersion = `
apiVersion: v1
kind: Pod
metadata:
  name: test
spec:
  runtimeClassName: contrast-cc
  containers:
    - name: contrast-initializer
      image: ghcr.io/edgelesssys/contrast/coordinator@sha256:65e7832acb46d952d5c96c824d6f370999b8f3d547b3c2c449e82d8dc4b4816c
`
)

func TestVerifyVersionsMatch(t *testing.T) {
	testCases := map[string]struct {
		k8sObjectYAML []byte
		version       string
		wantErr       bool
	}{
		"versions match": {
			k8sObjectYAML: fmt.Appendf(nil, statefulSetWithVersionTemplate, "v1.13.0"),
			version:       "v1.13.0",
		},
		"versions match with pod-role": {
			k8sObjectYAML: fmt.Appendf(nil, coordinatorWithVersionTemplate, "coordinator", "v1.13.0"),
			version:       "v1.13.0",
		},
		"cli version newer": {
			k8sObjectYAML: fmt.Appendf(nil, statefulSetWithVersionTemplate, "v1.12.0"),
			version:       "v1.13.0",
			wantErr:       true,
		},
		"cli version older": {
			k8sObjectYAML: fmt.Appendf(nil, statefulSetWithVersionTemplate, "v1.13.0"),
			version:       "v1.12.0",
			wantErr:       true,
		},
		"cli version mismatch with pod-role": {
			k8sObjectYAML: fmt.Appendf(nil, coordinatorWithVersionTemplate, "coordinator", "v1.13.0"),
			version:       "v1.12.0",
			wantErr:       true,
		},
		"cli version mismatch with differing pod-role unaffected": {
			k8sObjectYAML: fmt.Appendf(nil, coordinatorWithVersionTemplate, "not-coordinator", "v1.13.0"),
			version:       "v1.12.0",
		},
		"resource missing version skipped": {
			k8sObjectYAML: []byte(podWithoutVersion),
			version:       "v1.13.0",
		},
		"resource missing version skipped, dev build": {
			k8sObjectYAML: []byte(podWithoutVersion),
			version:       "v0.0.0-dev",
		},
		"resource with version, dev build": {
			k8sObjectYAML: fmt.Appendf(nil, statefulSetWithVersionTemplate, "v1.13.0"),
			version:       "v0.0.0-dev",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			toVerifySlice, err := kuberesource.UnmarshalApplyConfigurations(tc.k8sObjectYAML)
			require.NoError(err)

			verifier := verifier.VersionsMatch{Version: tc.version}

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

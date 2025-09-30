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
	podTemplate = `
apiVersion: v1
kind: Pod
metadata:
  name: test
spec:
  runtimeClassName: contrast-cc
  containers:
    - name: test
      image: "%s"
`
	statefulSetTemplate = `
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: test
spec:
  template:
    spec:
      runtimeClassName: contrast-cc
      containers:
      - name: test
        image: "%s"
`
)

func TestVerifyImageRef(t *testing.T) {
	testCases := map[string]struct {
		imageRef              string
		excludeContrastImages bool
		wantErr               bool
	}{
		"image ref empty": {
			imageRef: "bash",
			wantErr:  true,
		},
		"digest malformed, no tag": {
			imageRef: "bash@sha256:000",
			wantErr:  true,
		},
		"digest missing algorithm, no tag": {
			imageRef: "bash@0000000000000000000000000000000000000000000000000000000000000000",
			wantErr:  true,
		},
		"digest missing, with tag": {
			imageRef: "bash:0.0.1",
			wantErr:  true,
		},
		"digest malformed, with tag": {
			imageRef: "bash:0.0.1@sha256:000",
			wantErr:  true,
		},
		"digest missing algorithm, with tag": {
			imageRef: "bash:0.0.1@0000000000000000000000000000000000000000000000000000000000000000",
			wantErr:  true,
		},
		"image ref valid": {
			imageRef: "bash:0.0.1@sha256:0000000000000000000000000000000000000000000000000000000000000000",
		},
		"image ref valid, no tag": {
			imageRef: "bash@sha256:0000000000000000000000000000000000000000000000000000000000000000",
		},
		"contrast images excluded": {
			imageRef:              "ghcr.io/edgelesssys/contrast:latest",
			excludeContrastImages: true,
		},
		"contrast images not excluded": {
			imageRef: "ghcr.io/edgelesssys/contrast:latest",
			wantErr:  true,
		},
		"contrast image is pinned": {
			imageRef: "ghcr.io/edgelesssys/contrast:latest@sha256:0000000000000000000000000000000000000000000000000000000000000000",
		},
	}
	templates := map[string]string{
		"pod":         podTemplate,
		"statefulSet": statefulSetTemplate,
	}

	for tName, template := range templates {
		t.Run(tName, func(t *testing.T) {
			for name, tc := range testCases {
				t.Run(name, func(t *testing.T) {
					require := require.New(t)

					k8sObjectYAML := fmt.Appendf(nil, template, tc.imageRef)
					toVerifySlice, err := kuberesource.UnmarshalApplyConfigurations(k8sObjectYAML)
					require.NoError(err)

					verifier := verifier.ImageRefValid{
						ExcludeContrastImages: tc.excludeContrastImages,
					}

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
		})
	}
}

const (
	podNoImage = `
apiVersion: v1
kind: Pod
metadata:
  name: test
spec:
  runtimeClassName: contrast-cc
  containers:
    - name: test
`
	statefulSetNoImage = `
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: test
spec:
  template:
    spec:
      runtimeClassName: contrast-cc
      containers:
      - name: test
`
)

func TestVerifyImageRefMissing(t *testing.T) {
	testCases := map[string]string{
		"pod without image fails":          podNoImage,
		"stateful set without image fails": statefulSetNoImage,
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)

			toVerifySlice, err := kuberesource.UnmarshalApplyConfigurations([]byte(tc))
			require.NoError(err)

			verifier := verifier.ImageRefValid{}

			for _, toVerify := range toVerifySlice {
				require.Error(verifier.Verify(toVerify))
			}
		})
	}
}

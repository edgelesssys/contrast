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
      containers:
      - name: test
        image: "%s"
`
)

func TestVerifyImageRef(t *testing.T) {
	testCases := map[string]struct {
		imageRef string
		template string
		wantErr  bool
	}{
		"pod: image ref empty": {
			imageRef: "bash",
			template: podTemplate,
			wantErr:  true,
		},
		"pod: digest malformed, no tag": {
			imageRef: "bash@sha256:000",
			template: podTemplate,
			wantErr:  true,
		},
		"pod: digest missing algorithm, no tag": {
			imageRef: "bash@0000000000000000000000000000000000000000000000000000000000000000",
			template: podTemplate,
			wantErr:  true,
		},
		"pod: digest missing, with tag": {
			imageRef: "bash:0.0.1",
			template: podTemplate,
			wantErr:  true,
		},
		"pod: digest malformed, with tag": {
			imageRef: "bash:0.0.1@sha256:000",
			template: podTemplate,
			wantErr:  true,
		},
		"pod: digest missing algorithm, with tag": {
			imageRef: "bash:0.0.1@0000000000000000000000000000000000000000000000000000000000000000",
			template: podTemplate,
			wantErr:  true,
		},
		"pod: image ref valid": {
			imageRef: "bash:0.0.1@sha256:0000000000000000000000000000000000000000000000000000000000000000",
			template: podTemplate,
		},
		"pod: image ref valid, no tag": {
			imageRef: "bash@sha256:0000000000000000000000000000000000000000000000000000000000000000",
			template: podTemplate,
		},
		"statefulSet: image ref empty": {
			imageRef: "bash",
			template: statefulSetTemplate,
			wantErr:  true,
		},
		"statefulSet: digest malformed, no tag": {
			imageRef: "bash@sha256:000",
			template: statefulSetTemplate,
			wantErr:  true,
		},
		"statefulSet: digest missing algorithm, no tag": {
			imageRef: "bash@0000000000000000000000000000000000000000000000000000000000000000",
			template: statefulSetTemplate,
			wantErr:  true,
		},
		"statefulSet: digest missing, with tag": {
			imageRef: "bash:0.0.1",
			template: statefulSetTemplate,
			wantErr:  true,
		},
		"statefulSet: digest malformed, with tag": {
			imageRef: "bash:0.0.1@sha256:000",
			template: statefulSetTemplate,
			wantErr:  true,
		},
		"statefulSet: digest missing algorithm, with tag": {
			imageRef: "bash:0.0.1@0000000000000000000000000000000000000000000000000000000000000000",
			template: statefulSetTemplate,
			wantErr:  true,
		},
		"statefulSet: image ref valid": {
			imageRef: "bash:0.0.1@sha256:0000000000000000000000000000000000000000000000000000000000000000",
			template: statefulSetTemplate,
		},
		"statefulSet: image ref valid, no tag": {
			imageRef: "bash@sha256:0000000000000000000000000000000000000000000000000000000000000000",
			template: statefulSetTemplate,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)

			k8sObjectYAML := fmt.Appendf(nil, tc.template, tc.imageRef)
			toVerifySlice, err := kuberesource.UnmarshalApplyConfigurations(k8sObjectYAML)
			require.NoError(err)

			verifier := verifier.ImageRefValid{}

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

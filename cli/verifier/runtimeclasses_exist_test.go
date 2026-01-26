// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package verifier_test

import (
	"fmt"
	"testing"

	"github.com/edgelesssys/contrast/cli/verifier"
	"github.com/edgelesssys/contrast/internal/kuberesource"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

const (
	statefulSetWithoutSpec = `
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: test
spec:
  template:
`

	podWithoutSpec = `
apiVersion: v1
kind: Pod
metadata:
  name: test
`
	statefulSetWithoutRuntimeClassName = `
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: test
spec:
  template:
    spec:
      containers:
`

	podWithoutRuntimeClassName = `
apiVersion: v1
kind: Pod
metadata:
  name: test
spec:
  containers:
`

	statefulSetWithRuntimeClassName = `
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: test
spec:
  template:
    spec:
      runtimeClassName: %s
`

	podWithRuntimeClassName = `
apiVersion: v1
kind: Pod
metadata:
  name: test
spec:
  runtimeClassName: %s
`
)

func TestVerifyRuntimeClassesExist(t *testing.T) {
	testCases := map[string]struct {
		runtimeClassName string
		referenceValues  bool
		wantErr          bool
	}{
		"non-cc with reference values succeeds": {
			runtimeClassName: "non-cc",
			referenceValues:  true,
		},
		"non-cc without reference values succeeds": {
			runtimeClassName: "non-cc",
			referenceValues:  false,
		},

		"default case with reference values succeeds": {
			runtimeClassName: "contrast-cc",
			referenceValues:  true,
		},
		"default case without reference values fails": {
			runtimeClassName: "contrast-cc",
			referenceValues:  false,
			wantErr:          true,
		},

		"only prefix with reference values fails": {
			runtimeClassName: "contrast-cc-",
			referenceValues:  true,
			wantErr:          true,
		},
		"only prefix without reference values fails": {
			runtimeClassName: "contrast-cc-",
			referenceValues:  false,
			wantErr:          true,
		},

		"nonexistent with reference values fails": {
			runtimeClassName: "contrast-cc-nonexistent",
			referenceValues:  true,
			wantErr:          true,
		},
		"nonexistent without reference values fails": {
			runtimeClassName: "contrast-cc-nonexistent",
			referenceValues:  false,
			wantErr:          true,
		},

		"underspecified with reference values fails": {
			runtimeClassName: "contrast-cc-metal-qemu",
			referenceValues:  true,
			wantErr:          true,
		},
		"underspecified without reference values fails": {
			runtimeClassName: "contrast-cc-metal-qemu",
			referenceValues:  false,
			wantErr:          true,
		},

		"valid runtime class name with reference values succeeds": {
			runtimeClassName: "contrast-cc-metal-qemu-snp",
			referenceValues:  true,
		},
		"valid runtime class name without reference values succeeds": {
			runtimeClassName: "contrast-cc-metal-qemu-snp",
			referenceValues:  false,
		},

		"valid runtime class name with suffix, with reference values succeeds": {
			runtimeClassName: "contrast-cc-metal-qemu-snp-suffix",
			referenceValues:  true,
		},
		"valid runtime class name with suffix, without reference values succeeds": {
			runtimeClassName: "contrast-cc-metal-qemu-snp-suffix",
			referenceValues:  false,
		},
	}

	for name, tc := range testCases {
		for templateName, template := range map[string]string{"statefuleSet": statefulSetWithRuntimeClassName, "pod": podWithRuntimeClassName} {
			t.Run(fmt.Sprintf("%s %s", templateName, name), func(t *testing.T) {
				require := require.New(t)
				toVerifySlice, err := kuberesource.UnmarshalApplyConfigurations(fmt.Appendf(nil, template, tc.runtimeClassName))
				require.NoError(err)

				cmd := cobra.Command{}
				if tc.referenceValues {
					cmd.Flags().String("reference-values", "Metal-QEMU-SNP", "")
				} else {
					cmd.Flags().String("reference-values", "", "")
				}

				verifier := verifier.RuntimeClassesExist{Command: &cmd}

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

	trivialSuccessCases := map[string]string{
		"statefulSet without spec":             statefulSetWithoutSpec,
		"pod without spec":                     podWithoutSpec,
		"statefulSet without runtimeClassName": statefulSetWithoutRuntimeClassName,
		"pod without runtimeClassName":         podWithoutRuntimeClassName,
	}
	for name, template := range trivialSuccessCases {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			toVerifySlice, err := kuberesource.UnmarshalApplyConfigurations([]byte(template))
			require.NoError(err)

			cmd := cobra.Command{}
			cmd.Flags().String("reference-values", "", "")
			verifier := verifier.RuntimeClassesExist{Command: &cmd}

			for _, toVerify := range toVerifySlice {
				require.NoError(verifier.Verify(toVerify))
			}
		})
	}
}

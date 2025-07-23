// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package verifier_test

import (
	"errors"
	"testing"

	"github.com/edgelesssys/contrast/cli/verifier"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestVerificationPipeline(t *testing.T) {
	verificationSuccess := func(_ *unstructured.Unstructured) error { return nil }
	verificationFailure := func(_ *unstructured.Unstructured) error { return errors.New("some error") }
	testCases := map[string]struct {
		pipeline []verifier.Verifier
		wantErr  bool
	}{
		"good pipeline single function": {
			pipeline: []verifier.Verifier{verificationSuccess},
		},
		"bad pipeline single function": {
			pipeline: []verifier.Verifier{verificationFailure},
			wantErr:  true,
		},
		"good pipeline multiple functions": {
			pipeline: []verifier.Verifier{verificationSuccess, verificationSuccess, verificationSuccess},
		},
		"bad pipeline multiple functions": {
			pipeline: []verifier.Verifier{verificationSuccess, verificationFailure, verificationSuccess},
			wantErr:  true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			pipelineToTest := verifier.VerificationPipeline{
				Pipeline: tc.pipeline,
			}
			// using nil since individual pipelines are tested on their own
			err := pipelineToTest.Verify(nil)
			if tc.wantErr {
				require.Error(err)
			} else {
				require.NoError(err)
			}
		})
	}
}

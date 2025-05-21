// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build contrast_unstable_api

package sdk

import (
	"testing"
)

func TestVerify(t *testing.T) {
	tests := map[string]struct {
		expectedManifest []byte
		manifestHistory  [][]byte
		errMsg           string
	}{
		"Empty manifest history": {
			expectedManifest: []byte("expected"),
			manifestHistory:  [][]byte{},
			errMsg:           "manifest history is empty",
		},
		"Matching manifest": {
			expectedManifest: []byte("expected"),
			manifestHistory:  [][]byte{[]byte("old"), []byte("expected")},
		},
		"Non-matching manifest": {
			expectedManifest: []byte("expected"),
			manifestHistory:  [][]byte{[]byte("old"), []byte("current")},
			errMsg:           "active manifest does not match expected manifest",
		},
		"Matching manifest is not latest": {
			expectedManifest: []byte("expected"),
			manifestHistory:  [][]byte{[]byte("expected"), []byte("current")},
			errMsg:           "active manifest does not match expected manifest",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			client := Client{}
			err := client.Verify(tt.expectedManifest, tt.manifestHistory)

			if err != nil && err.Error() != tt.errMsg {
				t.Errorf("actual error: '%v', expected error: '%v'", err, tt.errMsg)
			}
		})
	}
}

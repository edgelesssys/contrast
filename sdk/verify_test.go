// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

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

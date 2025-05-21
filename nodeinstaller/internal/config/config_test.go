// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func TestValidate(t *testing.T) {
	testCases := []struct {
		name   string
		config Config
		valid  bool
	}{
		{
			name: "valid http File",
			config: Config{
				Files: []File{{
					URL:       "https://example.com/file1",
					Path:      "/path/to/file1",
					Integrity: "sha256-abcdef123456",
				}},
			},
			valid: true,
		},
		{
			name: "valid file File",
			config: Config{
				Files: []File{{
					URL:       "file:////example.com/file1",
					Path:      "/path/to/file1",
					Integrity: "sha256-abcdef123456",
				}},
			},
			valid: true,
		},
		{
			name: "missing URL",
			config: Config{
				Files: []File{{
					Path:      "/path/to/file1",
					Integrity: "sha256-abcdef123456",
				}},
			},
		},
		{
			name: "missing Path",
			config: Config{
				Files: []File{{
					URL:       "https://example.com/file1",
					Integrity: "sha256-abcdef123456",
				}},
			},
		},
		{
			name: "missing relative path",
			config: Config{
				Files: []File{{
					URL:       "https://example.com/file1",
					Path:      "path/to/file1",
					Integrity: "sha256-abcdef123456",
				}},
			},
		},
		{
			name: "missing Integrity",
			config: Config{
				Files: []File{{
					URL:  "https://example.com/file1",
					Path: "/path/to/file1",
				}},
			},
		},
		{
			name: "invalid URL",
			config: Config{
				Files: []File{{
					URL:       "invalid\x00url",
					Path:      "/path/to/file1",
					Integrity: "sha256-abcdef123456",
				}},
			},
		},
		{
			name: "invalid scheme",
			config: Config{
				Files: []File{{
					URL:       "ftp://example.com/file1",
					Path:      "/path/to/file1",
					Integrity: "sha256-abcdef123456",
				}},
			},
		},
		{
			name: "invalid Integrity algorithm",
			config: Config{
				Files: []File{{
					URL:       "https://example.com/file1",
					Path:      "/path/to/file1",
					Integrity: "md5-abcdef123456",
				}},
			},
		},
		{
			name: "invalid Integrity value",
			config: Config{
				Files: []File{{
					URL:       "https://example.com/file1",
					Path:      "/path/to/file1",
					Integrity: "sha256-xyz",
				}},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.config.Validate()
			if tc.valid {
				assert.NoError(t, err, "Expected no error, but got one")
			} else {
				assert.Error(t, err, "Expected error, but got none")
			}
		})
	}
}

// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

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
				RuntimeHandlerName: "contrast-cc",
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
				RuntimeHandlerName: "contrast-cc",
				Files: []File{{
					URL:       "file:////example.com/file1",
					Path:      "/path/to/file1",
					Integrity: "sha256-abcdef123456",
				}},
			},
			valid: true,
		},
		{
			name: "missing RuntimeHandlerName",
			config: Config{
				Files: []File{{
					URL:       "https://example.com/file1",
					Path:      "/path/to/file1",
					Integrity: "sha256-abcdef123456",
				}},
			},
		},
		{
			name: "RuntimeHandlerName too long",
			config: Config{
				RuntimeHandlerName: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				Files: []File{{
					URL:       "https://example.com/file1",
					Path:      "/path/to/file1",
					Integrity: "sha256-abcdef123456",
				}},
			},
		},
		{
			name: "RuntimeHandlerName has invalid characters",
			config: Config{
				RuntimeHandlerName: "invalid name=",
				Files: []File{{
					URL:       "https://example.com/file1",
					Path:      "/path/to/file1",
					Integrity: "sha256-abcdef123456",
				}},
			},
		},
		{
			name: "missing URL",
			config: Config{
				RuntimeHandlerName: "contrast-cc",
				Files: []File{{
					Path:      "/path/to/file1",
					Integrity: "sha256-abcdef123456",
				}},
			},
		},
		{
			name: "missing Path",
			config: Config{
				RuntimeHandlerName: "contrast-cc",
				Files: []File{{
					URL:       "https://example.com/file1",
					Integrity: "sha256-abcdef123456",
				}},
			},
		},
		{
			name: "missing relative path",
			config: Config{
				RuntimeHandlerName: "contrast-cc",
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
				RuntimeHandlerName: "contrast-cc",
				Files: []File{{
					URL:  "https://example.com/file1",
					Path: "/path/to/file1",
				}},
			},
		},
		{
			name: "invalid URL",
			config: Config{
				RuntimeHandlerName: "contrast-cc",
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
				RuntimeHandlerName: "contrast-cc",
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
				RuntimeHandlerName: "contrast-cc",
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
				RuntimeHandlerName: "contrast-cc",
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

// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package fileop

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func TestOp(t *testing.T) {
	testCases := map[string]struct {
		srcContent   []byte
		dstContent   []byte
		wantModified bool
		wantErr      bool
	}{
		"identical": {
			srcContent: []byte("foo"),
			dstContent: []byte("foo"),
		},
		"different": {
			srcContent:   []byte("foo"),
			dstContent:   []byte("bar"),
			wantModified: true,
		},
		"src not found": {
			dstContent: []byte("foo"),
			wantErr:    true,
		},
		"dst not found": {
			srcContent:   []byte("foo"),
			wantModified: true,
		},
	}

	t.Run("CopyOnDiff", func(t *testing.T) {
		for name, tc := range testCases {
			t.Run(name, func(t *testing.T) {
				assert := assert.New(t)
				require := require.New(t)

				emptyDir := t.TempDir()
				defer os.RemoveAll(emptyDir)
				src := filepath.Join(emptyDir, "src")
				dst := filepath.Join(emptyDir, "dst")
				if tc.srcContent != nil {
					require.NoError(os.WriteFile(src, tc.srcContent, 0o644))
					defer os.Remove(src)
				}
				if tc.dstContent != nil {
					require.NoError(os.WriteFile(dst, tc.dstContent, 0o644))
					defer os.Remove(dst)
				}
				osOP := NewDefault()
				modified, err := osOP.CopyOnDiff(src, dst)
				if tc.wantErr {
					require.Error(err)
					return
				}
				require.NoError(err)
				assert.Equal(tc.wantModified, modified)
				got, err := os.ReadFile(dst)
				require.NoError(err)
				assert.Equal(tc.srcContent, got)
			})
		}
	})

	t.Run("Copy", func(t *testing.T) {
		for name, tc := range testCases {
			t.Run(name, func(t *testing.T) {
				assert := assert.New(t)
				require := require.New(t)

				emptyDir := t.TempDir()
				defer os.RemoveAll(emptyDir)
				src := filepath.Join(emptyDir, "src")
				dst := filepath.Join(emptyDir, "dst")
				if tc.srcContent != nil {
					require.NoError(os.WriteFile(src, tc.srcContent, 0o644))
					defer os.Remove(src)
				}
				if tc.dstContent != nil {
					require.NoError(os.WriteFile(dst, tc.dstContent, 0o644))
					defer os.Remove(dst)
				}
				osOP := NewDefault()
				err := osOP.Copy(src, dst)
				if tc.wantErr {
					require.Error(err)
					return
				}
				require.NoError(err)
				got, err := os.ReadFile(dst)
				require.NoError(err)
				assert.Equal(tc.srcContent, got)
			})
		}
	})

	t.Run("Move", func(t *testing.T) {
		for name, tc := range testCases {
			t.Run(name, func(t *testing.T) {
				assert := assert.New(t)
				require := require.New(t)

				emptyDir := t.TempDir()
				defer os.RemoveAll(emptyDir)
				src := filepath.Join(emptyDir, "src")
				dst := filepath.Join(emptyDir, "dst")
				if tc.srcContent != nil {
					require.NoError(os.WriteFile(src, tc.srcContent, 0o644))
					defer os.Remove(src)
				}
				if tc.dstContent != nil {
					require.NoError(os.WriteFile(dst, tc.dstContent, 0o644))
					defer os.Remove(dst)
				}
				osOP := NewDefault()
				err := osOP.Move(src, dst)
				if tc.wantErr {
					require.Error(err)
					return
				}
				require.NoError(err)
				got, err := os.ReadFile(dst)
				require.NoError(err)
				assert.Equal(tc.srcContent, got)
				assert.NoFileExists(src)
			})
		}
	})
}

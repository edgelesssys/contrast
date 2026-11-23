// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckNoDeviceDeps(t *testing.T) {
	testCases := map[string]struct {
		files   map[string]string
		wantErr bool
	}{
		"no device deps":  {files: map[string]string{"multi-user.target.wants/foo.service": "", "foo.service": ""}},
		"wants":           {files: map[string]string{"dev-vda.device.wants/foo.service": ""}, wantErr: true},
		"requires":        {files: map[string]string{"dev-vda.device.requires/foo.service": ""}, wantErr: true},
		"upholds":         {files: map[string]string{"dev-vda.device.upholds/foo.service": ""}, wantErr: true},
		"unit file":       {files: map[string]string{"dev-vdb.device": "[Unit]\nWants=extra.service\n"}, wantErr: true},
		"drop-in":         {files: map[string]string{"dev-vda.device.d/override.conf": "[Unit]\nWants=foo.service\n"}, wantErr: true},
		"type drop-in":    {files: map[string]string{"device.d/extra.conf": "[Unit]\nWants=extra.service\n"}, wantErr: true},
		"empty directory": {files: map[string]string{}},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			err := checkNoDeviceDeps(writeFiles(t, tc.files))
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func writeFiles(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		path := filepath.Join(dir, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	}
	return dir
}

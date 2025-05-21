// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package embedbin

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var calledFromInstallLocation = flag.Bool("called-from-install-location", false, "set to true when running from the install location")

func TestMain(m *testing.M) {
	flag.Parse()
	if *calledFromInstallLocation {
		fmt.Println("called from install location")
		os.Exit(123)
	}
	os.Exit(m.Run())
}

func TestFallback(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	contents := []byte("test")
	fs := afero.NewMemMapFs()
	installer := RegularInstaller{fs: fs}
	installed, err := installer.Install("", contents)
	assert.True(installed.IsRegular())
	require.NoError(err)

	assert.NotEmpty(installed.Path())
	got, err := afero.ReadFile(fs, installed.Path())
	require.NoError(err)

	assert.Equal(contents, got)
	assert.NoError(installed.Uninstall())
}

func TestInstallAndRun(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	// installs itself
	selfExePath, err := os.Executable()
	require.NoError(err)
	testbin, err := os.ReadFile(selfExePath)
	require.NoError(err)
	installer := New()
	installed, err := installer.Install("", testbin)
	require.NoError(err)

	// ensure the installation worked
	assert.NotEmpty(installed.Path())
	got, err := os.ReadFile(installed.Path())
	require.NoError(err)
	assert.Equal(testbin, got)

	// run the installed binary with a flag to indicate that it was called from the install location
	out, err := exec.Command(installed.Path(), "-called-from-install-location").Output()
	assert.Error(err)
	assert.Contains(string(out), "called from install location")
	var exitErr *exec.ExitError
	require.ErrorAs(err, &exitErr)
	assert.Equal(123, exitErr.ExitCode())

	// uninstall should exit cleanly
	assert.NoError(installed.Uninstall())
}

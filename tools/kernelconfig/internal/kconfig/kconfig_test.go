// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package kconfig

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	input := `CONFIG_A=y
# CONFIG_B is not set
CONFIG_C="value"
# Comment
`
	c, err := Parse(bytes.NewBufferString(input))
	require.NoError(t, err)
	assert.Len(t, c.lines, 4)
	assert.Equal(t, 0, c.index["CONFIG_A"])
	assert.Equal(t, 1, c.index["CONFIG_B"])
	assert.Equal(t, 2, c.index["CONFIG_C"])
}

func TestSet(t *testing.T) {
	input := `CONFIG_A=y
# CONFIG_B is not set
`
	c, err := Parse(bytes.NewBufferString(input))
	require.NoError(t, err)

	c.Set("CONFIG_A", "n")
	assert.Equal(t, "CONFIG_A=n", c.lines[0])

	c.Set("CONFIG_B", "y")
	assert.Equal(t, "CONFIG_B=y", c.lines[1])

	c.Set("CONFIG_NEW", "somevalue")
	assert.Equal(t, "CONFIG_NEW=somevalue", c.lines[2])
}

func TestUnset(t *testing.T) {
	input := `CONFIG_A=y
# CONFIG_B is not set
`
	c, err := Parse(bytes.NewBufferString(input))
	require.NoError(t, err)

	c.Unset("CONFIG_A")
	assert.Equal(t, "# CONFIG_A is not set", c.lines[0])

	c.Unset("CONFIG_B")
	assert.Equal(t, "# CONFIG_B is not set", c.lines[1])

	c.Unset("CONFIG_NEW")
	assert.Equal(t, "# CONFIG_NEW is not set", c.lines[2])
}

func TestMarshal(t *testing.T) {
	input := `CONFIG_A=y
# CONFIG_B is not set
`
	c, err := Parse(bytes.NewBufferString(input))
	require.NoError(t, err)

	c.Set("CONFIG_A", "n")
	c.Unset("CONFIG_NEW")

	expected := `CONFIG_A=n
# CONFIG_B is not set
# CONFIG_NEW is not set
`
	assert.Equal(t, expected, string(c.Marshal()))
}

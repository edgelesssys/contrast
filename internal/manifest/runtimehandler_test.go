// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package manifest

import (
	"testing"

	"github.com/edgelesssys/contrast/internal/platforms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRuntimeHandler(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	for _, platform := range platforms.All() {
		runtimeHandler, err := RuntimeHandler(platform)
		require.NoError(err)
		assert.NotEmpty(runtimeHandler)
		assert.Less(len(runtimeHandler), 64, "runtime handler name can be 63 characters at most")
		assert.Regexp(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`, runtimeHandler, "runtimeHandlerName must be a lowercase RFC 1123 subdomain")
	}
}

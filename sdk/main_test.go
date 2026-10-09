// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build contrast_unstable_api

package sdk

import (
	"os"
	"testing"

	"github.com/edgelesssys/contrast/sdk/apiv1"
)

// TestMain makes the SDK speak API version v1 in the tests of this package.
//
// Outside of tests, no API version is supported yet, see [SupportedAPIVersions].
func TestMain(m *testing.M) {
	SupportedAPIVersions = []string{apiv1.Version}
	os.Exit(m.Run())
}

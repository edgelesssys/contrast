// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build e2e

package badamlvuln

import (
	"flag"
	"os"
	"testing"

	"github.com/edgelesssys/contrast/e2e/internal/contrasttest"
)

func TestBadAMLVulnerability(t *testing.T) {
	BadAMLTest(t, true)
}

func TestMain(m *testing.M) {
	contrasttest.RegisterFlags()
	flag.Parse()
	os.Exit(m.Run())
}

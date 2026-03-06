// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build e2e

package badamlsandbox

import (
	"flag"
	"os"
	"testing"

	badamlvuln "github.com/edgelesssys/contrast/e2e/badaml-vuln"
	"github.com/edgelesssys/contrast/e2e/internal/contrasttest"
)

func TestBadAMLVulnerability(t *testing.T) {
	badamlvuln.BadAMLTest(t, false)
}

func TestMain(m *testing.M) {
	contrasttest.RegisterFlags()
	flag.Parse()
	os.Exit(m.Run())
}

// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

// test-if: path:overlays/sets/badaml.nix
// test-if: path:overlays/sets/no-aml-sandbox.nix
// test-if: path:packages/by-name/kata/kernel-uvm
// test-if: path:packages/by-name/qemu-wrapped
// test-if: path:packages/by-name/badaml-payload
// test-if: path:e2e/badaml-vuln

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

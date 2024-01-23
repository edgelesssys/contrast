package main

import (
	_ "embed"
)

const (
	coordRootPEMFilename   = "coordinator-root.pem"
	coordIntermPEMFilename = "mesh-root.pem"
	manifestFilename       = "manifest.json"
	settingsFilename       = "settings.json"
	rulesFilename          = "rules.rego"
	policyDir              = "."
	verifyDir              = "./verify"
)

//go:generate bash -c "nix build .#genpolicy.settings-dev && install -D ./result/genpolicy-settings.json assets/genpolicy-settings.json && rm -rf result"
//go:generate bash -c "nix build .#genpolicy.rules && install -D ./result/genpolicy-rules.rego assets/genpolicy-rules.rego && rm -rf result"

var (
	//go:embed assets/genpolicy-settings.json
	defaultGenpolicySettings []byte
	//go:embed assets/genpolicy-rules.rego
	defaultRules []byte
)

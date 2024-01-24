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

var (
	//go:embed assets/genpolicy-settings.json
	defaultGenpolicySettings []byte
	//go:embed assets/genpolicy-rules.rego
	defaultRules []byte
)

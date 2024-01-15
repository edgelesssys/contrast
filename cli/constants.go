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
	//go:embed genpolicy-msft.json
	defaultGenpolicySettings []byte
	//go:embed rules.rego
	defaultRules []byte
)

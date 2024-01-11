package main

import (
	_ "embed"
)

const (
	coordRootPEMFilename   = "coordinator-root.pem"
	coordIntermPEMFilename = "mesh-root.pem"
	manifestFilename       = "manifest.json"
	rulesFilename          = "rules.rego"
	verifyDir              = "./verify"
)

var (
	//go:embed genpolicy-msft.json
	defaultGenpolicySettings []byte
	//go:embed rules.rego
	defaultRules []byte
)

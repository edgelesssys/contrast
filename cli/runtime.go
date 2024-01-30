package main

var (
	genpolicyPath = "genpolicy"

	// DefaultCoordinatorPolicyHash is derived from the coordinator release candidate and injected at release build time.
	//
	// It is intentionally left empty for dev builds.
	DefaultCoordinatorPolicyHash = "" // TODO(burgerdev): actually inject something at build time.
)

// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package allowlist

// Default lists the hosts that kds-proxy will proxy to. Any other host is rejected.
var Default = map[string]struct{}{
	"kdsintf.amd.com":               {},
	"api.trustedservices.intel.com": {},
	"rim.attestation.nvidia.com":    {},
}

// Allows reports whether host is in the allowlist.
func Allows(host string) bool {
	_, ok := Default[host]
	return ok
}

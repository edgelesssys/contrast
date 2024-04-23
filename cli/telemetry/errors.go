// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package telemetry

// classifyCmdErr maps errors to fixed strings to avoid leaking sensitive data inside error messages.
func classifyCmdErr(e error) string {
	if e == nil {
		return ""
	}
	switch {
	default:
		return "unknown error"
	}
}

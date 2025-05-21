// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

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

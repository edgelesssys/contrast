// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package main

import "context"

type TestCase struct {
	Kind    string `json:"kind"`
	Allowed bool   `json:"allowed"`
	Request any    `json:"request"`
}

type Policy interface {
	// AllowRequest evaluates the given test case against the policy.
	AllowRequest(ctx context.Context, tc TestCase) (allowed bool, prints string, err error)
}

// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInsecureRuntimesAllowed(t *testing.T) {
	testCases := map[string]struct {
		value string
		want  bool
	}{
		"empty":   {value: "", want: false},
		"true":    {value: "true", want: true},
		"one":     {value: "1", want: true},
		"false":   {value: "false", want: false},
		"garbage": {value: "yes please", want: false},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Setenv(allowInsecureEnvVar, tc.value)
			assert.Equal(t, tc.want, insecureRuntimesAllowed())
		})
	}
}

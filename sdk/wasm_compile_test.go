// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

//go:build js && wasm && contrast_unstable_api

package sdk

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompilation(t *testing.T) {
	// This doesn't actually test any behavior. The goal of this test
	// is solely to verify that the SDK builds with GOOS=js and GOARCH=wasm.

	_, err := New().ValidateAttestation(context.Background(), "", []byte{}, []byte{})
	require.Error(t, err)
}

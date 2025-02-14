// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

// Package retry provides functions to check if a gRPC error is retryable.
package retry

import (
	"errors"
	"strings"
	"syscall"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Retriable checks whether it makes sense to retry this gRPC call to the Coordinator.
func Retriable(err error) bool {
	if errors.Is(err, syscall.ECONNREFUSED) {
		return true
	}
	statusErr, isStatusErr := status.FromError(err)
	if !isStatusErr {
		return false
	}
	if statusErr.Code() == codes.Internal {
		return true
	}
	if statusErr.Code() != codes.Unavailable {
		return false
	}
	// TLS handshake errors cause this status code. We don't want to retry attestation failures,
	// so we check the wrapped message.
	msg := err.Error()
	if !strings.Contains(msg, "authentication handshake failed") {
		return true
	}

	// The only handshake failure worth retrying is an unexpectedly closed connection - EOF.
	return strings.Contains(msg, "EOF")
}

// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

// Package retry provides functions to check if a gRPC error is retryable.
package retry

import (
	"regexp"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	authenticationHandshakeRE = regexp.MustCompile(`transport: authentication handshake failed: ((?:\s|\w)+)`)
	retriableCauses           = map[string]struct{}{"EOF": {}, "context deadline exceeded": {}}
)

// ServiceIsUnavailable checks if the error is a grpc status with code Unavailable.
// In the special case of an authentication handshake failure, false is returned to prevent further retries,
// unless the cause of the handshake failure points to a transient condition.
func ServiceIsUnavailable(err error) (ret bool) {
	statusErr, ok := status.FromError(err)
	if !ok {
		return false
	}

	if statusErr.Code() != codes.Unavailable {
		return false
	}

	matches := authenticationHandshakeRE.FindStringSubmatch(err.Error())
	if len(matches) < 2 {
		return true
	}

	_, isRetriable := retriableCauses[matches[1]]
	return isRetriable
}

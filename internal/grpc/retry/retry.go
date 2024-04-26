// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

// Package retry provides functions to check if a gRPC error is retryable.
package retry

import (
	"errors"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	authHandshakeErr                 = `connection error: desc = "transport: authentication handshake failed`
	authHandshakeDeadlineExceededErr = `connection error: desc = "transport: authentication handshake failed: context deadline exceeded`
)

// grpcErr is the error type that is returned by the grpc client.
// taken from google.golang.org/grpc/status.FromError.
type grpcErr interface {
	GRPCStatus() *status.Status
	Error() string
}

// ServiceIsUnavailable checks if the error is a grpc status with code Unavailable.
// In the special case of an authentication handshake failure, false is returned to prevent further retries.
func ServiceIsUnavailable(err error) (ret bool) {
	var targetErr grpcErr
	if !errors.As(err, &targetErr) {
		return false
	}

	statusErr, ok := status.FromError(targetErr)
	if !ok {
		return false
	}

	if statusErr.Code() != codes.Unavailable {
		return false
	}

	// retry if the handshake deadline was exceeded
	if strings.HasPrefix(statusErr.Message(), authHandshakeDeadlineExceededErr) {
		return true
	}

	return !strings.HasPrefix(statusErr.Message(), authHandshakeErr)
}

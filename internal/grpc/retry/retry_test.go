// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package retry

import (
	"fmt"
	"net"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestRetriable(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "no error",
			err:  nil,
			want: false,
		},
		{
			name: "connection refused",
			err:  fmt.Errorf("dial tcp [::1]:2: connect: %w", syscall.ECONNREFUSED),
			want: true, // Maybe the service is just not yet up?
		},
		{
			name: "no route to host",
			err:  fmt.Errorf("dial tcp 192.0.2.1:80: connect: %w", syscall.EHOSTUNREACH),
			want: false, // This is an issue on the client side.
		},
		{
			name: "no such host",
			err:  fmt.Errorf("dial tcp: %w", &net.DNSError{Err: "no such host", Name: "domain.invalid", IsNotFound: true}),
			want: false, // DNS is unlikely to be the source of a transient issue.
		},
		{
			name: "grpc handshake failure bad certificate",
			err:  status.Error(codes.Unavailable, `connection error: desc = "transport: authentication handshake failed: bad certificate"`),
			want: false, // This is an error on the application level (bad atls), retrying is unlikely to help.
		},
		{
			name: "grpc handshake failure eof",
			err:  status.Error(codes.Unavailable, `connection error: desc = "transport: authentication handshake failed: EOF"`),
			want: true, // Most likely a connection reset, try to connect again.
		},
		{
			name: "grpc handshake failure context deadline exceeded",
			err:  status.Error(codes.Unavailable, `connection error: desc = "transport: authentication handshake failed: context deadline exceeded"`),
			want: false, // If the context expired it does not make sense to retry.
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, Retriable(tc.err))
		})
	}

	grpcCases := []struct {
		code codes.Code
		want bool
	}{
		{
			code: codes.PermissionDenied,
			want: false,
		},
		{
			code: codes.Unauthenticated,
			want: false,
		},
		{
			code: codes.InvalidArgument,
			want: false,
		},
		{
			code: codes.FailedPrecondition,
			want: false,
		},
		{
			code: codes.Aborted,
			want: false,
		},
		{
			code: codes.Unavailable,
			want: true, // Unavailable usually indicates a temprorary error.
		},
		{
			code: codes.Aborted,
			want: false,
		},
		{
			code: codes.Internal,
			want: true, // Internal may point to a transient problem in the Coordinator.
		},
		{
			code: codes.DeadlineExceeded,
			want: false, // This is triggered by client-side context expiration.
		},
		{
			code: codes.Canceled,
			want: false, // This is triggered by client-side cancellation.
		},
	}

	for _, tc := range grpcCases {
		t.Run("grpc "+tc.code.String(), func(t *testing.T) {
			assert.Equal(t, tc.want, Retriable(status.Error(tc.code, "generic message")))
		})
	}
}

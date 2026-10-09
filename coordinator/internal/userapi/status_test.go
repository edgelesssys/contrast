// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package userapi

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestStatusErr asserts that the errors returned by this package are both a gRPC status error and still identifiable with errors.Is.
func TestStatusErr(t *testing.T) {
	t.Run("statusErr", func(t *testing.T) {
		err := statusErr(codes.FailedPrecondition, ErrNeedsRecovery)

		assert.Equal(t, codes.FailedPrecondition, status.Code(err))
		assert.Equal(t, ErrNeedsRecovery.Error(), status.Convert(err).Message())
		assert.ErrorIs(t, err, ErrNeedsRecovery)
		assert.NotErrorIs(t, err, ErrNoManifest)
	})

	t.Run("statusErrf", func(t *testing.T) {
		err := statusErrf(codes.PermissionDenied, ErrInvalidSignature, "validating manifest signature: %v", errors.New("boom"))

		assert.Equal(t, codes.PermissionDenied, status.Code(err))
		assert.Equal(t, "validating manifest signature: boom", status.Convert(err).Message())
		assert.ErrorIs(t, err, ErrInvalidSignature)
	})

	t.Run("wrapped sentinels stay reachable", func(t *testing.T) {
		inner := errors.Join(ErrInsecureNotAllowed, ErrMixedManifestNotAllowed)
		err := statusErr(codes.InvalidArgument, inner)

		assert.ErrorIs(t, err, ErrInsecureNotAllowed)
		assert.ErrorIs(t, err, ErrMixedManifestNotAllowed)
	})
}

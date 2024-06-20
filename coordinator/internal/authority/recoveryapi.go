// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package authority

import (
	"context"
	"errors"

	"github.com/edgelesssys/contrast/internal/recoveryapi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	// ErrAlreadyRecovered is returned if seedEngine initialization was requested but a seed is already set.
	ErrAlreadyRecovered = errors.New("coordinator is already recovered")
	// ErrNeedsRecovery is returned if state exists, but no secrets are available, e.g. after restart.
	ErrNeedsRecovery = errors.New("coordinator is in recovery mode")
)

// Recover recovers the Coordinator from a seed and salt.
func (a *Authority) Recover(_ context.Context, req *recoveryapi.RecoverRequest) (*recoveryapi.RecoverResponse, error) {
	a.logger.Info("Recover called")

	if err := a.initSeedEngine(req.Seed, req.Salt); errors.Is(err, ErrAlreadyRecovered) {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	} else if err != nil {
		// Pretty sure this failed because the seed was bad.
		return nil, status.Errorf(codes.InvalidArgument, "initializing seed engine: %v", err)
	}

	if err := a.syncState(); err != nil {
		// This recovery attempt did not lead to a good state, let's roll it back.
		a.se.Store(nil)
		return nil, status.Errorf(codes.InvalidArgument, "recovery failed and was rolled back: %v", err)
	}
	return &recoveryapi.RecoverResponse{}, nil
}

// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package probes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/edgelesssys/contrast/coordinator/internal/stateguard"
	"github.com/stretchr/testify/assert"
)

func TestStartupProbe(t *testing.T) {
	testCases := map[string]struct {
		userapiStartedFirst   bool
		meshapiStartedFirst   bool
		recoveryStartedFirst  bool
		userapiStartedSecond  bool
		meshapiStartedSecond  bool
		recoveryStartedSecond bool
		want503First          bool
		want503Second         bool
	}{
		"all immediately online": {
			userapiStartedFirst:   true,
			meshapiStartedFirst:   true,
			recoveryStartedFirst:  true,
			userapiStartedSecond:  true,
			meshapiStartedSecond:  true,
			recoveryStartedSecond: true,
			want503First:          false,
			want503Second:         false,
		},
		"userapi never starts": {
			userapiStartedFirst:   false,
			meshapiStartedFirst:   true,
			recoveryStartedFirst:  true,
			userapiStartedSecond:  false,
			meshapiStartedSecond:  true,
			recoveryStartedSecond: true,
			want503First:          true,
			want503Second:         true,
		},
		"meshapi never starts": {
			userapiStartedFirst:   true,
			meshapiStartedFirst:   false,
			recoveryStartedFirst:  true,
			userapiStartedSecond:  true,
			meshapiStartedSecond:  false,
			recoveryStartedSecond: true,
			want503First:          true,
			want503Second:         true,
		},
		"recovery never starts": {
			userapiStartedFirst:   true,
			meshapiStartedFirst:   true,
			recoveryStartedFirst:  false,
			userapiStartedSecond:  true,
			meshapiStartedSecond:  true,
			recoveryStartedSecond: false,
			want503First:          true,
			want503Second:         true,
		},
		"userapi starts later": {
			userapiStartedFirst:   false,
			meshapiStartedFirst:   true,
			recoveryStartedFirst:  true,
			userapiStartedSecond:  true,
			meshapiStartedSecond:  true,
			recoveryStartedSecond: true,
			want503First:          true,
			want503Second:         false,
		},
		"meshapi starts later": {
			userapiStartedFirst:   true,
			meshapiStartedFirst:   false,
			recoveryStartedFirst:  true,
			userapiStartedSecond:  true,
			meshapiStartedSecond:  true,
			recoveryStartedSecond: true,
			want503First:          true,
			want503Second:         false,
		},
		"recovery starts later": {
			userapiStartedFirst:   true,
			meshapiStartedFirst:   true,
			recoveryStartedFirst:  false,
			userapiStartedSecond:  true,
			meshapiStartedSecond:  true,
			recoveryStartedSecond: true,
			want503First:          true,
			want503Second:         false,
		},
		"all start later": {
			userapiStartedFirst:   false,
			meshapiStartedFirst:   false,
			recoveryStartedFirst:  false,
			userapiStartedSecond:  true,
			meshapiStartedSecond:  true,
			recoveryStartedSecond: true,
			want503First:          true,
			want503Second:         false,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			req := httptest.NewRequest(http.MethodGet, "/probes/startup", nil)
			resp := httptest.NewRecorder()

			mux := http.NewServeMux()

			var userapiStarted, meshapiStarted, recoveryStarted atomic.Bool

			userapiStarted.Store(tc.userapiStartedFirst)
			meshapiStarted.Store(tc.meshapiStartedFirst)
			recoveryStarted.Store(tc.recoveryStartedFirst)

			handler := StartupHandler{
				UserapiStarted:  &userapiStarted,
				MeshapiStarted:  &meshapiStarted,
				RecoveryStarted: &recoveryStarted,
			}

			mux.Handle("/probes/startup", &handler)

			mux.ServeHTTP(resp, req)

			if tc.want503First {
				assert.Equal(http.StatusServiceUnavailable, resp.Code)
			} else {
				assert.Equal(http.StatusOK, resp.Code)
			}

			resp = httptest.NewRecorder()
			userapiStarted.Store(tc.userapiStartedSecond)
			meshapiStarted.Store(tc.meshapiStartedSecond)
			recoveryStarted.Store(tc.recoveryStartedSecond)

			mux.ServeHTTP(resp, req)

			if tc.want503Second {
				assert.Equal(http.StatusServiceUnavailable, resp.Code)
			} else {
				assert.Equal(http.StatusOK, resp.Code)
			}
		})
	}
}

func TestReadinessProbe(t *testing.T) {
	testCases := map[string]struct {
		hasActiveManifest bool
		getStateFails     bool
		want503           bool
	}{
		"manifest exists": {
			hasActiveManifest: true,
			want503:           false,
		},
		"manifest doesn't exist": {
			hasActiveManifest: false,
			want503:           true,
		},
		"get state fails while manifest exists": {
			hasActiveManifest: true,
			getStateFails:     true,
			want503:           true,
		},
		"get state fails while no manifest exists": {
			hasActiveManifest: false,
			getStateFails:     true,
			want503:           true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			req := httptest.NewRequest(http.MethodGet, "/probes/readiness", nil)
			resp := httptest.NewRecorder()

			auth := mockAuth{hasState: tc.hasActiveManifest, fails: tc.getStateFails}

			mux := http.NewServeMux()

			handler := ReadinessHandler{Guard: auth}
			mux.Handle("/probes/readiness", handler)

			mux.ServeHTTP(resp, req)
			if tc.want503 {
				assert.Equal(http.StatusServiceUnavailable, resp.Code)
			} else {
				assert.Equal(http.StatusOK, resp.Code)
			}
		})
	}
}

type mockAuth struct {
	hasState bool
	fails    bool
}

func (a mockAuth) GetState(context.Context) (*stateguard.State, error) {
	if a.fails {
		return nil, assert.AnError
	}
	if !a.hasState {
		return nil, nil
	}
	return &stateguard.State{}, nil
}

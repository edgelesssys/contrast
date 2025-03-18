// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package probes

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/edgelesssys/contrast/coordinator/history"
	"github.com/edgelesssys/contrast/coordinator/internal/authority"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStartupProbe(t *testing.T) {
	testCases := map[string]struct {
		userapiStartedFirst  bool
		meshapiStartedFirst  bool
		userapiStartedSecond bool
		meshapiStartedSecond bool
		want503First         bool
		want503Second        bool
	}{
		"all immediately online": {
			userapiStartedFirst:  true,
			meshapiStartedFirst:  true,
			userapiStartedSecond: true,
			meshapiStartedSecond: true,
			want503First:         false,
			want503Second:        false,
		},
		"userapi never starts": {
			userapiStartedFirst:  false,
			meshapiStartedFirst:  true,
			userapiStartedSecond: false,
			meshapiStartedSecond: true,
			want503First:         true,
			want503Second:        true,
		},
		"meshapi never starts": {
			userapiStartedFirst:  true,
			meshapiStartedFirst:  false,
			userapiStartedSecond: true,
			meshapiStartedSecond: false,
			want503First:         true,
			want503Second:        true,
		},
		"userapi starts later": {
			userapiStartedFirst:  false,
			meshapiStartedFirst:  true,
			userapiStartedSecond: true,
			meshapiStartedSecond: true,
			want503First:         true,
			want503Second:        false,
		},
		"meshapi starts later": {
			userapiStartedFirst:  true,
			meshapiStartedFirst:  false,
			userapiStartedSecond: true,
			meshapiStartedSecond: true,
			want503First:         true,
			want503Second:        false,
		},
		"both start later": {
			userapiStartedFirst:  false,
			meshapiStartedFirst:  false,
			userapiStartedSecond: true,
			meshapiStartedSecond: true,
			want503First:         true,
			want503Second:        false,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			req := httptest.NewRequest(http.MethodGet, "/probes/startup", nil)
			resp := httptest.NewRecorder()

			mux := http.NewServeMux()

			userapiStarted := tc.userapiStartedFirst
			meshapiStarted := tc.meshapiStartedFirst

			handler := StartupHandler{MeshapiStarted: &userapiStarted, UserapiStarted: &meshapiStarted}

			mux.Handle("/probes/startup", handler)

			mux.ServeHTTP(resp, req)

			if tc.want503First {
				assert.Equal(http.StatusServiceUnavailable, resp.Code)
			} else {
				assert.Equal(http.StatusOK, resp.Code)
			}

			resp = httptest.NewRecorder()
			userapiStarted = tc.userapiStartedSecond
			meshapiStarted = tc.meshapiStartedSecond

			mux.ServeHTTP(resp, req)

			if tc.want503Second {
				assert.Equal(http.StatusServiceUnavailable, resp.Code)
			} else {
				assert.Equal(http.StatusOK, resp.Code)
			}
		})
	}
}

func TestLivenessProbe(t *testing.T) {
	someTransition := "asdf"

	testCases := map[string]struct {
		hasTransition bool
		want503       bool
	}{
		"transition exists": {
			hasTransition: true,
			want503:       false,
		},
		"transition doesn't exist": {
			hasTransition: false,
			want503:       true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			fs := afero.Afero{Fs: afero.NewMemMapFs()}

			req := httptest.NewRequest(http.MethodGet, "/probes/liveness", nil)
			resp := httptest.NewRecorder()

			mux := http.NewServeMux()

			if tc.hasTransition {
				require.NoError(fs.WriteFile("transitions/latest", []byte(someTransition), 0o644))
			}

			hist := history.NewWithStore(history.NewAferoStore(&fs))

			handler := LivenessHandler{Hist: hist}
			mux.Handle("/probes/liveness", handler)

			mux.ServeHTTP(resp, req)
			if tc.want503 {
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

			handler := ReadinessHandler{Authority: auth}
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

func (a mockAuth) GetState() (*authority.State, error) {
	if a.fails {
		return nil, errors.New("some error")
	}
	if !a.hasState {
		return nil, nil
	}
	return &authority.State{}, nil
}

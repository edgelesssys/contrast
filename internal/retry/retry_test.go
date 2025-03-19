// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package retry

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
	testclock "k8s.io/utils/clock/testing"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m, goleak.IgnoreAnyFunction("github.com/bazelbuild/rules_go/go/tools/bzltestutil.RegisterTimeoutHandler.func1"))
}

func TestDo(t *testing.T) {
	testCases := map[string]struct {
		cancel    bool
		errors    []error
		retriable func(error) bool
		wantErr   error
	}{
		"no error": {
			errors: []error{
				nil,
			},
		},
		"permanent error": {
			errors: []error{
				errPermanent,
			},
			wantErr: errPermanent,
		},
		"service unavailable then success": {
			errors: []error{
				errRetriable,
				nil,
			},
		},
		"service unavailable then permanent error": {
			errors: []error{
				errRetriable,
				errPermanent,
			},
			wantErr: errPermanent,
		},
		"cancellation results in last error": {
			cancel: true,
			errors: []error{
				errRetriable,
				nil,
			},
			wantErr: errRetriable,
		},
		"predicate Always": {
			errors: []error{
				errPermanent,
				nil,
			},
			retriable: Always,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			predicate := isRetriable
			if tc.retriable != nil {
				predicate = tc.retriable
			}
			assert := assert.New(t)
			doer := newStubDoer(tc.errors)
			retrier := NewIntervalRetrier(doer, doer.interval, predicate, doer.clock)
			ctx, cancel := context.WithCancel(context.Background())
			if tc.cancel {
				cancel()
			} else {
				defer cancel()
			}

			assert.ErrorIs(retrier.Do(ctx), tc.wantErr)
		})
	}
}

type stubDoer struct {
	errs     []error
	clock    *testclock.FakeClock
	interval time.Duration

	count int
}

func newStubDoer(errs []error) *stubDoer {
	return &stubDoer{
		errs:     errs,
		clock:    testclock.NewFakeClock(time.Now().Add(-12 * time.Hour)),
		interval: time.Second,
	}
}

func (d *stubDoer) Do(_ context.Context) error {
	d.count++
	d.clock.Step(d.interval)
	return d.errs[d.count-1]
}

var (
	errRetriable = errors.New("retry me")
	errPermanent = errors.New("error")
)

func isRetriable(err error) bool {
	return errors.Is(err, errRetriable)
}

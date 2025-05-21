// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package spinner

import (
	"fmt"
	"io"
	"sync/atomic"
	"time"

	"k8s.io/utils/clock"
)

// Spinner is a simple progress indicator.
type Spinner struct {
	prefix       string
	ticker       clock.Ticker
	out          io.Writer
	callback     func(callbackEventType) // called after each dot. Only used for testing.
	stopChan     chan string
	stopDoneChan chan struct{}
	running      atomic.Bool
}

// New creates a new Spinner writing to out with a prefix printed on Start.
// The interval is the time between each dot printed.
func New(prefix string, interval time.Duration, out io.Writer) *Spinner {
	ticker := clock.RealClock{}.NewTicker(interval)
	return &Spinner{
		prefix:       prefix,
		ticker:       ticker,
		out:          out,
		stopChan:     make(chan string, 1),
		stopDoneChan: make(chan struct{}, 1),
	}
}

// Start starts the spinner.
func (s *Spinner) Start() {
	if !s.running.CompareAndSwap(false, true) {
		// already running
		return
	}
	go func() {
		s.paint(s.prefix, callbackEventTypeStart)
		defer func() {
			s.ticker.Stop()
			close(s.stopDoneChan)
			s.running.Store(false)
		}()
		for {
			select {
			case message := <-s.stopChan:
				s.paint(message, callbackEventTypeStop)
				return
			case <-s.ticker.C():
				s.paint(".", callbackEventTypeTick)
			}
		}
	}()
}

// Stop stops the spinner and prints the given message.
func (s *Spinner) Stop(message string) {
	if !s.running.Load() {
		return // not running
	}
	select {
	case s.stopChan <- message: // sent stop message
	default: // stopChan already has a message
	}
	<-s.stopDoneChan // wait for spinner to stop
}

func (s *Spinner) paint(msg string, eventType callbackEventType) {
	fmt.Fprint(s.out, msg)
	if s.callback != nil {
		s.callback(eventType)
	}
}

type callbackEventType int

const (
	callbackEventTypeTick callbackEventType = iota
	callbackEventTypeStop
	callbackEventTypeStart
)

// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package genpolicy

import (
	"bufio"
	"io"
	"log/slog"
	"regexp"
)

type logTranslator struct {
	r         *io.PipeReader
	w         *io.PipeWriter
	logger    *slog.Logger
	stopDoneC chan struct{}
}

func newLogTranslator(logger *slog.Logger) logTranslator {
	r, w := io.Pipe()
	l := logTranslator{
		r:         r,
		w:         w,
		logger:    logger,
		stopDoneC: make(chan struct{}),
	}
	l.startTranslate()
	return l
}

func (l logTranslator) Write(p []byte) (n int, err error) {
	return l.w.Write(p)
}

var genpolicyLogPrefixReg = regexp.MustCompile(`^\[[^\]\s]+\s+(\w+)\s+([^\]\s]+)\] (.*)`)

func (l logTranslator) startTranslate() {
	go func() {
		defer close(l.stopDoneC)
		scanner := bufio.NewScanner(l.r)
		for scanner.Scan() {
			line := scanner.Text()
			match := genpolicyLogPrefixReg.FindStringSubmatch(line)
			if len(match) != 4 {
				// genpolicy prints some warnings without the logger
				l.logger.Warn(line)
			} else {
				switch match[1] {
				case "ERROR":
					l.logger.Error(match[3], "position", match[2])
				case "WARN":
					l.logger.Warn(match[3], "position", match[2])
				case "INFO": // prints quite a lot, only show on debug
					l.logger.Debug(match[3], "position", match[2])
				}
			}
		}
	}()
}

func (l logTranslator) stop() {
	l.w.Close()
	<-l.stopDoneC
}

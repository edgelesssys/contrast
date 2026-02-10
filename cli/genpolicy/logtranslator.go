// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

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

var (
	genpolicyLogPrefixReg = regexp.MustCompile(`^\[[^\]\s]+\s+(\w+)\s+([^\]\s]+)\] (.*)`)
	errorMessage          = regexp.MustCompile(`^thread\s+'main'\s+(?:\(\d+\)\s+)?panicked\s+at`)
)

func (l logTranslator) startTranslate() {
	go func() {
		defer close(l.stopDoneC)
		scanner := bufio.NewScanner(l.r)
		// Default log level is initially set to 'WARN'. This is only relevant if the first line does not match the logging pattern.
		logLevel := "WARN"

		for scanner.Scan() {
			line := scanner.Text()
			match := genpolicyLogPrefixReg.FindStringSubmatch(line)
			if len(match) != 4 {
				// genpolicy prints some warnings without the logger
				// we continue logging on the latest used log-level

				// Error is written to stderr by genpolicy without using the logger,
				// simple regex to detect the error message on stderr
				if errorMessage.MatchString(line) {
					logLevel = "ERROR"
				}
				switch logLevel {
				case "ERROR":
					l.logger.Error(line)
				case "WARN":
					l.logger.Warn(line)
				case "INFO":
					fallthrough
				case "DEBUG":
					l.logger.Debug(line)
				}
			} else {
				switch match[1] {
				case "ERROR":
					l.logger.Error(match[3], "position", match[2])
				case "WARN":
					l.logger.Warn(match[3], "position", match[2])
				case "INFO":
					fallthrough // prints quite a lot, only show on debug
				case "DEBUG":
					l.logger.Debug(match[3], "position", match[2])
				}
				// Update the latest log level
				logLevel = match[1]
			}

		}
	}()
}

func (l logTranslator) stop() {
	l.w.Close()
	<-l.stopDoneC
}

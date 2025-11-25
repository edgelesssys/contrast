// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

// A best-effort implementation of a TDX event log of the precalculation
// to compare against actual TDX event logs.
package eventlog

import (
	"fmt"
	"os"
	"path"
)

// MrtdLogger is a logger for MRTD event logs.
type MrtdLogger struct {
	dir string
	buf []byte
}

// NewMrtdLogger creates a new [MrtdLogger] at the given directory
// and writes the TDH_MNG_INIT entry.
func NewMrtdLogger(dir string) *MrtdLogger {
	l := &MrtdLogger{dir, []byte{}}
	l.log([]byte("TDH_MNG_INIT\n"))
	return l
}

// MemPageAdd logs a TDH_MEM_PAGE_ADD event for the given GPA.
func (l *MrtdLogger) MemPageAdd(gpa uint64) {
	l.log([]byte(fmt.Sprintf("TDH_MEM_PAGE_ADD gpa=0x%x\n", gpa)))
}

// MrExtend logs a TDH_MR_EXTEND event for the given GPA.
func (l *MrtdLogger) MrExtend(gpa uint64) {
	l.log([]byte(fmt.Sprintf("TDH_MR_EXTEND gpa=0x%x\n", gpa)))
}

// SaveToFile writes the `TDH_MR_FINALIZE` event and saves the event log
// to `mrtd.log` in the event log directory.
func (l *MrtdLogger) SaveToFile() error {
	l.log([]byte("TDH_MR_FINALIZE\n"))
	if l.dir == "" {
		return nil
	}
	if err := os.WriteFile(path.Join(l.dir, "mrtd.log"), l.buf, 0o644); err != nil {
		return fmt.Errorf("writing mrtd eventlog to file: %w", err)
	}
	return nil
}

// log appends an entry to the event log buffer.
func (l *MrtdLogger) log(entry []byte) {
	l.buf = append(l.buf, entry...)
}

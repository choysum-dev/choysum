// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

// Package tap writes Test Anything Protocol (TAP) version 13-style streams
// used by QuickJS unit and e2e hosts.
package tap

import (
	"fmt"
	"io"
	"strconv"
)

// CaseError is an optional failure payload for a TAP case line.
type CaseError struct {
	Message string
	Stack   string
}

// Case is one TAP test point.
type Case struct {
	Name  string
	OK    bool
	Error *CaseError
}

// Report is a TAP plan plus case results.
type Report struct {
	Total int
	Cases []Case
}

// Write writes a TAP stream: "1..N" then ok/not ok lines.
// Failed cases include a YAML-ish diagnostic block with a quoted message.
func Write(w io.Writer, report *Report) {
	if w == nil || report == nil {
		return
	}
	fmt.Fprintf(w, "1..%d\n", report.Total)
	for i, c := range report.Cases {
		n := i + 1
		if c.OK {
			fmt.Fprintf(w, "ok %d - %s\n", n, c.Name)
			continue
		}
		msg := "failed"
		if c.Error != nil && c.Error.Message != "" {
			msg = c.Error.Message
			if c.Error.Stack != "" {
				msg = msg + "\n" + c.Error.Stack
			}
		}
		fmt.Fprintf(w, "not ok %d - %s\n  ---\n  message: %s\n  ...\n", n, c.Name, strconv.Quote(msg))
	}
}

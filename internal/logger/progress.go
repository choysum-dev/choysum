// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package logger

import (
	"fmt"
	"io"
	"os"
	"sync"
)

// braille spinner frames (same as npm)
var progressSpinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// ProgressLine provides a single-line in-place progress indicator.
// When the writer is not a terminal, Update is a no-op and Done falls
// back to plain fmt.Fprintln.
type ProgressLine struct {
	w   io.Writer
	tty bool
	mu  sync.Mutex
}

// NewProgressLine creates a ProgressLine that writes to w.
// Returns nil when w is nil.
func NewProgressLine(w io.Writer) *ProgressLine {
	if w == nil {
		return nil
	}
	return &ProgressLine{
		w:   w,
		tty: consoleTerminalWriter(w) && os.Getenv("NO_COLOR") == "",
	}
}

// Update overwrites the current line with a spinner frame and message.
// frame is typically an incrementing counter; the braille glyph is
// selected via frame % len(progressSpinnerFrames).
func (p *ProgressLine) Update(frame int, message string) {
	if p == nil || !p.tty {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	glyph := progressSpinnerFrames[frame%len(progressSpinnerFrames)]
	fmt.Fprintf(p.w, "\r\x1b[K%s %s", glyph, message)
}

// Done finalises the line with a result symbol and message, then
// moves to the next line. symbol is typically "✓" or "✗".
func (p *ProgressLine) Done(symbol, message string) {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.tty {
		fmt.Fprintln(p.w, message)
		return
	}
	fmt.Fprintf(p.w, "\r\x1b[K%s %s\n", symbol, message)
}

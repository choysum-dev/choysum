// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package logger

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"
)

// braille spinner frames (same as npm)
var progressSpinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

type progressLineContextKey struct{}

var progressOutputBarrier struct {
	mu     sync.Mutex
	active bool
}

type progressAwareWriter struct {
	w io.Writer
}

func wrapProgressAwareWriter(w io.Writer) io.Writer {
	if w == nil {
		return nil
	}
	if wrapped, ok := w.(*progressAwareWriter); ok {
		return wrapped
	}
	return &progressAwareWriter{w: w}
}

func (w *progressAwareWriter) Write(p []byte) (int, error) {
	progressOutputBarrier.mu.Lock()
	defer progressOutputBarrier.mu.Unlock()
	if progressOutputBarrier.active {
		_, _ = fmt.Fprint(w.w, "\r\x1b[K\n")
		progressOutputBarrier.active = false
	}
	return w.w.Write(p)
}

// ProgressLine provides a single-line in-place progress indicator.
// When the writer is not a terminal, Update is a no-op and Done falls
// back to plain fmt.Fprintln.
type ProgressLine struct {
	w   io.Writer
	tty bool
}

// WithProgressLine stores a ProgressLine in ctx so downstream operations can
// reuse the same terminal line for progress updates.
func WithProgressLine(ctx context.Context, line *ProgressLine) context.Context {
	if line == nil {
		return ctx
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, progressLineContextKey{}, line)
}

// ProgressLineFromContext returns a ProgressLine from ctx when present.
func ProgressLineFromContext(ctx context.Context) *ProgressLine {
	if ctx == nil {
		return nil
	}
	line, _ := ctx.Value(progressLineContextKey{}).(*ProgressLine)
	return line
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
	progressOutputBarrier.mu.Lock()
	defer progressOutputBarrier.mu.Unlock()
	glyph := progressSpinnerFrames[frame%len(progressSpinnerFrames)]
	fmt.Fprintf(p.w, "\r\x1b[K%s %s", glyph, message)
	progressOutputBarrier.active = true
}

// Done finalises the line with a result symbol and message, then
// moves to the next line. symbol is typically "✓" or "✗".
func (p *ProgressLine) Done(symbol, message string) {
	if p == nil {
		return
	}
	progressOutputBarrier.mu.Lock()
	defer progressOutputBarrier.mu.Unlock()
	if !p.tty {
		fmt.Fprintln(p.w, message)
		return
	}
	fmt.Fprintf(p.w, "\r\x1b[K%s %s\n", symbol, message)
	progressOutputBarrier.active = false
}

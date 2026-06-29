// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package logger

import (
	"context"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"sync"
	"time"
)

// braille spinner frames (same as npm)
var progressSpinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

type progressLineContextKey struct{}
type progressTickerContextKey struct{}

var progressOutputBarrier struct {
	mu     sync.Mutex
	active bool
	writer io.Writer
}

type progressAwareWriter struct {
	w io.Writer
}

func (w *progressAwareWriter) Unwrap() io.Writer {
	if w == nil {
		return nil
	}
	return w.w
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

func unwrapProgressWriter(w io.Writer) io.Writer {
	for {
		wrapped, ok := w.(*progressAwareWriter)
		if !ok || wrapped == nil {
			return w
		}
		w = wrapped.w
	}
}

func sameProgressWriter(left, right io.Writer) bool {
	left = unwrapProgressWriter(left)
	right = unwrapProgressWriter(right)
	if left == nil || right == nil {
		return false
	}
	leftType := reflect.TypeOf(left)
	rightType := reflect.TypeOf(right)
	if leftType != rightType || !leftType.Comparable() {
		return false
	}
	return left == right
}

func (w *progressAwareWriter) Write(p []byte) (int, error) {
	progressOutputBarrier.mu.Lock()
	shouldClear := progressOutputBarrier.active && sameProgressWriter(progressOutputBarrier.writer, w.w)
	if shouldClear {
		progressOutputBarrier.active = false
		progressOutputBarrier.writer = nil
	}
	progressOutputBarrier.mu.Unlock()

	if shouldClear {
		_, _ = fmt.Fprint(w.w, "\r\x1b[K")
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

func (p *ProgressLine) IsTTY() bool {
	return p != nil && p.tty
}

const defaultProgressTickerInterval = 120 * time.Millisecond

// ProgressTickerOptions configures the interval and per-tick behaviour
// of a ProgressTicker.
type ProgressTickerOptions struct {
	Interval time.Duration
	OnTick   func(now time.Time, message string)
}

// ProgressTicker continuously refreshes a single-line progress message.
// Use SetMessage to update the current message, Clear to hide it, and Stop
// to terminate the background ticker goroutine.
type ProgressTicker struct {
	line     *ProgressLine
	interval time.Duration
	onTick   func(now time.Time, message string)

	mu      sync.Mutex
	frame   int
	message string

	stopOnce sync.Once
	stopCh   chan struct{}
	doneCh   chan struct{}
}

// NewProgressTicker starts a ProgressTicker that redraws line at the
// configured interval.  When line is nil the ticker runs without rendering
// and is useful for pure heartbeat callbacks.
func NewProgressTicker(line *ProgressLine, opts ProgressTickerOptions) *ProgressTicker {
	interval := opts.Interval
	if interval <= 0 {
		interval = defaultProgressTickerInterval
	}
	ticker := &ProgressTicker{
		line:     line,
		interval: interval,
		onTick:   opts.OnTick,
	}
	if line == nil {
		return ticker
	}

	ticker.stopCh = make(chan struct{})
	ticker.doneCh = make(chan struct{})
	go ticker.run()
	return ticker
}

// WithProgressTicker stores a ProgressTicker in ctx so nested operations can
// reuse the same redraw loop instead of creating competing ticker goroutines.
func WithProgressTicker(ctx context.Context, ticker *ProgressTicker) context.Context {
	if ticker == nil {
		return ctx
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, progressTickerContextKey{}, ticker)
}

// ProgressTickerFromContext returns a ProgressTicker from ctx when present.
func ProgressTickerFromContext(ctx context.Context) *ProgressTicker {
	if ctx == nil {
		return nil
	}
	ticker, _ := ctx.Value(progressTickerContextKey{}).(*ProgressTicker)
	return ticker
}

func (p *ProgressTicker) run() {
	defer close(p.doneCh)
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-p.stopCh:
			return
		case now := <-ticker.C:
			p.mu.Lock()
			message := p.message
			if message != "" {
				p.frame++
				if p.line != nil {
					p.line.Update(p.frame, message)
				}
			}
			onTick := p.onTick
			p.mu.Unlock()

			if onTick != nil {
				onTick(now, message)
			}
		}
	}
}

// SetMessage updates the current ticker message and performs an immediate redraw.
func (p *ProgressTicker) SetMessage(message string) {
	if p == nil {
		return
	}
	message = strings.TrimSpace(message)
	if message == "" {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	p.message = message
	p.frame++
	if p.line != nil {
		p.line.Update(p.frame, message)
	}
}

// Clear removes the current message so the background ticker no longer redraws.
func (p *ProgressTicker) Clear() {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.message = ""
	if p.line != nil {
		p.line.Clear()
	}
}

// Stop terminates the background ticker goroutine.
func (p *ProgressTicker) Stop() {
	if p == nil || p.stopCh == nil {
		return
	}
	p.stopOnce.Do(func() {
		close(p.stopCh)
		<-p.doneCh
	})
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
	glyph := progressSpinnerFrames[frame%len(progressSpinnerFrames)]
	progressOutputBarrier.mu.Lock()
	progressOutputBarrier.active = true
	progressOutputBarrier.writer = unwrapProgressWriter(p.w)
	progressOutputBarrier.mu.Unlock()

	fmt.Fprintf(p.w, "\r\x1b[K%s %s", glyph, message)
}

// Clear erases the currently rendered progress line without emitting a newline.
func (p *ProgressLine) Clear() {
	if p == nil || !p.tty {
		return
	}
	progressOutputBarrier.mu.Lock()
	if sameProgressWriter(progressOutputBarrier.writer, p.w) {
		progressOutputBarrier.active = false
		progressOutputBarrier.writer = nil
	}
	progressOutputBarrier.mu.Unlock()

	fmt.Fprint(p.w, "\r\x1b[K")
}

// Done finalises the line with a result symbol and message, then
// moves to the next line. symbol is typically "✓" or "✗".
func (p *ProgressLine) Done(symbol, message string) {
	if p == nil {
		return
	}
	if !p.tty {
		fmt.Fprintln(p.w, message)
		return
	}
	progressOutputBarrier.mu.Lock()
	if sameProgressWriter(progressOutputBarrier.writer, p.w) {
		progressOutputBarrier.active = false
		progressOutputBarrier.writer = nil
	}
	progressOutputBarrier.mu.Unlock()

	fmt.Fprintf(p.w, "\r\x1b[K%s %s\n", symbol, message)
}

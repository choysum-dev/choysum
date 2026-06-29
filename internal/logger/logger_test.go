// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package logger

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/choysum-dev/choysum/pkg/config"
)

type frameHolder struct {
	frames [3]uintptr
}

type errWithFrame struct {
	frame frameHolder
	msg   string
}

func (e *errWithFrame) Error() string { return e.msg }

type quickjsErr struct {
	Stack string
	msg   string
}

func (e quickjsErr) Error() string { return e.msg }

func callerPCs() [3]uintptr {
	pcs := [3]uintptr{}
	n := runtime.Callers(2, pcs[:])
	if n == 0 {
		return pcs
	}
	return pcs
}

func testLogConfig(format string, level string) *config.LogConfig {
	return &config.LogConfig{Format: format, Level: level}
}

func stubConsoleTerminalWriter(t *testing.T, value bool) {
	t.Helper()
	original := consoleTerminalWriter
	consoleTerminalWriter = func(io.Writer) bool { return value }
	t.Cleanup(func() { consoleTerminalWriter = original })
}

func attrMap(attrs []slog.Attr) map[string]slog.Value {
	result := make(map[string]slog.Value, len(attrs))
	for _, attr := range attrs {
		result[attr.Key] = attr.Value
	}
	return result
}

var ansiRegexp = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(in string) string {
	return ansiRegexp.ReplaceAllString(in, "")
}

type concurrentBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (w *concurrentBuffer) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.Write(p)
}

func (w *concurrentBuffer) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.String()
}

func TestExtractPCFromErrorAndGetFileLineFromPC(t *testing.T) {
	pcs := callerPCs()
	err := &errWithFrame{frame: frameHolder{frames: pcs}, msg: "boom"}
	extracted := extractPCFromError(err)
	if len(extracted) == 0 {
		t.Fatal("expected program counters to be extracted")
	}

	lines := getFileLineFromPC(extracted)
	if len(lines) == 0 {
		t.Fatal("expected file/line data from program counters")
	}
	if !strings.Contains(lines[0], ":") || !strings.Contains(lines[0], ",") {
		t.Fatalf("unexpected frame line format: %q", lines[0])
	}

	if got := extractPCFromError(errors.New("plain")); got != nil {
		t.Fatalf("expected nil PCs for plain error, got %#v", got)
	}
	if got := getFileLineFromPC(nil); got != nil {
		t.Fatalf("expected nil file lines for nil PCs, got %#v", got)
	}
}

func TestExtractFromQuickjsError(t *testing.T) {
	err := quickjsErr{
		msg:   "quickjs failed",
		Stack: "ReferenceError\n    at app.vue:12\n    at helper.ts:3\nignored",
	}
	lines := extractFromQuickjsError(err)
	if len(lines) != 2 || lines[0] != "app.vue:12" || lines[1] != "helper.ts:3" {
		t.Fatalf("unexpected quickjs lines: %#v", lines)
	}

	if got := extractFromQuickjsError(errors.New("plain")); got != nil {
		t.Fatalf("expected nil for non-quickjs error, got %#v", got)
	}
}

func TestFmtErrAndReplaceAttr(t *testing.T) {
	wrapped := fmt.Errorf("outer: %w", quickjsErr{msg: "inner", Stack: "at page.ts:10"})
	trace := collectErrorTrace(wrapped)
	if trace.Msg != wrapped.Error() {
		t.Fatalf("collectErrorTrace() msg = %q, want %q", trace.Msg, wrapped.Error())
	}
	if !trace.HasFrames() {
		t.Fatal("expected collected error trace to include frames")
	}
	if len(trace.Trace) != 2 {
		t.Fatalf("collectErrorTrace() entry count = %d, want 2", len(trace.Trace))
	}
	if got := trace.Trace[1].Frames[0].File; got != "page.ts:10" {
		t.Fatalf("collectErrorTrace() quickjs frame = %q, want %q", got, "page.ts:10")
	}
	if collectErrorTrace(errors.New("plain")).HasFrames() {
		t.Fatal("expected plain error trace to have no frames")
	}

	value := fmtErr(wrapped)
	attrs := attrMap(value.Group())
	if attrs["msg"].String() != wrapped.Error() {
		t.Fatalf("fmtErr msg = %q, want %q", attrs["msg"].String(), wrapped.Error())
	}
	if attrs["trace"].Kind() != slog.KindAny {
		t.Fatalf("expected trace attr to be slog.Any, got %v", attrs["trace"].Kind())
	}

	replaced := replaceAttr(nil, slog.Any("err", wrapped))
	if replaced.Value.Kind() != slog.KindGroup {
		t.Fatalf("expected error attr to be converted to group, got %v", replaced.Value.Kind())
	}
	if unchanged := replaceAttr(nil, slog.String("msg", "ok")); unchanged.Value.String() != "ok" {
		t.Fatalf("expected non-error attr to remain unchanged, got %#v", unchanged)
	}
}

func TestNewHandlerAndLoggerWithWriter(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	stubConsoleTerminalWriter(t, false)

	jsonBuf := bytes.NewBuffer(nil)
	jsonLogger := NewLoggerWithWriter(testLogConfig("json", "debug"), jsonBuf)
	jsonLogger.Error("boom", slog.Any("err", errors.New("bad")))
	jsonOut := jsonBuf.String()
	if !strings.Contains(jsonOut, "\"msg\":\"boom\"") || !strings.Contains(jsonOut, "\"trace\"") {
		t.Fatalf("unexpected json logger output: %q", jsonOut)
	}
	if !strings.Contains(jsonOut, "\"system\":\"choysum\"") {
		t.Fatalf("expected system attr in json logger output, got %q", jsonOut)
	}
	if strings.Contains(jsonOut, "\"engine\":") || strings.Contains(jsonOut, "\"distribution\":") {
		t.Fatalf("expected engine/distribution attrs to be removed, got %q", jsonOut)
	}

	textBuf := bytes.NewBuffer(nil)
	textLogger := NewLoggerWithWriter(testLogConfig("text", "warn"), textBuf)
	textLogger.Info("skip me")
	if textBuf.Len() != 0 {
		t.Fatalf("expected info log to be filtered at warn level, got %q", textBuf.String())
	}
	textLogger.Error("visible")
	if !strings.Contains(textBuf.String(), "visible") {
		t.Fatalf("expected error log in text output, got %q", textBuf.String())
	}

	devBuf := bytes.NewBuffer(nil)
	devLogger := NewLoggerWithWriter(testLogConfig("devslog", "info"), devBuf)
	devLogger.Info("dev-mode")
	if !strings.Contains(devBuf.String(), "dev-mode") {
		t.Fatalf("expected devslog output, got %q", devBuf.String())
	}

	if logger := NewLoggerWithWriter(testLogConfig("json", "info"), nil); logger == nil {
		t.Fatal("expected logger when writer is nil")
	}
	if logger := NewLogger(testLogConfig("text", "info")); logger == nil {
		t.Fatal("expected NewLogger to return a logger")
	}
	if handler := newHandler(testLogConfig("unknown", "error"), io.Discard); handler == nil {
		t.Fatal("expected fallback text handler")
	}
}

func TestProgressLineKeepsStructuredLogsOnSeparateLine(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	stubConsoleTerminalWriter(t, true)

	buf := bytes.NewBuffer(nil)
	line := NewProgressLine(buf)
	if line == nil {
		t.Fatal("expected progress line")
	}

	logger := NewLoggerWithWriter(testLogConfig("text", "info"), buf)
	line.Update(0, "extracting package")
	logger.Info("origin registry fetch completed", "module", "core")
	line.Done("✓", "core module installation completed")

	out := buf.String()
	if strings.Contains(out, "extracting packagetime=") || strings.Contains(out, "extracting packagelevel=") {
		t.Fatalf("expected spinner and structured log to stay on separate lines, got %q", out)
	}
	if !strings.Contains(out, "\r\x1b[K") {
		t.Fatalf("expected progress barrier line clear before structured log, got %q", out)
	}
	if !strings.Contains(out, "msg=\"origin registry fetch completed\"") {
		t.Fatalf("expected structured log output, got %q", out)
	}
}

func TestProgressBarrierIsScopedPerWriter(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	stubConsoleTerminalWriter(t, true)

	stderrBuf := bytes.NewBuffer(nil)
	stdoutBuf := bytes.NewBuffer(nil)

	line := NewProgressLine(stderrBuf)
	if line == nil {
		t.Fatal("expected progress line")
	}
	line.Update(0, "fetching")

	wrappedStdout := wrapProgressAwareWriter(stdoutBuf)
	if _, err := wrappedStdout.Write([]byte("stdout message\n")); err != nil {
		t.Fatalf("write stdout: %v", err)
	}
	if strings.Contains(stdoutBuf.String(), "\r\x1b[K") {
		t.Fatalf("expected stdout stream to remain untouched by stderr progress barrier, got %q", stdoutBuf.String())
	}

	wrappedStderr := wrapProgressAwareWriter(stderrBuf)
	if _, err := wrappedStderr.Write([]byte("stderr message\n")); err != nil {
		t.Fatalf("write stderr: %v", err)
	}
	if got := strings.Count(stderrBuf.String(), "\r\x1b[K"); got < 2 {
		t.Fatalf("expected stderr stream to include progress clear from wrapped write, got %q", stderrBuf.String())
	}
}

func TestProgressTickerClearErasesRenderedLine(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	stubConsoleTerminalWriter(t, true)

	buf := bytes.NewBuffer(nil)
	line := NewProgressLine(buf)
	if line == nil {
		t.Fatal("expected progress line")
	}

	ticker := NewProgressTicker(line, ProgressTickerOptions{})
	defer ticker.Stop()

	ticker.SetMessage("installing")
	before := strings.Count(buf.String(), "\r\x1b[K")
	ticker.Clear()
	after := strings.Count(buf.String(), "\r\x1b[K")
	if after <= before {
		t.Fatalf("expected Clear() to erase rendered line, got output %q", buf.String())
	}
}

func TestUnwrapTerminalWriterPreservesUnderlyingWriter(t *testing.T) {
	underlying := bytes.NewBuffer(nil)
	wrapped := wrapProgressAwareWriter(underlying)
	if got := unwrapTerminalWriter(wrapped); got != underlying {
		t.Fatalf("unwrapTerminalWriter() = %#v, want %#v", got, underlying)
	}
}

func TestDevslogHandlerTraceErrorKeepsQuickJSFrames(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	stubConsoleTerminalWriter(t, false)

	buf := bytes.NewBuffer(nil)
	logger := NewLoggerWithWriter(testLogConfig("devslog", "error"), buf)
	err := fmt.Errorf(
		"failed to call function: %w",
		quickjsErr{msg: "ChoysumError: User not found or password is incorrect", Stack: "at page.ts:10\nat helper.ts:3"},
	)
	logger.Error("js interpreter failed", "error", err, "code", "USER_NOT_FOUND")
	out := stripANSI(buf.String())
	if !strings.Contains(out, "js interpreter failed") {
		t.Fatalf("expected devslog error message, got %q", out)
	}
	if !strings.Contains(out, "E error") {
		t.Fatalf("expected devslog to keep error layout, got %q", out)
	}
	if !strings.Contains(out, "page.ts:10") || !strings.Contains(out, "helper.ts:3") {
		t.Fatalf("expected quickjs stack frames in devslog output, got %q", out)
	}
	if strings.Contains(out, "G error") || strings.Contains(out, "S trace") || strings.Contains(out, "map[string]interface {}") {
		t.Fatalf("expected devslog output without group/map trace artifacts, got %q", out)
	}
}

func TestNewHandlerAutoSelectsTextForNonTerminal(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	stubConsoleTerminalWriter(t, false)

	buf := bytes.NewBuffer(nil)
	logger := NewLoggerWithWriter(testLogConfig("", "info"), buf)
	logger.Info("plain text")
	out := buf.String()
	if !strings.Contains(out, "level=INFO") || !strings.Contains(out, "msg=\"plain text\"") {
		t.Fatalf("expected text handler output for non-terminal writer, got %q", out)
	}
}

func TestNewHandlerAutoSelectsConsoleForTerminal(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	stubConsoleTerminalWriter(t, true)

	buf := bytes.NewBuffer(nil)
	logger := NewLoggerWithWriter(testLogConfig("", "info"), buf)
	logger.Info("console auto", "services", []string{"base", "task", "web"})
	out := buf.String()
	if !strings.Contains(out, "INFO choysum: console auto") {
		t.Fatalf("expected console header in auto console output, got %q", out)
	}
	if !strings.Contains(out, "services=[base, task, web]") {
		t.Fatalf("expected console slice rendering, got %q", out)
	}
	if strings.Contains(out, "system=") || strings.Contains(out, "level=") {
		t.Fatalf("expected console layout instead of text layout, got %q", out)
	}
	if strings.Contains(out, "logger_test.go:") {
		t.Fatalf("expected info output to omit source, got %q", out)
	}
}

func TestConsoleHandlerWarnShowsSourceAndRelativePath(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	stubConsoleTerminalWriter(t, false)

	buf := bytes.NewBuffer(nil)
	logger := NewLoggerWithWriter(testLogConfig("console", "debug"), buf)
	logger.Warn("warn message", "count", 2)
	out := buf.String()
	if !strings.Contains(out, "WARN choysum [") || !strings.Contains(out, "logger_test.go:") {
		t.Fatalf("expected warn output to include source, got %q", out)
	}
	if cwd, err := os.Getwd(); err == nil && strings.Contains(out, cwd) {
		t.Fatalf("expected source path to be shortened, got %q", out)
	}
	if !strings.Contains(out, "count=2") {
		t.Fatalf("expected warn attrs in console output, got %q", out)
	}
}

func TestConsoleHandlerColorFollowsTerminalAndNoColor(t *testing.T) {
	stubConsoleTerminalWriter(t, true)

	buf := bytes.NewBuffer(nil)
	logger := NewLoggerWithWriter(testLogConfig("console", "info"), buf)
	logger.Info("colored")
	if !strings.Contains(buf.String(), "\x1b[") {
		t.Fatalf("expected ANSI color codes for terminal console output, got %q", buf.String())
	}

	t.Setenv("NO_COLOR", "1")
	buf.Reset()
	logger = NewLoggerWithWriter(testLogConfig("console", "info"), buf)
	logger.Info("plain")
	if strings.Contains(buf.String(), "\x1b[") {
		t.Fatalf("expected NO_COLOR to disable ANSI color codes, got %q", buf.String())
	}
	if !strings.Contains(buf.String(), "INFO choysum: plain") {
		t.Fatalf("expected plain console output, got %q", buf.String())
	}
}

func TestConsoleHandlerTraceErrorRendersMultiLineBlock(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	stubConsoleTerminalWriter(t, false)

	buf := bytes.NewBuffer(nil)
	logger := NewLoggerWithWriter(testLogConfig("console", "error"), buf)
	tracedErr := fmt.Errorf("failed to call function: %w", quickjsErr{msg: "ChoysumError: User not found or password is incorrect", Stack: "at page.ts:10\nat helper.ts:3"})
	logger.Error("js interpreter failed", "error", tracedErr, "code", "USER_NOT_FOUND")
	out := buf.String()
	if !strings.Contains(out, "ERROR choysum [") || !strings.Contains(out, "]: js interpreter failed code=USER_NOT_FOUND") {
		t.Fatalf("expected console error header, got %q", out)
	}
	firstLine, _, _ := strings.Cut(out, "\n")
	if strings.Contains(firstLine, "error=") {
		t.Fatalf("expected traced error to move into multiline block, got %q", out)
	}
	if !strings.Contains(out, "\n  error:\n") {
		t.Fatalf("expected multiline error block, got %q", out)
	}
	if !strings.Contains(out, "\n    0: failed to call function\n") {
		t.Fatalf("expected unwrap cause line, got %q", out)
	}
	if !strings.Contains(out, "\n    1: ChoysumError: User not found or password is incorrect\n") {
		t.Fatalf("expected business error line, got %q", out)
	}
	if !strings.Contains(out, "page.ts:10") || !strings.Contains(out, "helper.ts:3") {
		t.Fatalf("expected quickjs frames in multiline block, got %q", out)
	}
	if strings.Count(out, "\n") < 4 {
		t.Fatalf("expected traced error output to span multiple lines, got %q", out)
	}
}

func TestConsoleHandlerTraceErrorShortensWrappedAbsoluteQuickJSFrame(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	stubConsoleTerminalWriter(t, false)

	buf := bytes.NewBuffer(nil)
	logger := NewLoggerWithWriter(testLogConfig("console", "error"), buf)
	tracedErr := fmt.Errorf(
		"failed to call function: %w",
		quickjsErr{msg: "ChoysumError: User not found or password is incorrect", Stack: "at Login (/tmp/choysum/dist/bundles/index.js:62710:15)"},
	)
	logger.Error("js interpreter failed", "error", tracedErr, "code", "USER_NOT_FOUND")
	out := buf.String()
	if !strings.Contains(out, "Login (dist/bundles/index.js:62710:15)") {
		t.Fatalf("expected wrapped quickjs frame path to be shortened, got %q", out)
	}
	if strings.Contains(out, "/tmp/choysum/dist/bundles/index.js:62710:15") {
		t.Fatalf("expected wrapped quickjs frame to avoid absolute path, got %q", out)
	}
}

func TestConsoleHandlerPlainErrorStaysSingleLine(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	stubConsoleTerminalWriter(t, false)

	buf := bytes.NewBuffer(nil)
	logger := NewLoggerWithWriter(testLogConfig("console", "error"), buf)
	logger.Error("plain failure", "error", errors.New("bad"))
	out := buf.String()
	if !strings.Contains(out, "error=bad") && !strings.Contains(out, "error=\"bad\"") {
		t.Fatalf("expected plain error to stay inline, got %q", out)
	}
	if strings.Contains(out, "\n  error:\n") {
		t.Fatalf("expected plain error to avoid multiline trace block, got %q", out)
	}
}

func TestNewProgressLine_NilWriterReturnsNil(t *testing.T) {
	if line := NewProgressLine(nil); line != nil {
		t.Fatal("expected nil ProgressLine for nil writer")
	}
}

func TestProgressLine_NilReceiverMethodsAreNoops(t *testing.T) {
	var line *ProgressLine
	line.Update(0, "test") // should not panic
	line.Clear()           // should not panic
	line.Done("✓", "done") // should not panic
}

func TestProgressLine_NonTTYUpdateIsNoop(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	stubConsoleTerminalWriter(t, false)

	buf := bytes.NewBuffer(nil)
	line := NewProgressLine(buf)
	if line == nil {
		t.Fatal("expected non-nil line")
	}
	line.Update(0, "should not appear")
	if out := buf.String(); out != "" {
		t.Fatalf("expected empty output for non-TTY Update, got %q", out)
	}
}

func TestProgressLine_NonTTYClearIsNoop(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	stubConsoleTerminalWriter(t, false)

	buf := bytes.NewBuffer(nil)
	line := NewProgressLine(buf)
	line.Clear()
	if out := buf.String(); out != "" {
		t.Fatalf("expected empty output for non-TTY Clear, got %q", out)
	}
}

func TestProgressLine_NonTTYDoneUsesPlainPrintln(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	stubConsoleTerminalWriter(t, false)

	buf := bytes.NewBuffer(nil)
	line := NewProgressLine(buf)
	line.Done("✓", "installation complete")
	out := buf.String()
	if !strings.Contains(out, "installation complete") {
		t.Fatalf("expected plain Done output, got %q", out)
	}
	if strings.Contains(out, "\r\x1b[K") {
		t.Fatalf("expected non-TTY Done to avoid escape sequences, got %q", out)
	}
}

func TestProgressTicker_SetMessageOnNilTickerIsNoop(t *testing.T) {
	var ticker *ProgressTicker
	ticker.SetMessage("test") // should not panic
}

func TestProgressTicker_ClearOnNilTickerIsNoop(t *testing.T) {
	var ticker *ProgressTicker
	ticker.Clear() // should not panic
}

func TestProgressTicker_StopOnNilTickerIsNoop(t *testing.T) {
	var ticker *ProgressTicker
	ticker.Stop() // should not panic
}

func TestNewProgressTicker_NilLineSkipsBackgroundGoroutine(t *testing.T) {
	ticker := NewProgressTicker(nil, ProgressTickerOptions{
		Interval: 10 * time.Millisecond,
		OnTick:   func(now time.Time, message string) {},
	})
	// Nil line means no background goroutine: Stop/Clear must be safe no-ops.
	ticker.Stop()
	ticker.Clear()
}

func TestProgressTicker_StopStopsBackgroundRedraws(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	stubConsoleTerminalWriter(t, true)

	buf := bytes.NewBuffer(nil)
	line := NewProgressLine(buf)
	ticker := NewProgressTicker(line, ProgressTickerOptions{Interval: 10 * time.Millisecond})
	ticker.SetMessage("running")
	time.Sleep(50 * time.Millisecond)
	ticker.Stop()
	beforeStop := strings.Count(buf.String(), "\r\x1b[K")
	time.Sleep(50 * time.Millisecond)
	afterStop := strings.Count(buf.String(), "\r\x1b[K")
	if afterStop != beforeStop {
		t.Fatalf("expected no further redraws after Stop(), before=%d after=%d", beforeStop, afterStop)
	}
}

func TestProgressTicker_ClearAfterStopIsIdempotent(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	stubConsoleTerminalWriter(t, true)

	buf := bytes.NewBuffer(nil)
	line := NewProgressLine(buf)
	ticker := NewProgressTicker(line, ProgressTickerOptions{Interval: 10 * time.Millisecond})
	ticker.SetMessage("running")
	ticker.Stop()
	ticker.Clear() // should not panic or redraw after stop
}

func TestProgressTicker_ClearSuppressesRedrawUntilNextMessage(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	stubConsoleTerminalWriter(t, true)

	buf := &concurrentBuffer{}
	line := NewProgressLine(buf)
	ticker := NewProgressTicker(line, ProgressTickerOptions{Interval: 10 * time.Millisecond})
	defer ticker.Stop()

	ticker.SetMessage("running")
	time.Sleep(40 * time.Millisecond)
	ticker.Clear()
	afterClear := strings.Count(buf.String(), "\r\x1b[K")
	time.Sleep(40 * time.Millisecond)
	afterWait := strings.Count(buf.String(), "\r\x1b[K")
	if afterWait != afterClear {
		t.Fatalf("expected no redraw after Clear() without a new message, before=%d after=%d", afterClear, afterWait)
	}

	ticker.SetMessage("resume")
	afterResume := strings.Count(buf.String(), "\r\x1b[K")
	if afterResume <= afterWait {
		t.Fatalf("expected redraw after SetMessage(), before=%d after=%d", afterWait, afterResume)
	}
}

// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package logger

import (
	"io"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/golang-cz/devslog"
)

const (
	identitySystem = "choysum"
)

func withIdentity(logger *slog.Logger) *slog.Logger {
	if logger == nil {
		return nil
	}
	return logger.With(
		"system", identitySystem,
	)
}

func replaceAttr(_ []string, a slog.Attr) slog.Attr {
	switch a.Value.Kind() {
	case slog.KindAny:
		switch v := a.Value.Any().(type) {
		case error:
			a.Value = fmtErr(v)
		}
	}
	return a
}

func replaceAttrForDevslog(_ []string, a slog.Attr) slog.Attr {
	switch a.Value.Kind() {
	case slog.KindAny:
		switch v := a.Value.Any().(type) {
		case error:
			a.Value = slog.AnyValue(fmtErrForDevslog(v))
		}
	}
	return a
}

func fmtErr(err error) slog.Value {
	return slogValueFromErrorTrace(collectErrorTrace(err))
}

func fmtErrForDevslog(err error) error {
	if err == nil {
		return nil
	}
	trace := collectErrorTrace(err)
	if !trace.HasFrames() {
		return err
	}

	var wrapped error
	for idx := len(trace.Trace) - 1; idx >= 0; idx-- {
		entry := trace.Trace[idx]
		wrapped = devslogTraceError{
			msg:   formatDevslogTraceEntry(entry),
			cause: wrapped,
		}
	}

	return wrapped
}

type devslogTraceError struct {
	msg   string
	cause error
}

func (e devslogTraceError) Error() string {
	if e.cause == nil {
		return e.msg
	}
	return e.msg + ": " + e.cause.Error()
}

func (e devslogTraceError) Unwrap() error {
	return e.cause
}

func formatDevslogTraceEntry(entry errorTraceEntry) string {
	if len(entry.Frames) == 0 {
		return entry.Msg
	}

	var builder strings.Builder
	builder.WriteString(entry.Msg)
	for index, frame := range entry.Frames {
		builder.WriteString("\n       ")
		builder.WriteString(strconv.Itoa(index))
		builder.WriteString(": ")
		builder.WriteString(formatDevslogTraceFrame(frame))
	}
	return builder.String()
}

func formatDevslogTraceFrame(frame errorTraceFrame) string {
	location := strings.TrimSpace(frame.File)
	function := strings.TrimSpace(frame.Function)
	if function == "" {
		return location
	}
	if location == "" {
		return function
	}
	return location + ", " + function
}

func cloneLogConfig(cfg *config.LogConfig) *config.LogConfig {
	if cfg == nil {
		return config.NewDefaultLogConfig()
	}
	cloned := *cfg
	return &cloned
}

func newHandler(logCfg *config.LogConfig, w io.Writer) slog.Handler {
	resolvedLogCfg := cloneLogConfig(logCfg)
	resolvedFormat := strings.ToLower(strings.TrimSpace(resolvedLogCfg.Format))
	slogLevel := slog.LevelInfo
	switch strings.ToLower(resolvedLogCfg.Level) {
	case "debug":
		slogLevel = slog.LevelDebug
	case "info":
		slogLevel = slog.LevelInfo
	case "warn":
		slogLevel = slog.LevelWarn
	case "error":
		slogLevel = slog.LevelError
	}

	var slogHandler slog.Handler
	switch resolvedFormat {
	case "devslog":
		slogHandler = devslog.NewHandler(w, &devslog.Options{
			HandlerOptions: &slog.HandlerOptions{
				AddSource:   true,
				Level:       slogLevel,
				ReplaceAttr: replaceAttrForDevslog,
			},
			TimeFormat:         "[2006-01-02 15:04:05.000]",
			SortKeys:           true,
			NewLineAfterLog:    false,
			MaxErrorStackTrace: 999,
		})
	case "json":
		slogHandler = slog.NewJSONHandler(w, &slog.HandlerOptions{
			AddSource:   true,
			Level:       slogLevel,
			ReplaceAttr: replaceAttr,
		})
	case "console":
		slogHandler = newConsoleHandler(w, slogLevel)
	case "text":
		slogHandler = slog.NewTextHandler(w, &slog.HandlerOptions{
			AddSource:   true,
			Level:       slogLevel,
			ReplaceAttr: replaceAttr,
		})
	case "":
		if consoleTerminalWriter(w) {
			slogHandler = newConsoleHandler(w, slogLevel)
			break
		}
		fallthrough
	default:
		slogHandler = slog.NewTextHandler(w, &slog.HandlerOptions{
			AddSource:   true,
			Level:       slogLevel,
			ReplaceAttr: replaceAttr,
		})
	}

	return slogHandler
}

func NewLogger(logCfg *config.LogConfig) *slog.Logger {
	return withIdentity(slog.New(newHandler(logCfg, wrapProgressAwareWriter(os.Stdout))))
}

func NewLoggerWithWriter(logCfg *config.LogConfig, w io.Writer) *slog.Logger {
	if w == nil {
		w = os.Stdout
	}
	return withIdentity(slog.New(newHandler(logCfg, wrapProgressAwareWriter(w))))
}

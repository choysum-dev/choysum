package logger

import (
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"runtime"
	"strings"
)

type errorTraceFrame struct {
	File     string
	Function string
}

type errorTraceEntry struct {
	Msg    string
	Frames []errorTraceFrame
}

type errorTrace struct {
	Msg   string
	Trace []errorTraceEntry
}

func collectErrorTrace(err error) errorTrace {
	if err == nil {
		return errorTrace{}
	}

	trace := errorTrace{Msg: err.Error()}
	for current := err; current != nil; current = errors.Unwrap(current) {
		trace.Trace = append(trace.Trace, errorTraceEntry{
			Msg:    errorTraceMessage(current),
			Frames: collectErrorFrames(current),
		})
	}
	return trace
}

func (t errorTrace) HasFrames() bool {
	for _, entry := range t.Trace {
		if len(entry.Frames) > 0 {
			return true
		}
	}
	return false
}

func slogValueFromErrorTrace(trace errorTrace) slog.Value {
	groupValues := []slog.Attr{slog.String("msg", trace.Msg)}
	frames := make([]map[string]interface{}, 0, len(trace.Trace))
	for _, entry := range trace.Trace {
		entryFrames := make([]map[string]string, 0, len(entry.Frames))
		for _, frame := range entry.Frames {
			entryFrames = append(entryFrames, map[string]string{"file": frame.File, "function": frame.Function})
		}
		frames = append(frames, map[string]interface{}{
			"msg":    entry.Msg,
			"frames": entryFrames,
		})
	}
	groupValues = append(groupValues, slog.Any("trace", frames))
	return slog.GroupValue(groupValues...)
}

func errorTraceMessage(err error) string {
	errMsg := err.Error()
	ue := errors.Unwrap(err)
	if ue != nil {
		errMsg, _ = strings.CutSuffix(errMsg, ue.Error())
		errMsg, _ = strings.CutSuffix(errMsg, ": ")
	}
	if errMsg == "" {
		errMsg = fmt.Sprintf("[%T]", err)
	}
	return errMsg
}

func collectErrorFrames(err error) []errorTraceFrame {
	frames := make([]errorTraceFrame, 0)
	for _, fileLine := range getFileLineFromPC(extractPCFromError(err)) {
		frames = append(frames, parsePCTraceFrame(fileLine))
	}
	for _, fileLine := range extractFromQuickjsError(err) {
		frames = append(frames, errorTraceFrame{File: fileLine})
	}
	return frames
}

func parsePCTraceFrame(fileLine string) errorTraceFrame {
	file, function, found := strings.Cut(fileLine, ",")
	if !found {
		return errorTraceFrame{File: fileLine}
	}
	return errorTraceFrame{File: file, Function: function}
}

// https://github.com/golang-cz/devslog/blob/4299e8a1981f37f5ad9b9d75f04a575cfcfd81f0/stacktrace.go#L30
func extractPCFromError(err error) (pc []uintptr) {
	v := reflect.ValueOf(err)
	if v.Kind() == reflect.Ptr && !v.IsNil() {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return nil
	}

	v = v.FieldByName("frame")
	if v.Kind() != reflect.Struct {
		return nil
	}

	// https://cs.opensource.google/go/x/exp/+/92128663:errors/frame.go;l=12
	//
	// type Frame struct {
	// 	 frames [3]uintptr
	// }
	v = v.FieldByName("frames")
	if v.Kind() != reflect.Array {
		return nil
	}

	// Skip first frame pointing at fmt.Errorf() or errors.New().
	skip := 1
	for i := skip; i < min(v.Len()); i++ {
		index := v.Index(i)
		if !index.CanUint() {
			return nil
		}

		pc = append(pc, uintptr(index.Uint()))
	}

	return pc
}

func extractFromQuickjsError(err error) []string {
	v := reflect.ValueOf(err)
	if v.Kind() == reflect.Ptr && !v.IsNil() {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return nil
	}

	// stack is string
	v = v.FieldByName("Stack")
	if v.Kind() != reflect.String {
		return nil
	}

	var fileLines []string
	for _, line := range strings.Split(v.String(), "\n") {
		if strings.Contains(line, "at ") {
			line = strings.TrimPrefix(strings.Trim(line, " "), "at ")
			fileLines = append(fileLines, line)
		}
	}

	return fileLines

}

// https://github.com/golang-cz/devslog/blob/4299e8a1981f37f5ad9b9d75f04a575cfcfd81f0/stacktrace.go#L9
func getFileLineFromPC(pcs []uintptr) (fileLines []string) {
	if len(pcs) == 0 {
		return nil
	}
	frames := runtime.CallersFrames(pcs[:])
	for {
		fr, more := frames.Next()
		fileLines = append(fileLines, fmt.Sprintf("%v:%v, %s", fr.File, fr.Line, fr.Function))
		if !more {
			break
		}
	}

	return fileLines
}

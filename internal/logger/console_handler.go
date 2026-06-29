package logger

import (
	"context"
	"encoding"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/term"
)

const (
	consoleTimeFormat  = "2006-01-02 15:04:05,000"
	consoleColorBlue   = "\x1b[34m"
	consoleColorGreen  = "\x1b[32m"
	consoleColorReset  = "\x1b[0m"
	consoleColorRed    = "\x1b[31m"
	consoleColorYellow = "\x1b[33m"
)

var (
	consoleTerminalWriter = func(w io.Writer) bool {
		w = unwrapTerminalWriter(w)
		file, ok := w.(*os.File)
		if !ok {
			return false
		}
		return term.IsTerminal(int(file.Fd()))
	}
	consoleWorkingDirectory = os.Getwd
)

func unwrapTerminalWriter(w io.Writer) io.Writer {
	for w != nil {
		unwrapper, ok := w.(interface{ Unwrap() io.Writer })
		if !ok {
			return w
		}
		next := unwrapper.Unwrap()
		if next == nil || next == w {
			return w
		}
		w = next
	}
	return w
}

type consoleGroupOrAttrs struct {
	group string
	attrs []slog.Attr
}

type consoleAttr struct {
	key   string
	value slog.Value
}

type consoleTraceBlock struct {
	key   string
	trace errorTrace
}

type consoleHandler struct {
	writer       io.Writer
	level        slog.Leveler
	goas         []consoleGroupOrAttrs
	mu           *sync.Mutex
	pid          int
	workingDir   string
	colorEnabled bool
}

func newConsoleHandler(w io.Writer, level slog.Leveler) slog.Handler {
	if w == nil {
		w = os.Stdout
	}
	if level == nil {
		level = slog.LevelInfo
	}
	workingDir, err := consoleWorkingDirectory()
	if err != nil {
		workingDir = ""
	}
	return &consoleHandler{
		writer:       w,
		level:        level,
		mu:           &sync.Mutex{},
		pid:          os.Getpid(),
		workingDir:   workingDir,
		colorEnabled: consoleTerminalWriter(w) && os.Getenv("NO_COLOR") == "",
	}
}

func (h *consoleHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level.Level()
}

func (h *consoleHandler) Handle(_ context.Context, record slog.Record) error {
	attrs := h.collectAttrs(&record)
	system, attrs := extractConsoleSystem(attrs)
	source := ""
	if shouldShowConsoleSource(record.Level) {
		source = h.consoleSource(record.PC)
	}

	var line strings.Builder
	line.Grow(256)
	line.WriteString(record.Time.Format(consoleTimeFormat))
	line.WriteByte(' ')
	line.WriteString(strconv.Itoa(h.pid))
	line.WriteByte(' ')
	line.WriteString(h.consoleLevel(record.Level))
	if system != "" {
		line.WriteByte(' ')
		line.WriteString(system)
	}
	if source != "" {
		line.WriteByte(' ')
		line.WriteByte('[')
		line.WriteString(source)
		line.WriteByte(']')
	}
	line.WriteString(": ")
	line.WriteString(record.Message)
	traceBlocks := make([]consoleTraceBlock, 0)
	for _, attr := range attrs {
		if block, ok := consoleTraceBlockFromAttr(attr); ok {
			traceBlocks = append(traceBlocks, block)
			continue
		}
		line.WriteByte(' ')
		line.WriteString(attr.key)
		line.WriteByte('=')
		line.WriteString(formatConsoleValue(attr.value))
	}
	line.WriteByte('\n')
	for _, block := range traceBlocks {
		line.WriteString(formatConsoleTraceBlock(block, h.workingDir))
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := io.WriteString(h.writer, line.String())
	return err
}

func (h *consoleHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	return h.withGroupOrAttrs(consoleGroupOrAttrs{attrs: attrs})
}

func (h *consoleHandler) WithGroup(group string) slog.Handler {
	if strings.TrimSpace(group) == "" {
		return h
	}
	return h.withGroupOrAttrs(consoleGroupOrAttrs{group: group})
}

func (h *consoleHandler) withGroupOrAttrs(goa consoleGroupOrAttrs) *consoleHandler {
	cloned := *h
	cloned.goas = append(append([]consoleGroupOrAttrs(nil), h.goas...), goa)
	return &cloned
}

func (h *consoleHandler) collectAttrs(record *slog.Record) []consoleAttr {
	var attrs []consoleAttr
	prefix := []string{}
	for _, goa := range h.goas {
		if goa.group != "" {
			prefix = append(prefix, goa.group)
			continue
		}
		attrs = appendConsoleAttrs(attrs, prefix, goa.attrs)
	}
	if record == nil {
		return attrs
	}
	record.Attrs(func(attr slog.Attr) bool {
		attrs = appendConsoleAttr(attrs, prefix, attr)
		return true
	})
	return attrs
}

func appendConsoleAttrs(attrs []consoleAttr, prefix []string, in []slog.Attr) []consoleAttr {
	for _, attr := range in {
		attrs = appendConsoleAttr(attrs, prefix, attr)
	}
	return attrs
}

func appendConsoleAttr(attrs []consoleAttr, prefix []string, attr slog.Attr) []consoleAttr {
	attr.Value = attr.Value.Resolve()
	if attr.Value.Kind() == slog.KindGroup {
		nextPrefix := prefix
		if strings.TrimSpace(attr.Key) != "" {
			nextPrefix = append(copyConsolePrefix(prefix), attr.Key)
		}
		for _, groupAttr := range attr.Value.Group() {
			attrs = appendConsoleAttr(attrs, nextPrefix, groupAttr)
		}
		return attrs
	}
	if strings.TrimSpace(attr.Key) == "" {
		return attrs
	}
	key := attr.Key
	if len(prefix) > 0 {
		parts := append(copyConsolePrefix(prefix), key)
		key = strings.Join(parts, ".")
	}
	return append(attrs, consoleAttr{key: key, value: attr.Value})
}

func copyConsolePrefix(prefix []string) []string {
	if len(prefix) == 0 {
		return nil
	}
	cloned := make([]string, len(prefix))
	copy(cloned, prefix)
	return cloned
}

func extractConsoleSystem(attrs []consoleAttr) (string, []consoleAttr) {
	filtered := make([]consoleAttr, 0, len(attrs))
	system := ""
	for _, attr := range attrs {
		if attr.key == "system" {
			system = consoleValuePlainText(attr.value)
			continue
		}
		filtered = append(filtered, attr)
	}
	return system, filtered
}

func shouldShowConsoleSource(level slog.Level) bool {
	return level < slog.LevelInfo || level >= slog.LevelWarn
}

func (h *consoleHandler) consoleSource(pc uintptr) string {
	if pc == 0 {
		return ""
	}
	frame, _ := runtime.CallersFrames([]uintptr{pc}).Next()
	if frame.File == "" || frame.Line <= 0 {
		return ""
	}
	path := shortenConsoleSourcePath(frame.File, h.workingDir)
	if path == "" {
		return ""
	}
	return path + ":" + strconv.Itoa(frame.Line)
}

func shortenConsoleSourcePath(filePath, workingDir string) string {
	cleaned := filepath.Clean(filePath)
	if cleaned == "." || cleaned == "" {
		return ""
	}
	if workingDir != "" {
		rel, err := filepath.Rel(workingDir, cleaned)
		if err == nil {
			rel = filepath.ToSlash(rel)
			if rel != "." && rel != ".." && !strings.HasPrefix(rel, "../") {
				return rel
			}
		}
	}
	parts := strings.Split(filepath.ToSlash(cleaned), "/")
	if len(parts) > 3 {
		parts = parts[len(parts)-3:]
	}
	return strings.Join(parts, "/")
}

func (h *consoleHandler) consoleLevel(level slog.Level) string {
	text := strings.ToUpper(level.String())
	if !h.colorEnabled {
		return text
	}
	return consoleLevelColor(level) + text + consoleColorReset
}

func consoleLevelColor(level slog.Level) string {
	switch {
	case level < slog.LevelInfo:
		return consoleColorBlue
	case level < slog.LevelWarn:
		return consoleColorGreen
	case level < slog.LevelError:
		return consoleColorYellow
	default:
		return consoleColorRed
	}
}

func formatConsoleValue(value slog.Value) string {
	value = value.Resolve()
	switch value.Kind() {
	case slog.KindString:
		return quoteConsoleString(value.String())
	case slog.KindInt64:
		return strconv.FormatInt(value.Int64(), 10)
	case slog.KindUint64:
		return strconv.FormatUint(value.Uint64(), 10)
	case slog.KindFloat64:
		return strconv.FormatFloat(value.Float64(), 'f', -1, 64)
	case slog.KindBool:
		return strconv.FormatBool(value.Bool())
	case slog.KindDuration:
		return value.Duration().String()
	case slog.KindTime:
		return value.Time().Format(time.RFC3339Nano)
	case slog.KindAny:
		return formatConsoleAny(value.Any())
	case slog.KindGroup:
		return "{}"
	default:
		return quoteConsoleString(value.String())
	}
}

func consoleValuePlainText(value slog.Value) string {
	formatted := formatConsoleValue(value)
	if unquoted, err := strconv.Unquote(formatted); err == nil {
		return unquoted
	}
	return formatted
}

func formatConsoleAny(v any) string {
	if v == nil {
		return "null"
	}
	if err, ok := v.(error); ok {
		return strconv.Quote(err.Error())
	}
	if textMarshaler, ok := v.(encoding.TextMarshaler); ok {
		text, err := textMarshaler.MarshalText()
		if err == nil {
			return quoteConsoleString(string(text))
		}
	}
	if stringer, ok := v.(fmt.Stringer); ok {
		return quoteConsoleString(stringer.String())
	}
	return formatConsoleReflect(reflect.ValueOf(v))
}

func consoleTraceBlockFromAttr(attr consoleAttr) (consoleTraceBlock, bool) {
	value := attr.value.Resolve()
	if value.Kind() != slog.KindAny {
		return consoleTraceBlock{}, false
	}
	err, ok := value.Any().(error)
	if !ok || err == nil {
		return consoleTraceBlock{}, false
	}
	trace := collectErrorTrace(err)
	if !trace.HasFrames() {
		return consoleTraceBlock{}, false
	}
	return consoleTraceBlock{key: attr.key, trace: trace}, true
}

func formatConsoleTraceBlock(block consoleTraceBlock, workingDir string) string {
	var builder strings.Builder
	builder.WriteString("  ")
	builder.WriteString(block.key)
	builder.WriteString(":\n")
	for index, entry := range block.trace.Trace {
		builder.WriteString("    ")
		builder.WriteString(strconv.Itoa(index))
		builder.WriteString(": ")
		builder.WriteString(entry.Msg)
		builder.WriteByte('\n')
		for frameIndex, frame := range entry.Frames {
			builder.WriteString("       ")
			builder.WriteString(strconv.Itoa(frameIndex))
			builder.WriteString(": ")
			builder.WriteString(formatConsoleTraceFrame(frame, workingDir))
			builder.WriteByte('\n')
		}
	}
	return builder.String()
}

func formatConsoleTraceFrame(frame errorTraceFrame, workingDir string) string {
	location := shortenConsoleTraceLocation(frame.File, workingDir)
	function := strings.TrimSpace(frame.Function)
	if location == "" {
		return function
	}
	if function == "" {
		return location
	}
	return location + ", " + function
}

func shortenConsoleTraceLocation(location, workingDir string) string {
	if shortened, ok := shortenWrappedConsoleTraceLocation(location, workingDir); ok {
		return shortened
	}
	if !strings.HasPrefix(location, "/") {
		return location
	}
	return shortenAbsoluteConsoleTraceLocation(location, workingDir)
}

func shortenWrappedConsoleTraceLocation(location, workingDir string) (string, bool) {
	if !strings.HasSuffix(location, ")") {
		return "", false
	}
	open := strings.LastIndex(location, " (")
	if open < 0 {
		return "", false
	}
	function := strings.TrimSpace(location[:open])
	inner := strings.TrimSpace(location[open+2 : len(location)-1])
	if !strings.HasPrefix(inner, "/") {
		return "", false
	}
	shortened := shortenAbsoluteConsoleTraceLocation(inner, workingDir)
	if shortened == "" {
		shortened = inner
	}
	if function == "" {
		return shortened, true
	}
	return function + " (" + shortened + ")", true
}

func shortenAbsoluteConsoleTraceLocation(location, workingDir string) string {
	pathPart := location
	suffix := ""
	if index := strings.Index(location, ":"); index >= 0 {
		pathPart = location[:index]
		suffix = location[index:]
	}
	shortened := shortenConsoleSourcePath(pathPart, workingDir)
	if shortened == "" {
		shortened = pathPart
	}
	return shortened + suffix
}

func formatConsoleReflect(value reflect.Value) string {
	value = unwrapConsoleValue(value)
	if !value.IsValid() {
		return "null"
	}
	switch value.Kind() {
	case reflect.String:
		return quoteConsoleString(value.String())
	case reflect.Bool:
		return strconv.FormatBool(value.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(value.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return strconv.FormatUint(value.Uint(), 10)
	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(value.Float(), 'f', -1, 64)
	case reflect.Slice, reflect.Array:
		return formatConsoleSlice(value)
	case reflect.Map:
		return formatConsoleMap(value)
	default:
		return quoteConsoleString(fmt.Sprint(value.Interface()))
	}
}

func unwrapConsoleValue(value reflect.Value) reflect.Value {
	for value.IsValid() {
		switch value.Kind() {
		case reflect.Interface, reflect.Pointer:
			if value.IsNil() {
				return reflect.Value{}
			}
			value = value.Elem()
		default:
			return value
		}
	}
	return value
}

func formatConsoleSlice(value reflect.Value) string {
	value = unwrapConsoleValue(value)
	if !value.IsValid() {
		return "[]"
	}
	parts := make([]string, 0, value.Len())
	for i := 0; i < value.Len(); i++ {
		parts = append(parts, formatConsoleReflect(value.Index(i)))
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func formatConsoleMap(value reflect.Value) string {
	value = unwrapConsoleValue(value)
	if !value.IsValid() {
		return "{}"
	}
	keys := value.MapKeys()
	sort.Slice(keys, func(i, j int) bool {
		return fmt.Sprint(keys[i].Interface()) < fmt.Sprint(keys[j].Interface())
	})
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, formatConsoleReflect(key)+"="+formatConsoleReflect(value.MapIndex(key)))
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

func quoteConsoleString(value string) string {
	if !needsConsoleQuoting(value) {
		return value
	}
	return strconv.Quote(value)
}

func needsConsoleQuoting(value string) bool {
	if value == "" {
		return true
	}
	return strings.ContainsAny(value, " \t\r\n\"\\")
}

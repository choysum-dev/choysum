// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package backendtsparser

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/buke/typescript-go-internal/v7/pkg/ast"
	"github.com/buke/typescript-go-internal/v7/pkg/bundled"
	"github.com/buke/typescript-go-internal/v7/pkg/checker"
	"github.com/buke/typescript-go-internal/v7/pkg/compiler"
	"github.com/buke/typescript-go-internal/v7/pkg/core"
	"github.com/buke/typescript-go-internal/v7/pkg/tsoptions"
	"github.com/buke/typescript-go-internal/v7/pkg/vfs"
	"github.com/buke/typescript-go-internal/v7/pkg/vfs/osvfs"
	"github.com/buke/typescript-go-internal/v7/pkg/vfs/wrapvfs"
	"github.com/choysum-dev/choysum/internal/protobuf/objectmessages"
	"golang.org/x/sync/singleflight"
)

const (
	protoTypeString     = "string"
	protoTypeDouble     = "double"
	protoTypeBool       = "bool"
	protoTypeEmpty      = "google.protobuf.Empty"
	protoTypeInt64      = "int64"
	protoTypeTimestamp  = "google.protobuf.Timestamp"
	protoTypeValue      = "google.protobuf.Value"
	protoRepeatedPrefix = "repeated "

	envDisableSemanticProto = "CHOYSUM_DISABLE_SEMANTIC_PROTO"

	// Cap retained Programs so large module trees do not pin unbounded checker state.
	defaultSemanticProgramCacheLimit = 16
)

// Test overrides for coverage of rare failure paths.
var (
	semanticLibsEmbedded      = bundled.Embedded
	buildSemanticProgramImpl  = buildSemanticProgram
	semanticProgramSourceFile = func(program *compiler.Program, path string) *ast.SourceFile {
		return program.GetSourceFile(path)
	}
	semanticProgramCacheLimit = defaultSemanticProgramCacheLimit
	// Invoked after an outer cache miss unlock, before singleflight.Do.
	semanticAfterCacheMissUnlock = func() {}
	// Invoked after a successful Program build, before the final cache publish.
	semanticAfterProgramBuild = func() {}
)

// semanticTypeResolver reduces service method parameter/return types to protobuf
// scalars via typescript-go-internal checker. Failures fall back to text mapping.
type semanticTypeResolver struct {
	mu         sync.Mutex
	cache      map[string]*semanticFileState
	cacheOrder []string // oldest → newest path keys for LRU eviction
	builds     singleflight.Group
	disabled   bool
	initOnce   sync.Once
	logger     *slog.Logger
}

type semanticFileState struct {
	content string
	program *compiler.Program
	file    *ast.SourceFile
}

func newSemanticTypeResolver(logger *slog.Logger) *semanticTypeResolver {
	return &semanticTypeResolver{
		cache:  make(map[string]*semanticFileState),
		logger: logger,
	}
}

func (r *semanticTypeResolver) ensureEnabled() bool {
	r.initOnce.Do(func() {
		if os.Getenv(envDisableSemanticProto) == "1" {
			r.disabled = true
			r.logWarn("semantic protobuf mapping disabled via " + envDisableSemanticProto)
			return
		}
		if !semanticLibsEmbedded {
			r.disabled = true
			r.logWarn("semantic protobuf mapping disabled: typescript-go bundled libs are not embedded")
		}
	})
	return !r.disabled
}

func (r *semanticTypeResolver) logWarn(msg string, args ...any) {
	if r == nil || r.logger == nil {
		return
	}
	r.logger.Warn(msg, args...)
}

func (r *semanticTypeResolver) logDebug(msg string, args ...any) {
	if r == nil || r.logger == nil {
		return
	}
	r.logger.Debug(msg, args...)
}

// resolveProtoType prefers semantic reduction, then text mapping.
func (r *semanticTypeResolver) resolveProtoType(path, content, className, methodName, paramName string, isReturn bool, tsAnnotation string) string {
	if mapped, ok := r.trySemantic(path, content, className, methodName, paramName, isReturn); ok {
		r.logDebug("semantic protobuf mapping hit",
			"path", path, "method", methodName, "param", paramName, "is_return", isReturn, "protobuf_type", mapped)
		return mapped
	}
	fallback := getProtoTypeFromTsType(tsAnnotation)
	r.logDebug("semantic protobuf mapping fallback",
		"path", path, "method", methodName, "param", paramName, "is_return", isReturn,
		"ts_annotation", tsAnnotation, "protobuf_type", fallback)
	return fallback
}

func (r *semanticTypeResolver) trySemantic(path, content, className, methodName, paramName string, isReturn bool) (string, bool) {
	if r == nil || !r.ensureEnabled() {
		return "", false
	}
	path = normalizeSemanticPath(path)
	if path == "" || content == "" || methodName == "" {
		return "", false
	}

	state, err := r.ensureFile(path, content)
	if err != nil {
		// Keep semantic mapping enabled for other files; only this lookup falls back.
		r.logWarn("semantic protobuf mapping failed for file", "path", path, "err", err)
		return "", false
	}

	method := findClassMethodNode(state.file, className, methodName)
	if method == nil {
		return "", false
	}
	methodDecl := method.AsMethodDeclaration()
	var typeNode *ast.Node
	if isReturn {
		typeNode = methodDecl.Type
	} else {
		var nodes []*ast.Node
		if methodDecl.Parameters != nil {
			nodes = methodDecl.Parameters.Nodes
		}
		typeNode = findParamTypeNode(nodes, paramName)
	}
	if typeNode == nil {
		return "", false
	}

	// Exclusive access: cached Programs are shared across concurrent lookups.
	c, done := state.program.GetTypeCheckerForFileExclusive(context.Background(), state.file)
	defer done()
	return mapCheckerTypeToProto(c, c.GetTypeFromTypeNode(typeNode), isReturn)
}

// ensureFile returns a cached Program for path/content. Compilation runs without
// holding r.mu; concurrent misses for the same path+content share one build.
func (r *semanticTypeResolver) ensureFile(path, content string) (*semanticFileState, error) {
	r.mu.Lock()
	if cached, ok := r.cache[path]; ok && cached != nil && cached.content == content {
		r.touchCacheLocked(path)
		r.mu.Unlock()
		r.logDebug("semantic program cache hit", "path", path)
		return cached, nil
	}
	r.mu.Unlock()
	semanticAfterCacheMissUnlock()

	v, err, _ := r.builds.Do(path+"\x00"+content, func() (any, error) {
		r.mu.Lock()
		if cached, ok := r.cache[path]; ok && cached != nil && cached.content == content {
			r.touchCacheLocked(path)
			r.mu.Unlock()
			return cached, nil
		}
		r.mu.Unlock()

		start := time.Now()
		program, file, buildErr := buildSemanticProgramImpl(path, content)
		elapsed := time.Since(start)
		if buildErr != nil {
			r.logDebug("semantic program build failed", "path", path, "elapsed_ms", elapsed.Milliseconds(), "err", buildErr)
			return nil, buildErr
		}
		r.logDebug("semantic program build ok", "path", path, "elapsed_ms", elapsed.Milliseconds())
		state := &semanticFileState{
			content: content,
			program: program,
			file:    file,
		}

		semanticAfterProgramBuild()

		r.mu.Lock()
		defer r.mu.Unlock()
		if cached, ok := r.cache[path]; ok && cached != nil && cached.content == content {
			r.touchCacheLocked(path)
			return cached, nil
		}
		r.putCacheLocked(path, state)
		return state, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*semanticFileState), nil
}

func (r *semanticTypeResolver) touchCacheLocked(path string) {
	for i, cachedPath := range r.cacheOrder {
		if cachedPath != path {
			continue
		}
		r.cacheOrder = append(r.cacheOrder[:i], r.cacheOrder[i+1:]...)
		break
	}
	r.cacheOrder = append(r.cacheOrder, path)
}

func (r *semanticTypeResolver) putCacheLocked(path string, state *semanticFileState) {
	r.cache[path] = state
	r.touchCacheLocked(path)
	limit := semanticProgramCacheLimit
	if limit <= 0 {
		limit = defaultSemanticProgramCacheLimit
	}
	for len(r.cache) > limit && len(r.cacheOrder) > 0 {
		oldest := r.cacheOrder[0]
		if oldest == path {
			// Newly inserted entry is newest; do not drop it from cacheOrder.
			break
		}
		r.cacheOrder = r.cacheOrder[1:]
		delete(r.cache, oldest)
	}
}

func buildSemanticProgram(path, content string) (*compiler.Program, *ast.SourceFile, error) {
	fs := newSemanticOverlayFS(path, content)
	currentDir := filepath.ToSlash(filepath.Dir(path))
	if currentDir == "" || currentDir == "." {
		currentDir = "/"
	}
	host := compiler.NewCompilerHost(currentDir, fs, bundled.LibPath(), nil, nil)
	opts := &core.CompilerOptions{
		Target:                 core.ScriptTargetES2020,
		Module:                 core.ModuleKindESNext,
		ModuleResolution:       core.ModuleResolutionKindBundler,
		StrictNullChecks:       core.TSTrue,
		SkipLibCheck:           core.TSTrue,
		NoEmit:                 core.TSTrue,
		ExperimentalDecorators: core.TSTrue,
		AllowJs:                core.TSTrue,
	}
	program := compiler.NewProgram(compiler.ProgramOptions{
		Host: host,
		Config: &tsoptions.ParsedCommandLine{
			ParsedConfig: &core.ParsedOptions{
				FileNames:       []string{path},
				CompilerOptions: opts,
			},
		},
		SingleThreaded: core.TSTrue,
	})
	file := semanticProgramSourceFile(program, path)
	if file == nil {
		return nil, nil, errSemanticSourceFileMissing
	}
	return program, file, nil
}

var errSemanticSourceFileMissing = semanticInitError("source file missing from program")

type semanticInitError string

func (e semanticInitError) Error() string { return string(e) }

func newSemanticOverlayFS(path, content string) vfs.FS {
	normalized := normalizeSemanticPath(path)
	base := osvfs.FS()
	caseSensitive := base.UseCaseSensitiveFileNames()
	overlay := wrapvfs.Wrap(base, wrapvfs.Replacements{
		FileExists: func(p string) bool {
			if semanticOverlayPathMatch(normalized, p, caseSensitive) {
				return true
			}
			return base.FileExists(p)
		},
		ReadFile: func(p string) (string, bool) {
			if semanticOverlayPathMatch(normalized, p, caseSensitive) {
				return content, true
			}
			return base.ReadFile(p)
		},
	})
	return bundled.WrapFS(overlay)
}

func semanticOverlayPathMatch(normalizedOverlay, requestPath string, caseSensitive bool) bool {
	if normalizedOverlay == "" {
		return false
	}
	got := normalizeSemanticPath(requestPath)
	if caseSensitive {
		return got == normalizedOverlay
	}
	return strings.EqualFold(got, normalizedOverlay)
}

func normalizeSemanticPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	return filepath.ToSlash(filepath.Clean(path))
}

func findClassMethodNode(file *ast.SourceFile, className, methodName string) *ast.Node {
	if file == nil || methodName == "" {
		return nil
	}
	var anonymous *ast.Node
	for _, stmt := range file.Statements.Nodes {
		if stmt == nil || stmt.Kind != ast.KindClassDeclaration {
			continue
		}
		name := nodeNameText(stmt)
		method := findMethodDeclaration(stmt, methodName)
		if method == nil {
			continue
		}
		if className == "" {
			return method
		}
		switch {
		case name == className:
			return method
		case name == "" && anonymous == nil:
			// Unnamed default-export class is only a fallback when className is set.
			anonymous = method
		}
	}
	return anonymous
}

func findMethodDeclaration(class *ast.Node, methodName string) *ast.Node {
	for _, member := range class.Members() {
		if member == nil || member.Kind != ast.KindMethodDeclaration {
			continue
		}
		if nodeNameText(member) == methodName {
			return member
		}
	}
	return nil
}

func nodeNameText(node *ast.Node) string {
	if node == nil || node.Name() == nil {
		return ""
	}
	return strings.TrimSpace(node.Name().Text())
}

func findParamTypeNode(nodes []*ast.Node, paramName string) *ast.Node {
	if paramName == "" {
		return nil
	}
	for _, paramNode := range nodes {
		if paramNode == nil || paramNode.Kind != ast.KindParameter {
			continue
		}
		paramDecl := paramNode.AsParameterDeclaration()
		if paramDecl.Name() == nil {
			continue
		}
		if strings.TrimSpace(paramDecl.Name().Text()) == paramName {
			return paramDecl.Type
		}
	}
	return nil
}

func mapCheckerTypeToProto(c *checker.Checker, t *checker.Type, isReturn bool) (string, bool) {
	if c == nil || t == nil || t == c.GetErrorType() {
		return "", false
	}
	if t.Flags()&checker.TypeFlagsAnyOrUnknown != 0 {
		return "", false
	}
	if promised := c.GetPromisedTypeOfPromise(t); promised != nil {
		t = promised
	}

	// Named object messages must be recognized before union expansion (e.g. ConditionEnvelope).
	if mapped, ok := mapRegisteredObjectMessage(t); ok {
		return mapped, true
	}

	parts := []*checker.Type{t}
	if t.IsUnion() {
		parts = t.Types()
	}
	parts = stripNullishParts(parts)
	if len(parts) == 0 {
		if isReturn {
			return protoTypeEmpty, true
		}
		return "", false
	}
	if len(parts) == 1 {
		t = parts[0]
		if mapped, ok := mapRegisteredObjectMessage(t); ok {
			return mapped, true
		}
		if c.IsArrayType(t) {
			elem := c.GetElementTypeOfArrayType(t)
			mapped, ok := mapCheckerTypeToProto(c, elem, false)
			if !ok || strings.HasPrefix(mapped, protoRepeatedPrefix) {
				return "", false
			}
			switch mapped {
			case protoTypeString, protoTypeDouble, protoTypeBool, protoTypeInt64:
				return protoRepeatedPrefix + mapped, true
			default:
				return "", false
			}
		}
		if mapped, ok := mapSpecialCheckerType(c, t); ok {
			return mapped, true
		}
		return mapProtoParts([]*checker.Type{t}, isReturn)
	}
	// String enums are unions of enum-literal members; classify before scalar fallback.
	if mapped, ok := classifyEnumLikeParts(parts); ok {
		return mapped, true
	}
	return mapProtoParts(parts, isReturn)
}

// mapRegisteredObjectMessage maps whitelisted TS object type aliases to protobuf messages.
func mapRegisteredObjectMessage(t *checker.Type) (string, bool) {
	name := typeAliasOrSymbolName(t)
	if name == "" {
		return "", false
	}
	return objectmessages.ProtoNameForTS(name)
}

func typeAliasOrSymbolName(t *checker.Type) string {
	if t == nil {
		return ""
	}
	if alias := t.Alias(); alias != nil {
		if sym := alias.Symbol(); sym != nil {
			if name := strings.TrimSpace(sym.Name); name != "" {
				return name
			}
		}
	}
	return typeSymbolName(t)
}

func stripNullishParts(parts []*checker.Type) []*checker.Type {
	out := make([]*checker.Type, 0, len(parts))
	for _, part := range parts {
		if part == nil {
			continue
		}
		flags := part.Flags()
		// Drop pure nullish constituents (null, undefined, or combinations of only those).
		if flags&^(checker.TypeFlagsNull|checker.TypeFlagsUndefined) == 0 {
			continue
		}
		out = append(out, part)
	}
	hasNonVoidLike := false
	for _, part := range out {
		if part.Flags()&checker.TypeFlagsVoidLike == 0 {
			hasNonVoidLike = true
			break
		}
	}
	if !hasNonVoidLike {
		return out
	}
	filtered := make([]*checker.Type, 0, len(out))
	for _, part := range out {
		if part.Flags()&checker.TypeFlagsVoidLike != 0 {
			continue
		}
		filtered = append(filtered, part)
	}
	return filtered
}

func mapSpecialCheckerType(c *checker.Checker, t *checker.Type) (string, bool) {
	if t == nil {
		return "", false
	}
	if t.Flags()&checker.TypeFlagsBigIntLike != 0 {
		return protoTypeInt64, true
	}
	if name := typeSymbolName(t); name != "" {
		switch name {
		case "Date":
			return protoTypeTimestamp, true
		case "Decimal", "BigDecimal":
			return protoTypeString, true
		}
	}
	if mapped, ok := mapEnumCheckerType(c, t); ok {
		return mapped, true
	}
	return "", false
}

func mapEnumCheckerType(_ *checker.Checker, t *checker.Type) (string, bool) {
	// Enum and enum-literal unions are classified from type flags / union parts.
	return classifyEnumLikeType(t)
}

// classifyEnumLikeType maps enum / enum-literal types to string or double.
// Multi-member string enums are handled via classifyEnumLikeParts from the
// union splitter in mapCheckerTypeToProto.
func classifyEnumLikeType(t *checker.Type) (string, bool) {
	if t == nil {
		return "", false
	}
	if t.Flags()&checker.TypeFlagsEnumLike == 0 {
		return "", false
	}
	if t.Flags()&checker.TypeFlagsStringLike != 0 {
		return protoTypeString, true
	}
	// Numeric and other enum-like types follow number→double.
	return protoTypeDouble, true
}

func classifyEnumLikeParts(parts []*checker.Type) (string, bool) {
	if len(parts) == 0 {
		return "", false
	}
	sawString := false
	sawNumber := false
	for _, part := range parts {
		if part == nil || part.Flags()&checker.TypeFlagsEnumLike == 0 {
			return "", false
		}
		if part.Flags()&checker.TypeFlagsStringLike != 0 {
			sawString = true
		}
		if part.Flags()&checker.TypeFlagsNumberLike != 0 {
			sawNumber = true
		}
	}
	switch {
	case sawString && !sawNumber:
		return protoTypeString, true
	default:
		// Numeric, mixed, or unclassified enum-like unions map like number→double.
		return protoTypeDouble, true
	}
}

func typeSymbolName(t *checker.Type) string {
	if t == nil || t.Symbol() == nil {
		return ""
	}
	return strings.TrimSpace(t.Symbol().Name)
}

func mapProtoParts(parts []*checker.Type, isReturn bool) (string, bool) {
	if len(parts) == 0 {
		return "", false
	}

	all := func(flag checker.TypeFlags) bool {
		for _, part := range parts {
			if part == nil || part.Flags()&flag == 0 {
				return false
			}
		}
		return true
	}

	switch {
	case all(checker.TypeFlagsStringLike):
		return protoTypeString, true
	case all(checker.TypeFlagsNumberLike):
		return protoTypeDouble, true
	case all(checker.TypeFlagsBooleanLike):
		return protoTypeBool, true
	case all(checker.TypeFlagsBigIntLike):
		return protoTypeInt64, true
	case isReturn && all(checker.TypeFlagsVoidLike):
		return protoTypeEmpty, true
	default:
		return "", false
	}
}

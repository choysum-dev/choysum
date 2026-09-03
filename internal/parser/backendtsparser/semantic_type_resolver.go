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

	"github.com/buke/typescript-go-internal/v7/pkg/ast"
	"github.com/buke/typescript-go-internal/v7/pkg/bundled"
	"github.com/buke/typescript-go-internal/v7/pkg/checker"
	"github.com/buke/typescript-go-internal/v7/pkg/compiler"
	"github.com/buke/typescript-go-internal/v7/pkg/core"
	"github.com/buke/typescript-go-internal/v7/pkg/tsoptions"
	"github.com/buke/typescript-go-internal/v7/pkg/vfs"
	"github.com/buke/typescript-go-internal/v7/pkg/vfs/osvfs"
	"github.com/buke/typescript-go-internal/v7/pkg/vfs/wrapvfs"
	"golang.org/x/sync/singleflight"
)

const (
	protoTypeString = "string"
	protoTypeDouble = "double"
	protoTypeBool   = "bool"
	protoTypeEmpty  = "google.protobuf.Empty"

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

// resolveProtoType prefers semantic reduction, then text mapping.
func (r *semanticTypeResolver) resolveProtoType(path, content, className, methodName, paramName string, isReturn bool, tsAnnotation string) string {
	if mapped, ok := r.trySemantic(path, content, className, methodName, paramName, isReturn); ok {
		return mapped
	}
	return getProtoTypeFromTsType(tsAnnotation)
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
		return cached, nil
	}
	r.mu.Unlock()

	v, err, _ := r.builds.Do(path+"\x00"+content, func() (any, error) {
		r.mu.Lock()
		if cached, ok := r.cache[path]; ok && cached != nil && cached.content == content {
			r.touchCacheLocked(path)
			r.mu.Unlock()
			return cached, nil
		}
		r.mu.Unlock()

		program, file, buildErr := buildSemanticProgramImpl(path, content)
		if buildErr != nil {
			return nil, buildErr
		}
		state := &semanticFileState{
			content: content,
			program: program,
			file:    file,
		}

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
	for _, stmt := range file.Statements.Nodes {
		if stmt == nil || stmt.Kind != ast.KindClassDeclaration {
			continue
		}
		name := nodeNameText(stmt)
		if className != "" && name != "" && name != className {
			continue
		}
		for _, member := range stmt.Members() {
			if member == nil || member.Kind != ast.KindMethodDeclaration {
				continue
			}
			if nodeNameText(member) == methodName {
				return member
			}
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

	parts := []*checker.Type{t}
	if t.IsUnion() {
		parts = t.Types()
	}
	return mapProtoParts(parts, isReturn)
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
	case isReturn && all(checker.TypeFlagsVoidLike):
		return protoTypeEmpty, true
	default:
		return "", false
	}
}

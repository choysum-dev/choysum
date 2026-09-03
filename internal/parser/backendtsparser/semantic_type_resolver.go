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
)

const (
	protoTypeString = "string"
	protoTypeDouble = "double"
	protoTypeBool   = "bool"
	protoTypeEmpty  = "google.protobuf.Empty"

	envDisableSemanticProto = "CHOYSUM_DISABLE_SEMANTIC_PROTO"
)

// semanticTypeResolver reduces service method parameter/return types to protobuf
// scalars via typescript-go-internal checker. Failures fall back to text mapping.
type semanticTypeResolver struct {
	mu       sync.Mutex
	cache    map[string]*semanticFileState
	disabled bool
	initOnce sync.Once
	logger   *slog.Logger
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
		if !bundled.Embedded {
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
func (r *semanticTypeResolver) resolveProtoType(path, content, methodName, paramName string, isReturn bool, tsAnnotation string) string {
	if mapped, ok := r.trySemantic(path, content, methodName, paramName, isReturn); ok {
		return mapped
	}
	return getProtoTypeFromTsType(tsAnnotation)
}

func (r *semanticTypeResolver) trySemantic(path, content, methodName, paramName string, isReturn bool) (string, bool) {
	if r == nil || !r.ensureEnabled() {
		return "", false
	}
	path = normalizeSemanticPath(path)
	if path == "" || content == "" || methodName == "" {
		return "", false
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	state, err := r.ensureFileLocked(path, content)
	if err != nil {
		r.disabled = true
		r.logWarn("semantic protobuf mapping disabled after program init failure", "path", path, "err", err)
		return "", false
	}
	if state == nil || state.file == nil || state.program == nil {
		return "", false
	}

	method := findClassMethodNode(state.file, methodName)
	if method == nil {
		return "", false
	}
	methodDecl := method.AsMethodDeclaration()
	var typeNode *ast.Node
	if isReturn {
		typeNode = methodDecl.Type
	} else {
		if paramName == "" || methodDecl.Parameters == nil {
			return "", false
		}
		for _, paramNode := range methodDecl.Parameters.Nodes {
			if paramNode == nil || paramNode.Kind != ast.KindParameter {
				continue
			}
			paramDecl := paramNode.AsParameterDeclaration()
			if paramDecl.Name() == nil {
				continue
			}
			name := strings.TrimSpace(paramDecl.Name().Text())
			if name == paramName {
				typeNode = paramDecl.Type
				break
			}
		}
	}
	if typeNode == nil {
		return "", false
	}

	c, done := state.program.GetTypeCheckerForFile(context.Background(), state.file)
	defer done()
	return mapCheckerTypeToProto(c, c.GetTypeFromTypeNode(typeNode), isReturn)
}

func (r *semanticTypeResolver) ensureFileLocked(path, content string) (*semanticFileState, error) {
	if cached, ok := r.cache[path]; ok && cached != nil && cached.content == content {
		return cached, nil
	}

	program, file, err := buildSemanticProgram(path, content)
	if err != nil {
		return nil, err
	}
	state := &semanticFileState{
		content: content,
		program: program,
		file:    file,
	}
	r.cache[path] = state
	return state, nil
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
	file := program.GetSourceFile(path)
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
	overlay := wrapvfs.Wrap(base, wrapvfs.Replacements{
		FileExists: func(p string) bool {
			if normalizeSemanticPath(p) == normalized {
				return true
			}
			return base.FileExists(p)
		},
		ReadFile: func(p string) (string, bool) {
			if normalizeSemanticPath(p) == normalized {
				return content, true
			}
			return base.ReadFile(p)
		},
	})
	return bundled.WrapFS(overlay)
}

func normalizeSemanticPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	return filepath.ToSlash(filepath.Clean(path))
}

func findClassMethodNode(file *ast.SourceFile, methodName string) *ast.Node {
	if file == nil || methodName == "" {
		return nil
	}
	for _, stmt := range file.Statements.Nodes {
		if stmt == nil || stmt.Kind != ast.KindClassDeclaration {
			continue
		}
		for _, member := range stmt.Members() {
			if member == nil || member.Kind != ast.KindMethodDeclaration {
				continue
			}
			if member.Name() == nil {
				continue
			}
			if strings.TrimSpace(member.Name().Text()) == methodName {
				return member
			}
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

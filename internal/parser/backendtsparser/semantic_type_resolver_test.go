// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package backendtsparser

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/buke/typescript-go-internal/v7/pkg/ast"
	"github.com/buke/typescript-go-internal/v7/pkg/bundled"
	"github.com/buke/typescript-go-internal/v7/pkg/checker"
	"github.com/buke/typescript-go-internal/v7/pkg/compiler"
	"github.com/choysum-dev/choysum/internal/parser"
	"github.com/choysum-dev/choysum/pkg/meta"
)

func TestSemanticTypeResolver_MapsAliasesAndLiteralUnions(t *testing.T) {
	if !bundled.Embedded {
		t.Skip("bundled libs not embedded")
	}
	if os.Getenv(envDisableSemanticProto) == "1" {
		t.Skip("semantic protobuf mapping disabled in environment")
	}

	runtimeScope := newBackendParserTestScope()
	module := &meta.Module{Path: "/virtual/modules/demo", ApplicationStr: "demo", Name: "demo"}
	p := NewTsParser(runtimeScope, module)

	path := "/virtual/modules/demo/service/model.ts"
	content := `
type Id = string
type Mode = 'read' | 'write'
type Flag = true | false
type Count = 1 | 2

export default class Demo {
  public static async ById(id: Id, mode: Mode, flag: Flag, count: Count): Promise<Id> {
    return id
  }

  public static async Done(): Promise<void | undefined> {}

  public static async Mixed(x: string | number): Promise<{ name: string }> {
    return { name: '' }
  }

  public static async NullableString(): Promise<string | null> {
    return null
  }

  public static async OptionalNumber(): Promise<number | undefined> {
    return undefined
  }

  public static async AnyValue(): Promise<any> {
    return 1
  }
}
`

	r, err := p.Parse(map[string]string{}, path, content)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if r.Model == nil {
		t.Fatal("expected model")
	}

	byName := map[string]*meta.Service{}
	for _, service := range r.Model.Services {
		byName[service.Name] = service
	}

	byId := byName["ById"]
	if byId == nil {
		t.Fatalf("missing ById, services=%v", r.Model.Services)
	}
	if byId.ProtobufType != "string" {
		t.Fatalf("ById return ProtobufType=%q, want string", byId.ProtobufType)
	}
	if len(byId.Parameters) != 4 {
		t.Fatalf("ById params=%v", byId.Parameters)
	}
	wantParams := []string{"string", "string", "bool", "double"}
	for i, want := range wantParams {
		if byId.Parameters[i].ProtobufType != want {
			t.Fatalf("ById param[%d] ProtobufType=%q, want %q", i, byId.Parameters[i].ProtobufType, want)
		}
	}

	done := byName["Done"]
	if done == nil || done.ProtobufType != "google.protobuf.Empty" {
		t.Fatalf("Done ProtobufType=%v", done)
	}

	mixed := byName["Mixed"]
	if mixed == nil {
		t.Fatal("missing Mixed")
	}
	if mixed.ProtobufType != "google.protobuf.Value" {
		t.Fatalf("Mixed return ProtobufType=%q, want Value", mixed.ProtobufType)
	}
	if len(mixed.Parameters) != 1 || mixed.Parameters[0].ProtobufType != "google.protobuf.Value" {
		t.Fatalf("Mixed params=%v", mixed.Parameters)
	}

	for _, name := range []string{"NullableString", "OptionalNumber", "AnyValue"} {
		service := byName[name]
		if service == nil || service.ProtobufType != "google.protobuf.Value" {
			t.Fatalf("%s ProtobufType=%v, want Value", name, service)
		}
	}
}

func TestSemanticTypeResolver_UsesSelectedModelClass(t *testing.T) {
	if !bundled.Embedded {
		t.Skip("bundled libs not embedded")
	}
	if os.Getenv(envDisableSemanticProto) == "1" {
		t.Skip("semantic protobuf mapping disabled in environment")
	}

	runtimeScope := newBackendParserTestScope()
	module := &meta.Module{Path: "/virtual/modules/demo", ApplicationStr: "demo", Name: "demo"}
	p := NewTsParser(runtimeScope, module)

	path := "/virtual/modules/demo/service/model.ts"
	content := `
class Helper {
  public static async Fetch(id: number): Promise<number> {
    return id
  }
}

export default class Demo {
  public static async Fetch(id: string): Promise<string> {
    return id
  }
}
`
	r, err := p.Parse(map[string]string{}, path, content)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if r.Model == nil || len(r.Model.Services) != 1 {
		t.Fatalf("unexpected model: %+v", r.Model)
	}
	service := r.Model.Services[0]
	if service.ProtobufType != "string" {
		t.Fatalf("return ProtobufType=%q, want string from Demo", service.ProtobufType)
	}
	if len(service.Parameters) != 1 || service.Parameters[0].ProtobufType != "string" {
		t.Fatalf("params=%v, want string from Demo", service.Parameters)
	}
}

func TestSemanticTypeResolver_AnonymousDefaultExportClass(t *testing.T) {
	if !bundled.Embedded {
		t.Skip("bundled libs not embedded")
	}
	if os.Getenv(envDisableSemanticProto) == "1" {
		t.Skip("semantic protobuf mapping disabled in environment")
	}

	path := "/virtual/modules/demo/service/anon.ts"
	content := `
export default class {
  public static async Fetch(id: string): Promise<string> {
    return id
  }
}
`
	r := newSemanticTypeResolver(nil)
	// Callers may pass a logical model name even when the class declaration is anonymous.
	got := r.resolveProtoType(path, content, "AnonModel", "Fetch", "id", false, "string")
	if got != "string" {
		t.Fatalf("anonymous class param ProtobufType=%q, want string", got)
	}
	got = r.resolveProtoType(path, content, "AnonModel", "Fetch", "", true, "string")
	if got != "string" {
		t.Fatalf("anonymous class return ProtobufType=%q, want string", got)
	}
}

func TestSemanticTypeResolver_FallsBackWhenDisabled(t *testing.T) {
	t.Setenv(envDisableSemanticProto, "1")

	runtimeScope := newBackendParserTestScope()
	module := &meta.Module{Path: "/virtual/modules/demo", ApplicationStr: "demo", Name: "demo"}
	p := NewTsParser(runtimeScope, module)

	path := "/virtual/modules/demo/service/model.ts"
	content := `
type Id = string
export default class Demo {
  public static async ById(id: Id): Promise<Id> {
    return id
  }
}
`
	r, err := p.Parse(map[string]string{}, path, content)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if r.Model == nil || len(r.Model.Services) != 1 {
		t.Fatalf("unexpected model: %+v", r.Model)
	}
	service := r.Model.Services[0]
	if service.ProtobufType != "google.protobuf.Value" {
		t.Fatalf("return ProtobufType=%q, want Value fallback", service.ProtobufType)
	}
	if len(service.Parameters) != 1 || service.Parameters[0].ProtobufType != "google.protobuf.Value" {
		t.Fatalf("params=%v", service.Parameters)
	}
}

func TestSemanticTypeResolver_BranchCoverage(t *testing.T) {
	if !bundled.Embedded {
		t.Skip("bundled libs not embedded")
	}

	t.Run("nil receiver and logger logWarn", func(t *testing.T) {
		(*semanticTypeResolver)(nil).logWarn("ignored")
		newSemanticTypeResolver(nil).logWarn("ignored-nil-logger")
		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, nil))
		newSemanticTypeResolver(logger).logWarn("logged", "k", "v")
		if !bytes.Contains(buf.Bytes(), []byte("logged")) {
			t.Fatalf("expected logged warning, got %q", buf.String())
		}
	})

	t.Run("disabled when libs not embedded", func(t *testing.T) {
		t.Setenv(envDisableSemanticProto, "")
		prev := semanticLibsEmbedded
		semanticLibsEmbedded = false
		defer func() { semanticLibsEmbedded = prev }()

		r := newSemanticTypeResolver(nil)
		if r.ensureEnabled() {
			t.Fatal("expected semantic mapping disabled without embedded libs")
		}
		got := r.resolveProtoType("/x.ts", "export class X {}", "X", "M", "", true, "string")
		if got != "string" {
			t.Fatalf("fallback got %q", got)
		}
	})

	t.Run("empty inputs and cache hit", func(t *testing.T) {
		r := newSemanticTypeResolver(nil)
		if _, ok := r.trySemantic("", "content", "Demo", "ById", "", true); ok {
			t.Fatal("empty path should fail")
		}
		if _, ok := r.trySemantic("/virtual/a.ts", "", "Demo", "ById", "", true); ok {
			t.Fatal("empty content should fail")
		}
		if _, ok := r.trySemantic("/virtual/a.ts", "content", "Demo", "", "", true); ok {
			t.Fatal("empty method should fail")
		}

		path := "/virtual/modules/demo/service/cache.ts"
		content := `
export default class Demo {
  public static async Echo(v: string): Promise<string> { return v }
}
`
		got1 := r.resolveProtoType(path, content, "Demo", "Echo", "v", false, "string")
		got2 := r.resolveProtoType(path, content, "Demo", "Echo", "v", false, "string")
		if got1 != "string" || got2 != "string" {
			t.Fatalf("cache path got %q %q", got1, got2)
		}
	})

	t.Run("program init failure does not disable later files", func(t *testing.T) {
		r := newSemanticTypeResolver(nil)
		prev := buildSemanticProgramImpl
		buildSemanticProgramImpl = func(path, content string) (*compiler.Program, *ast.SourceFile, error) {
			return nil, nil, errors.New("boom")
		}
		defer func() { buildSemanticProgramImpl = prev }()

		if mapped, ok := r.trySemantic("/virtual/fail.ts", "export class X {}", "X", "M", "", true); ok || mapped != "" {
			t.Fatalf("expected failure fallback, got %q %v", mapped, ok)
		}
		if !r.ensureEnabled() {
			t.Fatal("single-file failure must not disable resolver")
		}
	})

	t.Run("missing source file error path", func(t *testing.T) {
		prev := semanticProgramSourceFile
		semanticProgramSourceFile = func(program *compiler.Program, path string) *ast.SourceFile { return nil }
		defer func() { semanticProgramSourceFile = prev }()

		_, _, err := buildSemanticProgram("/virtual/missing.ts", "export class X {}")
		if err == nil {
			t.Fatal("expected missing source file error")
		}
		if err.Error() != string(errSemanticSourceFileMissing) {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("relative path currentDir fallback and overlay miss", func(t *testing.T) {
		fs := newSemanticOverlayFS("model.ts", "export const x = 1")
		if !fs.FileExists("model.ts") {
			t.Fatal("expected overlay file")
		}
		if _, ok := fs.ReadFile("/definitely-missing-choysum-semantic-file"); ok {
			t.Fatal("unexpected hit for missing file")
		}
		// Bare relative paths exercise filepath.Dir → "." → "/" currentDir fallback.
		_, _, _ = buildSemanticProgram("model.ts", `
export default class Demo {
  public static async Ping(): Promise<boolean> { return true }
}
`)
	})

	t.Run("method and param lookup misses", func(t *testing.T) {
		r := newSemanticTypeResolver(nil)
		path := "/virtual/modules/demo/service/lookup.ts"
		content := `
export default class Demo {
  public static async Typed(x: string): Promise<string> { return x }
  public static async Untyped(x) { return x }
}
`
		if _, ok := r.trySemantic(path, content, "Demo", "Missing", "", true); ok {
			t.Fatal("missing method should fail")
		}
		if _, ok := r.trySemantic(path, content, "Helper", "Typed", "", true); ok {
			t.Fatal("wrong class should fail")
		}
		if _, ok := r.trySemantic(path, content, "Demo", "Typed", "", false); ok {
			t.Fatal("empty param name should fail")
		}
		if _, ok := r.trySemantic(path, content, "Demo", "Typed", "nope", false); ok {
			t.Fatal("missing param should fail")
		}
		if _, ok := r.trySemantic(path, content, "Demo", "Untyped", "", true); ok {
			t.Fatal("missing return annotation should fail")
		}
	})

	t.Run("findClassMethodNode guards", func(t *testing.T) {
		if findClassMethodNode(nil, "Demo", "X") != nil {
			t.Fatal("nil file")
		}
		if findClassMethodNode(&ast.SourceFile{}, "Demo", "") != nil {
			t.Fatal("empty method")
		}
		path := "/virtual/modules/demo/service/find.ts"
		content := `
class Other { public static async X(): Promise<void> {} }
export default class Demo {
  public static async Y(): Promise<void> {}
  public z = 1
}
`
		program, file, err := buildSemanticProgram(path, content)
		if err != nil {
			t.Fatalf("build: %v", err)
		}
		_ = program
		if findClassMethodNode(file, "Demo", "X") != nil {
			t.Fatal("method on other class must not match")
		}
		if findClassMethodNode(file, "Demo", "missing") != nil {
			t.Fatal("missing method")
		}
		if findClassMethodNode(file, "Demo", "Y") == nil {
			t.Fatal("expected Demo.Y")
		}
	})

	t.Run("program cache is bounded and replaces content", func(t *testing.T) {
		prevLimit := semanticProgramCacheLimit
		semanticProgramCacheLimit = 2
		defer func() { semanticProgramCacheLimit = prevLimit }()

		r := newSemanticTypeResolver(nil)
		contentA := `
export default class Demo {
  public static async Echo(v: string): Promise<string> { return v }
}
`
		paths := []string{
			"/virtual/modules/demo/service/a.ts",
			"/virtual/modules/demo/service/b.ts",
			"/virtual/modules/demo/service/c.ts",
		}
		for _, path := range paths {
			got := r.resolveProtoType(path, contentA, "Demo", "Echo", "v", false, "string")
			if got != "string" {
				t.Fatalf("path %s got %q", path, got)
			}
		}
		r.mu.Lock()
		size := len(r.cache)
		_, hasA := r.cache[paths[0]]
		_, hasC := r.cache[paths[2]]
		r.mu.Unlock()
		if size != 2 {
			t.Fatalf("cache size=%d, want 2", size)
		}
		if hasA {
			t.Fatal("oldest path should have been evicted")
		}
		if !hasC {
			t.Fatal("newest path should remain cached")
		}

		contentB := `
export default class Demo {
  public static async Echo(v: number): Promise<number> { return v }
}
`
		got := r.resolveProtoType(paths[2], contentB, "Demo", "Echo", "v", false, "number")
		if got != "double" {
			t.Fatalf("content replace got %q", got)
		}
		r.mu.Lock()
		cached := r.cache[paths[2]]
		r.mu.Unlock()
		if cached == nil || cached.content != contentB {
			t.Fatal("expected cache entry replaced for updated content")
		}

		// When the newest key is also front of cacheOrder, stop without dropping it from order.
		semanticProgramCacheLimit = 1
		r.mu.Lock()
		r.cacheOrder = []string{paths[2]}
		r.cache = map[string]*semanticFileState{
			paths[2]:             cached,
			"/virtual/orphan.ts": cached,
		}
		r.putCacheLocked(paths[2], cached)
		if _, ok := r.cache[paths[2]]; !ok {
			r.mu.Unlock()
			t.Fatal("newest path must remain in cache")
		}
		found := false
		for _, ordered := range r.cacheOrder {
			if ordered == paths[2] {
				found = true
				break
			}
		}
		r.mu.Unlock()
		if !found {
			t.Fatal("newest path must remain in cacheOrder")
		}
	})

	t.Run("helper edge coverage", func(t *testing.T) {
		if nodeNameText(nil) != "" {
			t.Fatal("nil node name")
		}
		factory := ast.NewNodeFactory(ast.NodeFactoryHooks{})
		unnamedMethod := factory.NewMethodDeclaration(nil, nil, nil, nil, nil, nil, nil, nil, nil)
		if nodeNameText(unnamedMethod) != "" {
			t.Fatal("unnamed method should have empty name text")
		}
		unnamedParam := factory.NewParameterDeclaration(nil, nil, nil, nil, nil, nil)
		nonParam := factory.NewMethodDeclaration(nil, nil, nil, nil, nil, nil, nil, nil, nil)
		if findParamTypeNode([]*ast.Node{nil, unnamedParam, nonParam}, "x") != nil {
			t.Fatal("nil/unnamed/non-parameter nodes should not match")
		}
		if findParamTypeNode(nil, "") != nil {
			t.Fatal("empty param name")
		}

		if got, ok := mapProtoParts(nil, true); ok || got != "" {
			t.Fatalf("empty parts (%q,%v)", got, ok)
		}
		if got, ok := mapProtoParts([]*checker.Type{nil}, true); ok || got != "" {
			t.Fatalf("nil part (%q,%v)", got, ok)
		}

		path := "/virtual/modules/demo/service/any.ts"
		content := `export default class Demo { public static async X(): Promise<string> { return '' } }`
		program, file, err := buildSemanticProgram(path, content)
		if err != nil {
			t.Fatalf("build: %v", err)
		}
		c, done := program.GetTypeCheckerForFileExclusive(context.Background(), file)
		defer done()
		if got, ok := mapCheckerTypeToProto(c, c.GetAnyType(), true); ok || got != "" {
			t.Fatalf("any (%q,%v)", got, ok)
		}
		if got, ok := mapCheckerTypeToProto(c, c.GetErrorType(), true); ok || got != "" {
			t.Fatalf("error (%q,%v)", got, ok)
		}
		if got, ok := mapCheckerTypeToProto(c, c.GetUnknownType(), false); ok || got != "" {
			t.Fatalf("unknown (%q,%v)", got, ok)
		}
	})
}

func TestSemanticTypeResolver_ConcurrentSameFileBuildsOnce(t *testing.T) {
	if !bundled.Embedded {
		t.Skip("bundled libs not embedded")
	}
	if os.Getenv(envDisableSemanticProto) == "1" {
		t.Skip("semantic protobuf mapping disabled in environment")
	}

	orig := buildSemanticProgramImpl
	t.Cleanup(func() { buildSemanticProgramImpl = orig })

	var builds atomic.Int32
	buildSemanticProgramImpl = func(path, content string) (*compiler.Program, *ast.SourceFile, error) {
		builds.Add(1)
		time.Sleep(50 * time.Millisecond)
		return orig(path, content)
	}

	r := newSemanticTypeResolver(nil)
	path := "/virtual/modules/demo/service/concurrent.ts"
	content := `
export default class Demo {
  public static async Echo(v: string): Promise<string> { return v }
}
`
	const n = 8
	var wg sync.WaitGroup
	errs := make(chan string, n)
	wg.Add(n)
	for range n {
		go func() {
			defer wg.Done()
			got := r.resolveProtoType(path, content, "Demo", "Echo", "v", false, "string")
			if got != "string" {
				errs <- "got " + got
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	if got := builds.Load(); got != 1 {
		t.Fatalf("builds=%d, want 1", got)
	}
}

func TestSemanticTypeResolver_ConcurrentCachedLookups(t *testing.T) {
	if !bundled.Embedded {
		t.Skip("bundled libs not embedded")
	}
	if os.Getenv(envDisableSemanticProto) == "1" {
		t.Skip("semantic protobuf mapping disabled in environment")
	}

	r := newSemanticTypeResolver(nil)
	path := "/virtual/modules/demo/service/cached_concurrent.ts"
	content := `
export default class Demo {
  public static async Echo(v: string): Promise<number> { return 1 }
}
`
	if got := r.resolveProtoType(path, content, "Demo", "Echo", "", true, "number"); got != "double" {
		t.Fatalf("warm cache got %q", got)
	}

	const n = 16
	var wg sync.WaitGroup
	errs := make(chan string, n)
	wg.Add(n)
	for range n {
		go func() {
			defer wg.Done()
			ret := r.resolveProtoType(path, content, "Demo", "Echo", "", true, "number")
			param := r.resolveProtoType(path, content, "Demo", "Echo", "v", false, "string")
			if ret != "double" || param != "string" {
				errs <- "ret=" + ret + " param=" + param
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
}

func TestResolveProtobufType_WithoutSemanticFallsBack(t *testing.T) {
	p := &tsFileParser{TsParser: &parser.TsParser{}}
	if got := p.resolveProtobufType("Demo", "M", "x", false, "boolean"); got != "bool" {
		t.Fatalf("got %q", got)
	}
}

func TestMapCheckerTypeToProto_NilSafe(t *testing.T) {
	if got, ok := mapCheckerTypeToProto(nil, nil, true); ok || got != "" {
		t.Fatalf("got (%q, %v)", got, ok)
	}
}

func TestSemanticOverlayRejectsEmptyPathMatch(t *testing.T) {
	fs := newSemanticOverlayFS("   ", "secret")
	// Empty/blank source paths must not become a catch-all overlay match.
	if got, ok := fs.ReadFile("/virtual/modules/demo/service/x.ts"); ok && got == "secret" {
		t.Fatal("empty overlay path must not serve overlay content for other lookups")
	}
}

func TestSemanticOverlayPathMatch(t *testing.T) {
	const overlay = "/Virtual/Modules/Demo/service/Model.ts"
	if semanticOverlayPathMatch("", overlay, true) {
		t.Fatal("empty overlay must not match")
	}
	if !semanticOverlayPathMatch(overlay, overlay, true) {
		t.Fatal("exact match required on case-sensitive FS")
	}
	if semanticOverlayPathMatch(overlay, strings.ToLower(overlay), true) {
		t.Fatal("case-sensitive FS must reject casing drift")
	}
	if !semanticOverlayPathMatch(overlay, strings.ToLower(overlay), false) {
		t.Fatal("case-insensitive FS must accept casing drift")
	}
}

func TestSemanticOverlayUsesAbsPathAndOSFallback(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.ts")
	content := "export const n = 1\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	fs := newSemanticOverlayFS(path, "export const overlay = 1\n")
	got, ok := fs.ReadFile(path)
	if !ok || got != "export const overlay = 1\n" {
		t.Fatalf("overlay read got %q ok=%v", got, ok)
	}
	other := filepath.Join(dir, "other.ts")
	if err := os.WriteFile(other, []byte("export const other = 2\n"), 0o644); err != nil {
		t.Fatalf("write other: %v", err)
	}
	got, ok = fs.ReadFile(other)
	if !ok || got != "export const other = 2\n" {
		t.Fatalf("os fallback got %q ok=%v", got, ok)
	}
	if !fs.FileExists(other) {
		t.Fatal("expected other to exist via os fallback")
	}
}

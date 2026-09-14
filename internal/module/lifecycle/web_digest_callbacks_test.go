// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/choysum-dev/choysum/internal/defaultscope"
	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
)

func TestForceWebBuildFromEnv(t *testing.T) {
	t.Setenv("CHOYSUM_FORCE_WEB_BUILD", "")
	if forceWebBuildFromEnv() {
		t.Fatal("empty env should be false")
	}
	for _, v := range []string{"1", "true", "YES", "on"} {
		t.Setenv("CHOYSUM_FORCE_WEB_BUILD", v)
		if !forceWebBuildFromEnv() {
			t.Fatalf("%q should force rebuild", v)
		}
	}
	t.Setenv("CHOYSUM_FORCE_WEB_BUILD", "nope")
	if forceWebBuildFromEnv() {
		t.Fatal("unknown value should be false")
	}
}

func TestGlobalWebSkipCallbacks(t *testing.T) {
	cfg := &config.Config{
		Db: &config.DbConfig{
			Dialect: "sqlite",
			DSN:     filepath.Join(t.TempDir(), "web-digest.db"),
		},
		Compile: &config.CompileConfig{SourceMap: true, Minify: false, TreeShaking: false, BundleMode: "bundle"},
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	runtimeScope := defaultscope.NewDefaultScope(
		context.Background(),
		scopetest.FactoryInputFromConfig(cfg),
		logger,
	)
	if err := runtimeScope.Session().AutoMigrate(modmeta.CatalogEntities()...); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}

	modulesPath := t.TempDir()
	distWeb := filepath.Join(t.TempDir(), "web")
	if err := os.MkdirAll(distWeb, 0o755); err != nil {
		t.Fatalf("mkdir web: %v", err)
	}
	if err := os.WriteFile(filepath.Join(distWeb, "index.html"), []byte("<html>ok</html>"), 0o644); err != nil {
		t.Fatalf("write index: %v", err)
	}
	modPath := filepath.Join(modulesPath, "web")
	entry := filepath.Join(modPath, "web", "index.ts")
	if err := os.MkdirAll(filepath.Dir(entry), 0o755); err != nil {
		t.Fatalf("mkdir entry: %v", err)
	}
	if err := os.WriteFile(entry, []byte("export default {}\n"), 0o644); err != nil {
		t.Fatalf("write entry: %v", err)
	}
	mod := &meta.Module{
		Name: "web", Version: "1.0.0", Status: meta.Installed,
		Path: modPath, WebEntryPoint: "web/index.ts", ApplicationStr: "web",
	}
	if err := runtimeScope.Session().Create(mod).Error; err != nil {
		t.Fatalf("create module: %v", err)
	}

	manager := NewModuleManager(runtimeScope, nil)
	manager.runtimeOptions = runtimeOptions{
		modulesPath:        modulesPath,
		distPath:           filepath.Dir(distWeb),
		tmpPath:            t.TempDir(),
		defaultChoysumPath: t.TempDir(),
		compileBundleMode:  config.NewDefaultCompileConfig().BundleMode,
	}

	skipFn, rememberFn := manager.globalWebSkipCallbacks()
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := skipFn(canceled, distWeb); err == nil {
		t.Fatal("expected canceled skip")
	}
	if err := rememberFn(canceled, distWeb, "x"); err == nil {
		t.Fatal("expected canceled remember")
	}

	skip, dig, err := skipFn(context.Background(), distWeb)
	if err != nil {
		t.Fatalf("skipFn: %v", err)
	}
	if dig == "" {
		t.Fatal("expected digest")
	}
	if skip {
		t.Fatal("first call should rebuild (no stamp)")
	}
	if err := rememberFn(context.Background(), distWeb, dig); err != nil {
		t.Fatalf("remember: %v", err)
	}
	skip, dig2, err := skipFn(context.Background(), distWeb)
	if err != nil || !skip || dig2 != dig {
		t.Fatalf("second call skip=%v dig=%q err=%v", skip, dig2, err)
	}

	t.Setenv("CHOYSUM_FORCE_WEB_BUILD", "1")
	skip, forcedDig, err := skipFn(context.Background(), distWeb)
	if err != nil || skip || forcedDig == "" {
		t.Fatalf("force rebuild: skip=%v dig=%q err=%v", skip, forcedDig, err)
	}
	t.Setenv("CHOYSUM_FORCE_WEB_BUILD", "")

	// Digest compare failure (stamp path is a directory) rebuilds without error.
	stampPath := filepath.Join(distWeb, ".choysum_web_input_digest")
	if err := os.Remove(stampPath); err != nil {
		t.Fatalf("remove stamp: %v", err)
	}
	if err := os.Mkdir(stampPath, 0o755); err != nil {
		t.Fatalf("mkdir stamp: %v", err)
	}
	skip, dig, err = skipFn(context.Background(), distWeb)
	if err != nil || skip || dig == "" {
		t.Fatalf("stamp-dir compare fallback: skip=%v dig=%q err=%v", skip, dig, err)
	}
	_ = os.RemoveAll(stampPath)

	// Digest compute failure (unreadable hashed source) rebuilds without error.
	t.Run("unreadable api web digest fallback", func(t *testing.T) {
		apiWeb := filepath.Join(modulesPath, "api", "web", "blocked.ts")
		if err := os.MkdirAll(filepath.Dir(apiWeb), 0o755); err != nil {
			t.Fatalf("mkdir api web: %v", err)
		}
		if err := os.WriteFile(apiWeb, []byte("export {}\n"), 0o644); err != nil {
			t.Fatalf("write api web: %v", err)
		}
		if err := os.Chmod(apiWeb, 0); err != nil {
			t.Fatalf("chmod api web: %v", err)
		}
		t.Cleanup(func() { _ = os.Chmod(apiWeb, 0o644) })
		if _, err := os.ReadFile(apiWeb); err == nil {
			t.Skip("filesystem permits read despite mode 0")
		}
		skip, dig, err := skipFn(context.Background(), distWeb)
		if err != nil || skip || dig != "" {
			t.Fatalf("digest compute fallback: skip=%v dig=%q err=%v", skip, dig, err)
		}
	})

	// Load failure path: nil session manager still returns skip=false without error.
	bad := &ModuleManager{runtimeScope: &webSkipNilSessionScope{inner: runtimeScope}}
	badSkip, _ := bad.globalWebSkipCallbacks()
	skip, dig, err = badSkip(context.Background(), distWeb)
	if err != nil || skip || dig != "" {
		t.Fatalf("nil session fallback: skip=%v dig=%q err=%v", skip, dig, err)
	}
}

type webSkipNilSessionScope struct {
	inner scope.Scope
}

func (s *webSkipNilSessionScope) Run(fn func(scope.Scope) error) error { return fn(s) }
func (s *webSkipNilSessionScope) Transactor() scope.Transactor         { return s.inner.Transactor() }
func (s *webSkipNilSessionScope) Session() *scope.Session              { return nil }
func (s *webSkipNilSessionScope) WithContext(ctx context.Context) scope.Scope {
	return &webSkipNilSessionScope{inner: s.inner.WithContext(ctx)}
}
func (s *webSkipNilSessionScope) Context() context.Context { return s.inner.Context() }
func (s *webSkipNilSessionScope) Logger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

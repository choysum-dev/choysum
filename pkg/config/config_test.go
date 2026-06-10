// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package config

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"
)

func writeTestConfig(t *testing.T, body string) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(strings.TrimSpace(body)+"\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

func canonicalPath(t *testing.T, path string) string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(path)
	if err == nil {
		path = resolved
	}
	return filepath.Clean(path)
}

func TestResolveDefaultChoysumPaths(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	root, err := ResolveDefaultChoysumPaths()
	if err != nil {
		t.Fatalf("resolveDefaultChoysumPaths() error = %v", err)
	}

	wantRoot, _ := filepath.Abs(filepath.Join(homeDir, ".choysum"))
	if canonicalPath(t, root) != canonicalPath(t, wantRoot) {
		t.Fatalf("root = %q, want %q", root, wantRoot)
	}
}

func TestResolveDefaultTmpPath(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	tmpPath, err := ResolveDefaultTmpPath()
	if err != nil {
		t.Fatalf("ResolveDefaultTmpPath() error = %v", err)
	}

	wantTmpPath, _ := filepath.Abs(filepath.Join(homeDir, ".choysum", "tmp"))
	if canonicalPath(t, tmpPath) != canonicalPath(t, wantTmpPath) {
		t.Fatalf("tmp path = %q, want %q", tmpPath, wantTmpPath)
	}
}

func TestResolveDefaultDistPath(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	distPath, err := ResolveDefaultDistPath()
	if err != nil {
		t.Fatalf("ResolveDefaultDistPath() error = %v", err)
	}

	wantDistPath, _ := filepath.Abs(filepath.Join(homeDir, ".choysum", "dist"))
	if canonicalPath(t, distPath) != canonicalPath(t, wantDistPath) {
		t.Fatalf("dist path = %q, want %q", distPath, wantDistPath)
	}
}

func TestNewConfigMergesDefaultsAndNormalizesPaths(t *testing.T) {
	type customSection struct {
		Enabled bool `mapstructure:"enabled"`
	}

	var custom customSection
	cfgPath := writeTestConfig(t, strings.Join([]string{
		"default_choysum_path: ./.choysum-bootstrap",
		"modules_path: rel-modules",
		"dist_path: rel-dist",
		"npm_path: rel-npm",
		"compile:",
		"  bundleMode: \" application \"",
		"  production: false",
		"auth:",
		"  jwt:",
		"    identityCache:",
		"      enabled: false",
		"  grpcEntryPolicy:",
		"    auth.User/Login:",
		"      skipAuthentication: true",
		"frontendEnv:",
		"  CHOYSUM_APP_NAME: Custom UI",
		"backendEnv:",
		"  CHOYSUM_SOFT_DELETE_ENABLED: false",
		"custom:",
		"  enabled: true",
	}, "\n"))

	cfg, err := NewConfig(
		cfgPath,
		UnmarshalKey("custom", &custom),
		AfterUnmarshal(func(_ *viper.Viper, cfg *Config) error {
			cfg.BackendEnv["POST_HOOK"] = "done"
			return nil
		}),
	)
	if err != nil {
		t.Fatalf("NewConfig returned error: %v", err)
	}

	if !filepath.IsAbs(cfg.ModulesPath) || !filepath.IsAbs(cfg.DistPath) || !filepath.IsAbs(cfg.NpmPath) || !filepath.IsAbs(cfg.DefaultChoysumPath) || !filepath.IsAbs(cfg.TmpPath) {
		t.Fatalf("expected absolute paths, got modules=%q dist=%q npm=%q default_choysum=%q tmp=%q", cfg.ModulesPath, cfg.DistPath, cfg.NpmPath, cfg.DefaultChoysumPath, cfg.TmpPath)
	}
	if cfg.NPMRegistryURL != DefaultNPMRegistryURL {
		t.Fatalf("npm_registry_url = %q, want %q", cfg.NPMRegistryURL, DefaultNPMRegistryURL)
	}
	if cfg.ModuleCatalogIndexURL != DefaultModuleCatalogIndexURL {
		t.Fatalf("module_catalog_index_url = %q, want %q", cfg.ModuleCatalogIndexURL, DefaultModuleCatalogIndexURL)
	}
	if strings.TrimSpace(cfg.DefaultChoysumPath) == "" {
		t.Fatal("expected default_choysum_path to be non-empty")
	}
	wantTmp := filepath.Join(cfg.DefaultChoysumPath, "tmp")
	if canonicalPath(t, cfg.TmpPath) != canonicalPath(t, wantTmp) {
		t.Fatalf("tmp path = %q, want %q", cfg.TmpPath, wantTmp)
	}
	if got, want := cfg.Compile.BundleMode, string(BundleModeApplication); got != want {
		t.Fatalf("bundle mode = %q, want %q", got, want)
	}
	if cfg.Auth == nil || cfg.Auth.JWT == nil {
		t.Fatal("expected auth and jwt config to be initialized")
	}
	if !cfg.Auth.Enabled {
		t.Fatal("expected auth.enabled to default to true when omitted")
	}
	if cfg.Auth.Type != "jwt" {
		t.Fatalf("expected auth.type default to jwt, got %q", cfg.Auth.Type)
	}
	if cfg.Auth.JWT.IdentityCache == nil || cfg.Auth.JWT.IdentityCache.Enabled {
		t.Fatal("expected jwt.identityCache.enabled override to be applied")
	}
	if cfg.Auth.JWT.IdentityCache.Backend != "memory" {
		t.Fatalf("expected jwt.identityCache.backend default to memory, got %q", cfg.Auth.JWT.IdentityCache.Backend)
	}
	if !cfg.Auth.GrpcAuthentication || !cfg.Auth.GrpcMethodAccess || !cfg.Auth.GrpcRecordRule || !cfg.Auth.GrpcCompanyFilter || !cfg.Auth.GrpcFieldRule {
		t.Fatal("expected auth boolean defaults to remain enabled when omitted")
	}
	if !reflect.DeepEqual(cfg.Auth.JobTokenAllowedSANs, NewDefaultAuthConfig().JobTokenAllowedSANs) {
		t.Fatalf("job token SAN defaults not merged: got %#v", cfg.Auth.JobTokenAllowedSANs)
	}
	if cfg.Auth.GrpcEntryPolicy["auth.User/Register"] == nil {
		t.Fatal("expected default grpc entry policy to be merged")
	}
	if cfg.Auth.GrpcEntryPolicy["auth.User/Login"] == nil || !cfg.Auth.GrpcEntryPolicy["auth.User/Login"].SkipAuthentication {
		t.Fatal("expected configured grpc entry policy to be preserved")
	}
	if got := cfg.FrontendEnv["BASE_URL"]; got == nil || got == "" {
		t.Fatalf("expected default frontend env BASE_URL to exist, got %#v", got)
	}
	if got := cfg.BackendEnv["POST_HOOK"]; got != "done" {
		t.Fatalf("after unmarshal hook result = %#v, want %q", got, "done")
	}
	if !custom.Enabled {
		t.Fatal("expected custom section to unmarshal")
	}
}

func TestNewConfigRejectsLegacyJWTCacheKeys(t *testing.T) {
	cfgPath := writeTestConfig(t, `
default_choysum_path: ./.choysum-bootstrap
auth:
  jwt:
    cacheEnabled: false
    cacheSize: 128
    cacheTTL: 2m
`)

	_, err := NewConfig(cfgPath)
	if err == nil {
		t.Fatal("expected legacy JWT identity cache keys to be rejected")
	}
	if !strings.Contains(err.Error(), "auth.jwt.cacheEnabled (use auth.jwt.identityCache.enabled)") {
		t.Fatalf("expected cacheEnabled guidance in error, got %v", err)
	}
	if !strings.Contains(err.Error(), "auth.jwt.cacheSize (use auth.jwt.identityCache.size)") {
		t.Fatalf("expected cacheSize guidance in error, got %v", err)
	}
	if !strings.Contains(err.Error(), "auth.jwt.cacheTTL (use auth.jwt.identityCache.ttl)") {
		t.Fatalf("expected cacheTTL guidance in error, got %v", err)
	}
}

func TestNewConfigRejectsLegacyModuleCatalogKeys(t *testing.T) {
	cfgPath := writeTestConfig(t, `
registry_index_url: https://index.example.dev/v1/index.json
registries:
  official:
    url: https://index.example.dev/v1/index.json
    indexURL: https://index.example.dev/v1/index.json
`)

	_, err := NewConfig(cfgPath)
	if err == nil {
		t.Fatal("expected legacy module catalog keys to be rejected")
	}
	if !strings.Contains(err.Error(), "legacy module catalog config keys are no longer supported") {
		t.Fatalf("expected legacy module catalog rejection header, got %v", err)
	}
	if !strings.Contains(err.Error(), "registry_index_url (use module_catalog_index_url)") {
		t.Fatalf("expected registry_index_url guidance, got %v", err)
	}
	if !strings.Contains(err.Error(), "registries (use module_catalog_index_url)") {
		t.Fatalf("expected registries guidance, got %v", err)
	}
	if !strings.Contains(err.Error(), "registries.official.url (use module_catalog_index_url)") {
		t.Fatalf("expected registries.official.url guidance, got %v", err)
	}
	if !strings.Contains(err.Error(), "registries.official.indexURL (use module_catalog_index_url)") {
		t.Fatalf("expected registries.official.indexURL guidance, got %v", err)
	}
}

func TestMergeStringMapUsesOverrideValues(t *testing.T) {
	merged := mergeStringMap(
		map[string]any{"shared": "base", "baseOnly": true},
		map[string]any{"shared": "override", "overrideOnly": 1},
	)
	if got := merged["shared"]; got != "override" {
		t.Fatalf("shared value = %#v, want %q", got, "override")
	}
	if got := merged["baseOnly"]; got != true {
		t.Fatalf("baseOnly = %#v, want true", got)
	}
	if got := merged["overrideOnly"]; got != 1 {
		t.Fatalf("overrideOnly = %#v, want 1", got)
	}
}

func TestNewConfigRejectsInvalidBundleMode(t *testing.T) {
	cfgPath := writeTestConfig(t, `
compile:
  bundleMode: invalid
`)

	_, err := NewConfig(cfgPath)
	if err == nil {
		t.Fatal("expected invalid bundle mode error")
	}
	if !strings.Contains(err.Error(), "invalid compile.bundleMode") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewConfigDefaultsMissingDefaultChoysumPath(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("CHOYSUM_DEFAULT_CHOYSUM_PATH", "")

	cfgPath := writeTestConfig(t, `
modules_path: rel-modules
`)

	cfg, err := NewConfig(cfgPath)
	if err != nil {
		t.Fatalf("NewConfig returned error: %v", err)
	}

	wantDefaultChoysumPath, _ := filepath.Abs(filepath.Join(homeDir, ".choysum"))
	if canonicalPath(t, cfg.DefaultChoysumPath) != canonicalPath(t, wantDefaultChoysumPath) {
		t.Fatalf("default_choysum_path = %q, want %q", cfg.DefaultChoysumPath, wantDefaultChoysumPath)
	}
	wantTmpPath := filepath.Join(wantDefaultChoysumPath, "tmp")
	if canonicalPath(t, cfg.TmpPath) != canonicalPath(t, wantTmpPath) {
		t.Fatalf("tmp_path = %q, want %q", cfg.TmpPath, wantTmpPath)
	}
}

func TestNewConfigNormalizesRelativeJWTKeyPathsAgainstConfigDir(t *testing.T) {
	cfgPath := writeTestConfig(t, `
default_choysum_path: ./.choysum-custom
auth:
  jwt:
    privateKeyFile: ./jwt/private.pem
    publicKeyFile: ./jwt/public.pem
`)

	cfg, err := NewConfig(cfgPath)
	if err != nil {
		t.Fatalf("NewConfig returned error: %v", err)
	}

	wantPrivate := filepath.Join(filepath.Dir(cfgPath), "jwt", "private.pem")
	wantPublic := filepath.Join(filepath.Dir(cfgPath), "jwt", "public.pem")
	wantPrivate, _ = filepath.Abs(wantPrivate)
	wantPublic, _ = filepath.Abs(wantPublic)

	if canonicalPath(t, cfg.Auth.JWT.PrivateKeyFile) != canonicalPath(t, wantPrivate) {
		t.Fatalf("private key path = %q, want %q", cfg.Auth.JWT.PrivateKeyFile, wantPrivate)
	}
	if canonicalPath(t, cfg.Auth.JWT.PublicKeyFile) != canonicalPath(t, wantPublic) {
		t.Fatalf("public key path = %q, want %q", cfg.Auth.JWT.PublicKeyFile, wantPublic)
	}
}

func TestNewConfigAuthDefaultsWithPartialAuthSection(t *testing.T) {
	cfgPath := writeTestConfig(t, `
default_choysum_path: ./.choysum-custom
auth:
  internalKey: dev-internal-key
`)

	cfg, err := NewConfig(cfgPath)
	if err != nil {
		t.Fatalf("NewConfig returned error: %v", err)
	}

	if cfg.Auth == nil {
		t.Fatal("expected auth config to be initialized")
	}
	if !cfg.Auth.Enabled {
		t.Fatal("expected auth.enabled to default to true")
	}
	if cfg.Auth.Type != "jwt" {
		t.Fatalf("auth.type = %q, want jwt", cfg.Auth.Type)
	}
	if cfg.Auth.JWT == nil {
		t.Fatal("expected auth.jwt defaults to be initialized")
	}
}

func TestNewConfigAuthHttpAuthDefaultsWithPartialSection(t *testing.T) {
	cfgPath := writeTestConfig(t, `
default_choysum_path: ./.choysum-custom
auth:
  httpAuth:
    enabled: true
`)

	cfg, err := NewConfig(cfgPath)
	if err != nil {
		t.Fatalf("NewConfig returned error: %v", err)
	}

	if cfg.Auth == nil || cfg.Auth.HttpAuth == nil {
		t.Fatalf("expected auth.httpAuth defaults, got auth=%#v", cfg.Auth)
	}

	defaults := NewDefaultAuthConfig().HttpAuth
	if defaults == nil {
		t.Fatal("expected default auth.httpAuth config")
	}

	httpAuth := cfg.Auth.HttpAuth
	if !httpAuth.Enabled {
		t.Fatal("expected auth.httpAuth.enabled to remain true")
	}
	if !reflect.DeepEqual(httpAuth.ExcludedPaths, defaults.ExcludedPaths) {
		t.Fatalf("unexpected auth.httpAuth.excludedPaths: got %#v want %#v", httpAuth.ExcludedPaths, defaults.ExcludedPaths)
	}
	if !reflect.DeepEqual(httpAuth.TokenExtractors, defaults.TokenExtractors) {
		t.Fatalf("unexpected auth.httpAuth.tokenExtractors: got %#v want %#v", httpAuth.TokenExtractors, defaults.TokenExtractors)
	}
	if httpAuth.ResponseFormat != defaults.ResponseFormat {
		t.Fatalf("unexpected auth.httpAuth.responseFormat: got %q want %q", httpAuth.ResponseFormat, defaults.ResponseFormat)
	}
	if httpAuth.CookieName != defaults.CookieName {
		t.Fatalf("unexpected auth.httpAuth.cookieName: got %q want %q", httpAuth.CookieName, defaults.CookieName)
	}
	if httpAuth.QueryParamName != defaults.QueryParamName {
		t.Fatalf("unexpected auth.httpAuth.queryParamName: got %q want %q", httpAuth.QueryParamName, defaults.QueryParamName)
	}
	if len(httpAuth.ExcludedRegex) != 0 {
		t.Fatalf("unexpected auth.httpAuth.excludedRegex: got %#v", httpAuth.ExcludedRegex)
	}
}

func TestNewConfigAuthHttpAuthExplicitDisablePreserved(t *testing.T) {
	cfgPath := writeTestConfig(t, `
default_choysum_path: ./.choysum-custom
auth:
  httpAuth:
    enabled: false
`)

	cfg, err := NewConfig(cfgPath)
	if err != nil {
		t.Fatalf("NewConfig returned error: %v", err)
	}

	if cfg.Auth == nil || cfg.Auth.HttpAuth == nil {
		t.Fatalf("expected auth.httpAuth defaults, got auth=%#v", cfg.Auth)
	}
	if cfg.Auth.HttpAuth.Enabled {
		t.Fatal("expected explicit auth.httpAuth.enabled=false to be preserved")
	}
	if cfg.Auth.HttpAuth.ResponseFormat != "json" {
		t.Fatalf("expected other auth.httpAuth defaults to be populated, got responseFormat=%q", cfg.Auth.HttpAuth.ResponseFormat)
	}
}

func TestNewConfigAuthJWTDefaultsWithPartialJWTSection(t *testing.T) {
	cfgPath := writeTestConfig(t, `
default_choysum_path: ./.choysum-custom
auth:
  enabled: true
  jwt:
    privateKeyFile: ./jwt/private.pem
    publicKeyFile: ./jwt/public.pem
`)

	cfg, err := NewConfig(cfgPath)
	if err != nil {
		t.Fatalf("NewConfig returned error: %v", err)
	}

	if cfg.Auth == nil || cfg.Auth.JWT == nil {
		t.Fatal("expected auth.jwt config to be initialized")
	}
	if !cfg.Auth.JWT.AutoGenerateKeys {
		t.Fatal("expected auth.jwt.autoGenerateKeys to default to true when omitted")
	}
	if cfg.Auth.JWT.RevokeStore != "database" {
		t.Fatalf("expected auth.jwt.revokeStore default database, got %q", cfg.Auth.JWT.RevokeStore)
	}
	if cfg.Auth.Type != "jwt" {
		t.Fatalf("expected auth.type default jwt, got %q", cfg.Auth.Type)
	}
}

func TestNewConfigNormalizesDefaultChoysumPathAndDerivesTmpPathWhenEmpty(t *testing.T) {
	t.Setenv("CHOYSUM_DEFAULT_CHOYSUM_PATH", "")

	cfgPath := writeTestConfig(t, `
default_choysum_path: ./.choysum-custom
tmp_path: ""
`)

	cfg, err := NewConfig(cfgPath)
	if err != nil {
		t.Fatalf("NewConfig returned error: %v", err)
	}

	wantDefaultChoysumPath := filepath.Join(filepath.Dir(cfgPath), ".choysum-custom")
	wantDefaultChoysumPath, _ = filepath.Abs(wantDefaultChoysumPath)
	if canonicalPath(t, cfg.DefaultChoysumPath) != canonicalPath(t, wantDefaultChoysumPath) {
		t.Fatalf("default_choysum_path = %q, want %q", cfg.DefaultChoysumPath, wantDefaultChoysumPath)
	}

	wantTmpPath := filepath.Join(wantDefaultChoysumPath, "tmp")
	if canonicalPath(t, cfg.TmpPath) != canonicalPath(t, wantTmpPath) {
		t.Fatalf("tmp path = %q, want %q", cfg.TmpPath, wantTmpPath)
	}
}

func TestNewConfigRejectsRootTmpPath(t *testing.T) {
	rootPath := filepath.VolumeName(os.TempDir()) + string(filepath.Separator)
	cfgPath := writeTestConfig(t, "default_choysum_path: ./.choysum-custom\n"+"tmp_path: \""+rootPath+"\"\n")

	_, err := NewConfig(cfgPath)
	if err == nil || !strings.Contains(err.Error(), "tmp_path must be a non-root directory") {
		t.Fatalf("expected non-root tmp_path validation error, got %v", err)
	}
}

func TestDefaultConfigPrefersLocalModulesDirectory(t *testing.T) {
	origWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	defer func() {
		_ = os.Chdir(origWD)
	}()

	workDir := t.TempDir()
	if err := os.Mkdir(filepath.Join(workDir, "modules"), 0o755); err != nil {
		t.Fatalf("mkdir modules: %v", err)
	}
	if err := os.Chdir(workDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	cfg := defaultConfig()
	wantModules, _ := filepath.Abs(filepath.Join(workDir, "modules"))
	wantNpm, _ := filepath.Abs(filepath.Join(workDir, "node_modules"))
	for _, path := range []string{wantNpm} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatalf("mkdir expected path %q: %v", path, err)
		}
	}
	if canonicalPath(t, cfg.ModulesPath) != canonicalPath(t, wantModules) {
		t.Fatalf("modules path = %q, want %q", cfg.ModulesPath, wantModules)
	}
	if strings.TrimSpace(cfg.DistPath) != "" {
		t.Fatalf("expected dist_path empty before path invariants, got %q", cfg.DistPath)
	}
	if canonicalPath(t, cfg.NpmPath) != canonicalPath(t, wantNpm) {
		t.Fatalf("npm path = %q, want %q", cfg.NpmPath, wantNpm)
	}
	if strings.TrimSpace(cfg.DefaultChoysumPath) != "" {
		t.Fatalf("expected default_choysum_path empty by default, got %q", cfg.DefaultChoysumPath)
	}
	if strings.TrimSpace(cfg.TmpPath) != "" {
		t.Fatalf("expected tmp_path empty by default, got %q", cfg.TmpPath)
	}
	if cfg.NPMRegistryURL != DefaultNPMRegistryURL {
		t.Fatalf("expected npm_registry_url default %q, got %q", DefaultNPMRegistryURL, cfg.NPMRegistryURL)
	}
	if cfg.ModuleCatalogIndexURL != DefaultModuleCatalogIndexURL {
		t.Fatalf("expected module_catalog_index_url default %q, got %q", DefaultModuleCatalogIndexURL, cfg.ModuleCatalogIndexURL)
	}
}

func TestDefaultConfigUsesRelativeModulesWhenLocalDirMissing(t *testing.T) {
	origWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	defer func() {
		_ = os.Chdir(origWD)
	}()

	workDir := t.TempDir()
	if err := os.Chdir(workDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	cfg := defaultConfig()
	wantModules, _ := filepath.Abs(filepath.Join(workDir, "modules"))
	wantNpm, _ := filepath.Abs(filepath.Join(workDir, "node_modules"))
	for _, path := range []string{wantModules, wantNpm} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatalf("mkdir expected path %q: %v", path, err)
		}
	}
	if canonicalPath(t, cfg.ModulesPath) != canonicalPath(t, wantModules) {
		t.Fatalf("modules path = %q, want %q", cfg.ModulesPath, wantModules)
	}
	if strings.TrimSpace(cfg.DistPath) != "" {
		t.Fatalf("expected dist_path empty before path invariants, got %q", cfg.DistPath)
	}
	if canonicalPath(t, cfg.NpmPath) != canonicalPath(t, wantNpm) {
		t.Fatalf("npm path = %q, want %q", cfg.NpmPath, wantNpm)
	}
	if strings.TrimSpace(cfg.DefaultChoysumPath) != "" {
		t.Fatalf("expected default_choysum_path empty by default, got %q", cfg.DefaultChoysumPath)
	}
	if strings.TrimSpace(cfg.TmpPath) != "" {
		t.Fatalf("expected tmp_path empty by default, got %q", cfg.TmpPath)
	}
	if cfg.NPMRegistryURL != DefaultNPMRegistryURL {
		t.Fatalf("expected npm_registry_url default %q, got %q", DefaultNPMRegistryURL, cfg.NPMRegistryURL)
	}
	if cfg.ModuleCatalogIndexURL != DefaultModuleCatalogIndexURL {
		t.Fatalf("expected module_catalog_index_url default %q, got %q", DefaultModuleCatalogIndexURL, cfg.ModuleCatalogIndexURL)
	}
	if cfg.Log == nil || cfg.Db == nil || cfg.Compile == nil || cfg.Server == nil || cfg.Task == nil {
		t.Fatalf("expected nested defaults to be initialized: %#v", cfg)
	}
	if cfg.Auth != nil {
		t.Fatalf("expected auth to remain nil before config load path invariants, got %#v", cfg.Auth)
	}
	if cfg.FrontendEnv == nil || cfg.BackendEnv == nil {
		t.Fatalf("expected env maps to be initialized: frontend=%#v backend=%#v", cfg.FrontendEnv, cfg.BackendEnv)
	}
}

func TestNewConfigDerivesDistPathFromDefaultChoysumPathWhenOmitted(t *testing.T) {
	t.Setenv("CHOYSUM_DEFAULT_CHOYSUM_PATH", "")

	cfgPath := writeTestConfig(t, `
default_choysum_path: ./.choysum-custom
`)

	cfg, err := NewConfig(cfgPath)
	if err != nil {
		t.Fatalf("NewConfig returned error: %v", err)
	}

	wantDistPath := filepath.Join(filepath.Dir(cfgPath), ".choysum-custom", "dist")
	wantDistPath, _ = filepath.Abs(wantDistPath)
	if canonicalPath(t, cfg.DistPath) != canonicalPath(t, wantDistPath) {
		t.Fatalf("dist_path = %q, want %q", cfg.DistPath, wantDistPath)
	}
}

func TestNewDefaultFrontendEnv(t *testing.T) {
	t.Run("nil config falls back to root development env", func(t *testing.T) {
		env := NewDefaultFrontendEnv(nil)
		if env["BASE_URL"] != "/" {
			t.Fatalf("BASE_URL = %#v, want /", env["BASE_URL"])
		}
		if env["MODE"] != "development" || env["PROD"] != false || env["DEV"] != true {
			t.Fatalf("unexpected nil-config mode flags: %#v", env)
		}
	})

	t.Run("development mode defaults", func(t *testing.T) {
		cfg := &Config{
			Compile: &CompileConfig{Production: false},
			Server:  &ServerConfig{WebBaseURL: "/portal/"},
		}
		env := NewDefaultFrontendEnv(cfg)
		if env["BASE_URL"] != "/portal/" {
			t.Fatalf("BASE_URL = %#v, want /portal/", env["BASE_URL"])
		}
		if env["MODE"] != "development" || env["PROD"] != false || env["DEV"] != true {
			t.Fatalf("unexpected dev mode flags: %#v", env)
		}
	})

	t.Run("production mode trims trailing slash", func(t *testing.T) {
		cfg := &Config{
			Compile: &CompileConfig{Production: true},
			Server:  &ServerConfig{WebBaseURL: "/web"},
		}
		env := NewDefaultFrontendEnv(cfg)
		if env["BASE_URL"] != "/web/" {
			t.Fatalf("BASE_URL = %#v, want /web/", env["BASE_URL"])
		}
		if env["MODE"] != "production" || env["PROD"] != true || env["DEV"] != false {
			t.Fatalf("unexpected prod mode flags: %#v", env)
		}
	})
}

func TestConfigUnmarshalEnvOverrideAndHookErrors(t *testing.T) {
	cfgPath := writeTestConfig(t, `
default_choysum_path: ./.choysum-custom
modules_path: from-config
compile:
  production: false
server:
  webBaseUrl: /console/
`)
	envOnlyCfgPath := writeTestConfig(t, `
default_choysum_path: ./.choysum-custom
modules_path: from-config
`)

	t.Run("custom env prefix overrides config values", func(t *testing.T) {
		envModules := filepath.Join(t.TempDir(), "env-modules")
		envNPMRegistryURL := "https://registry.npmmirror.com"
		envModuleCatalogIndexURL := "https://index.example.dev/v1/index.json"
		t.Setenv("CHOYSUM_TEST_MODULES_PATH", envModules)
		t.Setenv("CHOYSUM_TEST_NPM_REGISTRY_URL", envNPMRegistryURL)
		t.Setenv("CHOYSUM_TEST_MODULE_CATALOG_INDEX_URL", envModuleCatalogIndexURL)

		cfg := defaultConfig()
		if err := cfg.unmarshal(cfgPath, WithEnvPrefix("CHOYSUM_TEST")); err != nil {
			t.Fatalf("unmarshal() error = %v", err)
		}
		if canonicalPath(t, cfg.ModulesPath) != canonicalPath(t, envModules) {
			t.Fatalf("modules path = %q, want env override %q", cfg.ModulesPath, envModules)
		}
		if cfg.NPMRegistryURL != envNPMRegistryURL {
			t.Fatalf("npm_registry_url = %q, want env override %q", cfg.NPMRegistryURL, envNPMRegistryURL)
		}
		if cfg.ModuleCatalogIndexURL != envModuleCatalogIndexURL {
			t.Fatalf("module_catalog_index_url = %q, want env override %q", cfg.ModuleCatalogIndexURL, envModuleCatalogIndexURL)
		}
	})

	t.Run("invalid module catalog index url fails validation", func(t *testing.T) {
		cfg := defaultConfig()
		err := cfg.unmarshal(cfgPath, WithDefaults(func(cfg *Config) {
			cfg.ModuleCatalogIndexURL = "https://index.example.dev/v1/catalog.json"
		}))
		if err == nil || !strings.Contains(err.Error(), "module_catalog_index_url") {
			t.Fatalf("expected invalid module_catalog_index_url validation error, got %v", err)
		}
	})

	t.Run("custom env prefix overrides server factory selectors", func(t *testing.T) {
		t.Setenv("CHOYSUM_TEST_SERVER_JS_ENGINE_FACTORY", "engine-from-env")
		t.Setenv("CHOYSUM_TEST_SERVER_JS_EXECUTOR_FACTORY", "executor-from-env")

		cfg := defaultConfig()
		if err := cfg.unmarshal(cfgPath, WithEnvPrefix("CHOYSUM_TEST")); err != nil {
			t.Fatalf("unmarshal() error = %v", err)
		}
		if cfg.Server == nil {
			t.Fatalf("expected server config after unmarshal, got %#v", cfg)
		}
		if cfg.Server.JsEngineFactory != "engine-from-env" {
			t.Fatalf("server.jsEngineFactory = %q, want engine-from-env", cfg.Server.JsEngineFactory)
		}
		if cfg.Server.JsExecutorFactory != "executor-from-env" {
			t.Fatalf("server.jsExecutorFactory = %q, want executor-from-env", cfg.Server.JsExecutorFactory)
		}
	})

	t.Run("custom env prefix overrides jwt identity cache with new env names", func(t *testing.T) {
		t.Setenv("CHOYSUM_TEST_AUTH_JWT_IDENTITY_CACHE_ENABLED", "false")
		t.Setenv("CHOYSUM_TEST_AUTH_JWT_IDENTITY_CACHE_BACKEND", "memory")
		t.Setenv("CHOYSUM_TEST_AUTH_JWT_IDENTITY_CACHE_SIZE", "321")
		t.Setenv("CHOYSUM_TEST_AUTH_JWT_IDENTITY_CACHE_TTL", "90s")

		cfg := defaultConfig()
		if err := cfg.unmarshal(cfgPath, WithEnvPrefix("CHOYSUM_TEST")); err != nil {
			t.Fatalf("unmarshal() error = %v", err)
		}
		if cfg.Auth == nil || cfg.Auth.JWT == nil || cfg.Auth.JWT.IdentityCache == nil {
			t.Fatalf("expected jwt identity cache config, got %#v", cfg.Auth)
		}
		if cfg.Auth.JWT.IdentityCache.Enabled {
			t.Fatal("expected new env name to disable jwt identity cache")
		}
		if cfg.Auth.JWT.IdentityCache.Backend != "memory" {
			t.Fatalf("identity cache backend = %q, want memory", cfg.Auth.JWT.IdentityCache.Backend)
		}
		if cfg.Auth.JWT.IdentityCache.Size != 321 {
			t.Fatalf("identity cache size = %d, want 321", cfg.Auth.JWT.IdentityCache.Size)
		}
		if cfg.Auth.JWT.IdentityCache.TTL != 90*time.Second {
			t.Fatalf("identity cache ttl = %v, want 90s", cfg.Auth.JWT.IdentityCache.TTL)
		}
	})

	t.Run("custom env prefix populates sensitive nested keys without config sections", func(t *testing.T) {
		t.Setenv("CHOYSUM_TEST_DB_DIALECT", "postgres")
		t.Setenv("CHOYSUM_TEST_DB_DSN", "postgres://env-only")
		t.Setenv("CHOYSUM_TEST_AUTH_INTERNAL_KEY", "env-internal-key")

		cfg := defaultConfig()
		if err := cfg.unmarshal(envOnlyCfgPath, WithEnvPrefix("CHOYSUM_TEST")); err != nil {
			t.Fatalf("unmarshal() error = %v", err)
		}
		if cfg.Db == nil {
			t.Fatalf("expected db config after unmarshal, got %#v", cfg)
		}
		if cfg.Db.Dialect != "postgres" {
			t.Fatalf("db.dialect = %q, want postgres", cfg.Db.Dialect)
		}
		if cfg.Db.DSN != "postgres://env-only" {
			t.Fatalf("db.dsn = %q, want env override", cfg.Db.DSN)
		}
		if cfg.Auth == nil {
			t.Fatalf("expected auth config after unmarshal, got %#v", cfg)
		}
		if cfg.Auth.InternalKey != "env-internal-key" {
			t.Fatalf("auth.internalKey = %q, want env override", cfg.Auth.InternalKey)
		}
	})

	t.Run("custom env prefix ignores removed jwt identity cache aliases", func(t *testing.T) {
		t.Setenv("CHOYSUM_TEST_AUTH_JWT_IDENTITYCACHE_ENABLED", "false")
		t.Setenv("CHOYSUM_TEST_AUTH_JWT_IDENTITYCACHE_SIZE", "321")
		t.Setenv("CHOYSUM_TEST_AUTH_JWT_IDENTITYCACHE_TTL", "90s")
		t.Setenv("CHOYSUM_TEST_AUTH_JWT_CACHE_ENABLED", "false")
		t.Setenv("CHOYSUM_TEST_AUTH_JWT_CACHE_SIZE", "654")
		t.Setenv("CHOYSUM_TEST_AUTH_JWT_CACHE_TTL", "45s")

		cfg := defaultConfig()
		if err := cfg.unmarshal(cfgPath, WithEnvPrefix("CHOYSUM_TEST")); err != nil {
			t.Fatalf("unmarshal() error = %v", err)
		}
		if cfg.Auth == nil || cfg.Auth.JWT == nil || cfg.Auth.JWT.IdentityCache == nil {
			t.Fatalf("expected jwt identity cache config, got %#v", cfg.Auth)
		}
		if !cfg.Auth.JWT.IdentityCache.Enabled {
			t.Fatal("expected removed jwt identity cache env aliases to be ignored")
		}
		if cfg.Auth.JWT.IdentityCache.Backend != "memory" {
			t.Fatalf("identity cache backend = %q, want default memory", cfg.Auth.JWT.IdentityCache.Backend)
		}
		if cfg.Auth.JWT.IdentityCache.Size != 10000 {
			t.Fatalf("identity cache size = %d, want default 10000", cfg.Auth.JWT.IdentityCache.Size)
		}
		if cfg.Auth.JWT.IdentityCache.TTL != 5*time.Minute {
			t.Fatalf("identity cache ttl = %v, want default 5m", cfg.Auth.JWT.IdentityCache.TTL)
		}
	})

	t.Run("pre hook errors propagate", func(t *testing.T) {
		cfg := defaultConfig()
		err := cfg.unmarshal(cfgPath, Option{pre: func(*viper.Viper, *Config) error {
			return errors.New("pre failed")
		}})
		if err == nil {
			t.Fatalf("expected pre hook error, got %v", err)
		}
		if !IsLoadStage(err, LoadStageDecode) {
			t.Fatalf("expected decode stage error, got %v", err)
		}
		if !strings.Contains(err.Error(), "pre failed") {
			t.Fatalf("expected pre hook message, got %v", err)
		}
	})

	t.Run("post hook errors propagate", func(t *testing.T) {
		cfg := defaultConfig()
		err := cfg.unmarshal(cfgPath, AfterUnmarshal(func(*viper.Viper, *Config) error {
			return errors.New("post failed")
		}))
		if err == nil {
			t.Fatalf("expected post hook error, got %v", err)
		}
		if !IsLoadStage(err, LoadStageApply) {
			t.Fatalf("expected apply stage error, got %v", err)
		}
		if !strings.Contains(err.Error(), "post failed") {
			t.Fatalf("expected post hook message, got %v", err)
		}
	})
}

func TestDefaultConfigPathUsesLocalAndFallsBackWhenMissing(t *testing.T) {
	origWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	defer func() {
		_ = os.Chdir(origWD)
	}()

	workDir := t.TempDir()
	if err := os.Chdir(workDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	localConfig := filepath.Join(workDir, "config.yaml")
	if err := os.WriteFile(localConfig, []byte("log: {}\n"), 0o644); err != nil {
		t.Fatalf("write config %q: %v", localConfig, err)
	}

	got, err := DefaultConfigPath()
	if err != nil {
		t.Fatalf("DefaultConfigPath returned error: %v", err)
	}
	wantLocal, _ := filepath.Abs(localConfig)
	if canonicalPath(t, got) != canonicalPath(t, wantLocal) {
		t.Fatalf("config path = %q, want local %q", got, wantLocal)
	}

	if err := os.Remove(localConfig); err != nil {
		t.Fatalf("remove local config: %v", err)
	}
	got, err = DefaultConfigPath()
	if err != nil {
		t.Fatalf("DefaultConfigPath missing fallback returned error: %v", err)
	}
	if got != "" {
		t.Fatalf("expected empty config path fallback, got %q", got)
	}
}

func TestValidateModuleCatalogIndexURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		raw         string
		wantErrPart string
	}{
		{name: "empty", raw: "   ", wantErrPart: "module_catalog_index_url is required"},
		{name: "invalid url", raw: "https://%zz", wantErrPart: "invalid module_catalog_index_url"},
		{name: "unsupported scheme", raw: "ftp://index.example.dev/v1/index.json", wantErrPart: "only http/https are supported"},
		{name: "missing host", raw: "https:///v1/index.json", wantErrPart: "host is required"},
		{name: "missing index json", raw: "https://index.example.dev/v1/catalog.json", wantErrPart: "must point to an index.json resource"},
		{name: "valid https", raw: "https://index.example.dev/v1/index.json", wantErrPart: ""},
		{name: "valid http", raw: "http://index.example.dev/v1/index.json", wantErrPart: ""},
		{name: "valid upper index path", raw: "https://index.example.dev/v1/INDEX.JSON", wantErrPart: ""},
		{name: "valid with query", raw: "https://index.example.dev/v1/index.json?cache=1", wantErrPart: ""},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateModuleCatalogIndexURL(tt.raw)
			if tt.wantErrPart == "" {
				if err != nil {
					t.Fatalf("ValidateModuleCatalogIndexURL(%q) error = %v", tt.raw, err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErrPart) {
				t.Fatalf("ValidateModuleCatalogIndexURL(%q) error = %v, want substring %q", tt.raw, err, tt.wantErrPart)
			}
		})
	}
}

func TestLegacyRegistryEntryKeysHelpers(t *testing.T) {
	t.Parallel()

	if got := legacyRegistryEntryKeys(map[string]any{
		" url ":     "https://legacy.example.dev",
		"INDEXURL":  "https://legacy.example.dev/v1/index.json",
		"index_url": "https://legacy.example.dev/v1/index.json",
	}); !reflect.DeepEqual(got, []string{"indexURL", "index_url", "url"}) {
		t.Fatalf("legacyRegistryEntryKeys(map[string]any) = %#v, want %#v", got, []string{"indexURL", "index_url", "url"})
	}

	if got := legacyRegistryEntryKeys(map[any]any{
		" url ":     "https://legacy.example.dev",
		"INDEXURL":  "https://legacy.example.dev/v1/index.json",
		"index_url": "https://legacy.example.dev/v1/index.json",
	}); !reflect.DeepEqual(got, []string{"indexURL", "index_url", "url"}) {
		t.Fatalf("legacyRegistryEntryKeys(map[any]any) = %#v, want %#v", got, []string{"indexURL", "index_url", "url"})
	}

	if got := legacyRegistryEntryKeys("not-a-map"); got != nil {
		t.Fatalf("legacyRegistryEntryKeys(non-map) = %#v, want nil", got)
	}

	if got := collectLegacyRegistryEntryKeys(map[string]any{"name": "official"}); len(got) != 0 {
		t.Fatalf("collectLegacyRegistryEntryKeys(no legacy keys) = %#v, want empty", got)
	}
}

func TestRejectLegacyModuleCatalogConfigKeys(t *testing.T) {
	t.Parallel()

	if err := rejectLegacyModuleCatalogConfigKeys(nil); err != nil {
		t.Fatalf("rejectLegacyModuleCatalogConfigKeys(nil) error = %v", err)
	}

	loadViper := func(t *testing.T, body string) *viper.Viper {
		t.Helper()
		v := viper.New()
		v.SetConfigType("yaml")
		if err := v.ReadConfig(strings.NewReader(strings.TrimSpace(body) + "\n")); err != nil {
			t.Fatalf("ReadConfig() error = %v", err)
		}
		return v
	}

	t.Run("accepts current index-url config", func(t *testing.T) {
		v := loadViper(t, `
module_catalog_index_url: https://index.choysum.dev/v1/index.json
`)
		if err := rejectLegacyModuleCatalogConfigKeys(v); err != nil {
			t.Fatalf("rejectLegacyModuleCatalogConfigKeys(current) error = %v", err)
		}
	})

	t.Run("rejects root legacy key", func(t *testing.T) {
		v := loadViper(t, `
registry_index_url: https://legacy.example.dev/v1/index.json
`)
		err := rejectLegacyModuleCatalogConfigKeys(v)
		if err == nil || !strings.Contains(err.Error(), "registry_index_url (use module_catalog_index_url)") {
			t.Fatalf("rejectLegacyModuleCatalogConfigKeys(root legacy) error = %v", err)
		}
	})

	t.Run("rejects nested legacy registry entry keys", func(t *testing.T) {
		v := loadViper(t, `
registries:
  official:
    url: https://legacy.example.dev/v1/index.json
    indexURL: https://legacy.example.dev/v1/index.json
  community:
    index_url: https://legacy.example.dev/v1/index.json
`)
		err := rejectLegacyModuleCatalogConfigKeys(v)
		if err == nil {
			t.Fatal("expected nested legacy keys to be rejected")
		}
		for _, want := range []string{
			"registries (use module_catalog_index_url)",
			"registries.official.url (use module_catalog_index_url)",
			"registries.official.indexURL (use module_catalog_index_url)",
			"registries.community.index_url (use module_catalog_index_url)",
		} {
			if !strings.Contains(err.Error(), want) {
				t.Fatalf("rejectLegacyModuleCatalogConfigKeys(nested legacy) error = %v, missing %q", err, want)
			}
		}
	})
}

// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cmd

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	clicompat "github.com/choysum-dev/choysum/internal/cli/compat"
	cliruntime "github.com/choysum-dev/choysum/internal/cli/runtime"
	"github.com/choysum-dev/choysum/internal/config/snapshot"
	modmeta "github.com/choysum-dev/choysum/internal/module/meta"
	internalorigin "github.com/choysum-dev/choysum/internal/module/origin"
	sourceregistry "github.com/choysum-dev/choysum/internal/module/origin/registry"
	"github.com/choysum-dev/choysum/internal/testing/scopetest"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/spf13/cobra"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func executeCommandForTest(t *testing.T, cmd *cobra.Command, args ...string) (string, error) {
	t.Helper()

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

func TestNewModuleCmd_SubcommandsAndWorkflow(t *testing.T) {
	workspaceRoot := t.TempDir()
	modulesPath := filepath.Join(workspaceRoot, "modules")
	if err := os.MkdirAll(modulesPath, 0o755); err != nil {
		t.Fatalf("create modules path: %v", err)
	}

	writeCommandPackage(t, modulesPath, "auth", `{
		"name": "@choysum-dev/auth",
		"version": "1.2.3",
		"description": "test module",
		"license": "Apache-2.0",
		"author": "test",
		"type": "module",
		"main": "index.ts",
		"choysum": {
			"moduleName": "auth",
			"application": "auth",
			"category": "test",
			"depends": [],
			"entryPoints": {"service": "./service/index.ts", "web": "./web/index.ts"}
		}
	}`)

	defaultChoysumPath := t.TempDir()
	cfg := &config.Config{
		ModulesPath:        modulesPath,
		TmpPath:            filepath.Join(defaultChoysumPath, "tmp"),
		ConfigPath:         filepath.Join(workspaceRoot, "config.yaml"),
		DefaultChoysumPath: defaultChoysumPath,
	}
	envGetter := func() scope.Scope {
		return &commandTestScope{ctx: context.Background(), cfg: cfg}
	}
	runtimeOptionsGetter := func() cliruntime.Options {
		options := cliruntime.NewScopeInputConfigOptions(snapshot.New(cfg))
		return cliruntime.Options{
			DefaultChoysumPath:    options.DefaultChoysumPath,
			ModulesPath:           options.ModulesPath,
			TmpPath:               options.TmpPath,
			ModuleCatalogIndexURL: strings.TrimSpace(options.ModuleCatalogIndexURL),
		}
	}

	moduleCmd := newModuleCmd(envGetter, runtimeOptionsGetter)
	output, err := executeCommandForTest(t, moduleCmd, "fetch", "auth")
	if err != nil {
		t.Fatalf("module fetch failed: %v", err)
	}
	if !strings.Contains(output, "Fetched module auth@v1.2.3") {
		t.Fatalf("unexpected fetch output: %q", output)
	}

	output, err = executeCommandForTest(t, moduleCmd, "info", "auth")
	if err != nil {
		t.Fatalf("module info failed: %v", err)
	}
	if !strings.Contains(output, `"module_name": "auth"`) || !strings.Contains(output, `"version": "v1.2.3"`) {
		t.Fatalf("unexpected info output: %q", output)
	}

	output, err = executeCommandForTest(t, moduleCmd, "list")
	if err != nil {
		t.Fatalf("module list failed: %v", err)
	}
	if !strings.Contains(output, "auth") || !strings.Contains(output, "local") {
		t.Fatalf("unexpected list output: %q", output)
	}

	output, err = executeCommandForTest(t, moduleCmd, "search", "au")
	if err != nil {
		t.Fatalf("module search failed: %v", err)
	}
	if !strings.Contains(output, "auth") {
		t.Fatalf("expected module search output to include auth, got %q", output)
	}

	if _, err := executeCommandForTest(t, moduleCmd, "purge", "auth"); err != nil {
		t.Fatalf("module purge failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(modulesPath, "auth")); !os.IsNotExist(err) {
		t.Fatalf("expected purged module directory to be removed, stat err=%v", err)
	}

	store := internalorigin.NewLockStore(internalorigin.WithLockStoreDefaultChoysumPath(cfg.DefaultChoysumPath))
	lock, err := store.Read(workspaceRoot)
	if err != nil {
		t.Fatalf("read lock file after purge: %v", err)
	}
	if _, ok := lock.Modules["auth"]; ok {
		t.Fatal("expected auth binding to be removed after purge")
	}
}

func TestNewModuleCmd_RequiresRuntimeScope(t *testing.T) {
	moduleCmd := newModuleCmd(func() scope.Scope { return nil }, func() cliruntime.Options { return cliruntime.Options{} })
	if _, err := executeCommandForTest(t, moduleCmd, "fetch", "auth"); err == nil || !strings.Contains(err.Error(), "scope is not initialized") {
		t.Fatalf("expected environment initialization error, got %v", err)
	}
}

func TestNewModuleCmd_RuntimeOptionsPrefix(t *testing.T) {
	modulesPath := t.TempDir()
	scopeGetter := func() scope.Scope {
		return &commandTestScope{ctx: context.Background(), cfg: newCommandTestConfig(modulesPath)}
	}
	moduleCmd := newModuleCmd(scopeGetter, func() cliruntime.Options { return cliruntime.Options{} })

	if _, err := executeCommandForTest(t, moduleCmd, "search", "auth"); err == nil || !strings.Contains(err.Error(), "module: invalid runtime options") {
		t.Fatalf("expected prefixed runtime options error, got %v", err)
	}
}

func TestModuleUtilityHelpers(t *testing.T) {
	defaultURL, err := resolveModuleCatalogIndexURL(cliruntime.Options{})
	if err != nil {
		t.Fatalf("resolveModuleCatalogIndexURL(default) error = %v", err)
	}
	if defaultURL != config.DefaultModuleCatalogIndexURL {
		t.Fatalf("resolveModuleCatalogIndexURL(default) = %q, want %q", defaultURL, config.DefaultModuleCatalogIndexURL)
	}

	customURL, err := resolveModuleCatalogIndexURL(cliruntime.Options{ModuleCatalogIndexURL: " https://index.acme.dev/v1/index.json "})
	if err != nil {
		t.Fatalf("resolveModuleCatalogIndexURL(custom) error = %v", err)
	}
	if customURL != "https://index.acme.dev/v1/index.json" {
		t.Fatalf("resolveModuleCatalogIndexURL(custom) = %q, want %q", customURL, "https://index.acme.dev/v1/index.json")
	}

	if _, err := resolveModuleCatalogIndexURL(cliruntime.Options{ModuleCatalogIndexURL: "https://index.acme.dev/v1/catalog.json"}); err == nil || !strings.Contains(err.Error(), "index.json") {
		t.Fatalf("expected invalid module_catalog_index_url error, got %v", err)
	}

	if ctx := contextFromCommand(nil); ctx == nil {
		t.Fatal("contextFromCommand(nil) returned nil context")
	}
	cmd := &cobra.Command{}
	ctxKey := struct{}{}
	wantCtx := context.WithValue(context.Background(), ctxKey, "module")
	cmd.SetContext(wantCtx)
	if gotCtx := contextFromCommand(cmd); gotCtx != wantCtx {
		t.Fatalf("contextFromCommand(command) did not return command context")
	}

	if got := nullStringValue(sql.NullString{}); got != "" {
		t.Fatalf("nullStringValue(invalid) = %q, want empty", got)
	}
	if got := nullStringValue(sql.NullString{String: "  v1.2.3  ", Valid: true}); got != "v1.2.3" {
		t.Fatalf("nullStringValue(valid) = %q, want %q", got, "v1.2.3")
	}

	payload := moduleIndexViewToPayload(moduleIndexView{
		ModuleName:    " auth ",
		OriginType:    " registry ",
		OriginRef:     " auth@v1.2.3 ",
		Available:     true,
		Version:       sql.NullString{String: " v1.2.3 ", Valid: true},
		LocalPath:     sql.NullString{String: " /tmp/auth ", Valid: true},
		InstallStatus: sql.NullString{},
	})
	if payload["moduleName"] != "auth" || payload["originType"] != "registry" || payload["originRef"] != "auth@v1.2.3" {
		t.Fatalf("unexpected payload identity fields: %#v", payload)
	}
	if payload["version"] != "v1.2.3" || payload["localPath"] != "/tmp/auth" || payload["installStatus"] != "" {
		t.Fatalf("unexpected payload optional fields: %#v", payload)
	}
	if available, ok := payload["available"].(bool); !ok || !available {
		t.Fatalf("unexpected payload availability: %#v", payload["available"])
	}
}

func TestFilterCatalogModuleByCompatibility_SourceConsistency(t *testing.T) {
	item := &sourceregistry.CatalogModule{
		Name:          "demo",
		LatestVersion: "v2.0.0",
		Versions:      []string{"v1.0.0", "v2.0.0"},
		VersionCLIRanges: map[string]string{
			"v1.0.0": ">=1.0.0 <2.0.0",
			"v2.0.0": ">=2.0.0 <3.0.0",
		},
		Source: &sourceregistry.CatalogSource{
			Type:      "npm",
			Package:   "@choysum-dev/demo",
			Version:   "v2.0.0",
			Tarball:   "https://registry.npmjs.org/@choysum-dev/demo/-/demo-2.0.0.tgz",
			Integrity: "sha512-demo",
		},
	}

	t.Run("drops source when filtered latest changes", func(t *testing.T) {
		filtered, err := clicompat.FilterCatalogModuleByCompatibility(item, "v1.5.0")
		if err != nil {
			t.Fatalf("filterCatalogModuleByCompatibility() error = %v", err)
		}
		if filtered.LatestVersion != "v1.0.0" {
			t.Fatalf("filtered latest version = %q, want %q", filtered.LatestVersion, "v1.0.0")
		}
		if filtered.Source != nil {
			t.Fatalf("filtered source = %#v, want nil when latest version changes", filtered.Source)
		}
	})

	t.Run("keeps source when filtered latest unchanged", func(t *testing.T) {
		filtered, err := clicompat.FilterCatalogModuleByCompatibility(item, "v2.1.0")
		if err != nil {
			t.Fatalf("filterCatalogModuleByCompatibility() error = %v", err)
		}
		if filtered.Source == nil {
			t.Fatal("filtered source is nil, want source metadata")
		}
		if filtered.Source.Version != "v2.0.0" {
			t.Fatalf("filtered source version = %q, want %q", filtered.Source.Version, "v2.0.0")
		}
		if filtered.Source.Tarball != item.Source.Tarball || filtered.Source.Integrity != item.Source.Integrity {
			t.Fatalf("filtered source payload changed unexpectedly: %#v", filtered.Source)
		}
	})
}

func TestModuleRemoteCommandBranches(t *testing.T) {
	runtimeScope := &commandTestScope{ctx: context.Background(), cfg: &config.Config{}}

	catalogServer := startRemoteRegistryCatalogServer(t, []remoteCatalogModule{
		{Name: "auth", LatestVersion: "v1.2.3", Description: "Authentication", Versions: []string{"v1.0.0", "v1.2.3"}},
	})
	defer catalogServer.Close()

	runtimeOptions := cliruntime.Options{ModuleCatalogIndexURL: catalogServer.URL + "/v1/index.json"}

	t.Run("search validates query", func(t *testing.T) {
		cmd := &cobra.Command{}
		if err := runModuleSearchRemote(cmd, runtimeScope, runtimeOptions, "   "); err == nil || !strings.Contains(err.Error(), "query is required") {
			t.Fatalf("expected query required error, got %v", err)
		}
	})

	t.Run("search wraps remote catalog query errors", func(t *testing.T) {
		failingCatalog := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "upstream unavailable", http.StatusBadGateway)
		}))
		defer failingCatalog.Close()

		cmd := &cobra.Command{}
		err := runModuleSearchRemote(cmd, runtimeScope, cliruntime.Options{ModuleCatalogIndexURL: failingCatalog.URL + "/v1/index.json"}, "auth")
		if err == nil {
			t.Fatal("expected remote catalog query error")
		}
		if !strings.Contains(err.Error(), "query remote module catalog index") {
			t.Fatalf("expected wrapped remote catalog query error, got %v", err)
		}
	})

	t.Run("search renders result table", func(t *testing.T) {
		var out bytes.Buffer
		cmd := &cobra.Command{}
		cmd.SetOut(&out)

		if err := runModuleSearchRemote(cmd, runtimeScope, runtimeOptions, "auth"); err != nil {
			t.Fatalf("runModuleSearchRemote() error = %v", err)
		}
		output := out.String()
		if !strings.Contains(output, "MODULE") || !strings.Contains(output, "auth") || !strings.Contains(output, "v1.2.3") {
			t.Fatalf("unexpected remote search output: %q", output)
		}
	})

	t.Run("search reports empty result", func(t *testing.T) {
		var out bytes.Buffer
		cmd := &cobra.Command{}
		cmd.SetOut(&out)

		if err := runModuleSearchRemote(cmd, runtimeScope, runtimeOptions, "missing"); err != nil {
			t.Fatalf("runModuleSearchRemote(missing) error = %v", err)
		}
		if !strings.Contains(out.String(), `No remote modules matched query "missing"`) {
			t.Fatalf("unexpected missing-search output: %q", out.String())
		}
	})

	t.Run("list renders and handles empty catalog", func(t *testing.T) {
		var listOut bytes.Buffer
		listCmd := &cobra.Command{}
		listCmd.SetOut(&listOut)

		if err := runModuleListRemote(listCmd, runtimeScope, runtimeOptions, "v0.0.0-0", false); err != nil {
			t.Fatalf("runModuleListRemote() error = %v", err)
		}
		if !strings.Contains(listOut.String(), "MODULE") || !strings.Contains(listOut.String(), "auth") {
			t.Fatalf("unexpected remote list output: %q", listOut.String())
		}

		emptyServer := startRemoteRegistryCatalogServer(t, nil)
		defer emptyServer.Close()
		emptyOptions := cliruntime.Options{ModuleCatalogIndexURL: emptyServer.URL + "/v1/index.json"}

		var emptyOut bytes.Buffer
		emptyCmd := &cobra.Command{}
		emptyCmd.SetOut(&emptyOut)
		if err := runModuleListRemote(emptyCmd, runtimeScope, emptyOptions, "v0.0.0-0", false); err != nil {
			t.Fatalf("runModuleListRemote(empty) error = %v", err)
		}
		if !strings.Contains(emptyOut.String(), "No remote modules found.") {
			t.Fatalf("unexpected empty remote list output: %q", emptyOut.String())
		}
	})

	t.Run("list filters to compatible version", func(t *testing.T) {
		compatibleCatalog := startRemoteRegistryCatalogServer(t, []remoteCatalogModule{
			{
				Name:          "auth",
				LatestVersion: "v1.2.3",
				Description:   "Authentication",
				Versions:      []string{"v0.9.0", "v1.2.3"},
				VersionCLIRanges: map[string]string{
					"v0.9.0": ">=0.0.0-0 <0.0.0",
					"v1.2.3": ">=1.0.0 <2.0.0",
				},
			},
			{
				Name:          "billing",
				LatestVersion: "v1.0.0",
				Description:   "Billing",
				Versions:      []string{"v1.0.0"},
				VersionCLIRanges: map[string]string{
					"v1.0.0": ">=1.0.0 <2.0.0",
				},
			},
		})
		defer compatibleCatalog.Close()

		var out bytes.Buffer
		listCmd := &cobra.Command{}
		listCmd.SetOut(&out)

		err := runModuleListRemote(listCmd, runtimeScope, cliruntime.Options{ModuleCatalogIndexURL: compatibleCatalog.URL + "/v1/index.json"}, "v0.0.0-0", false)
		if err != nil {
			t.Fatalf("runModuleListRemote(compat-filtered) error = %v", err)
		}
		output := out.String()
		if !strings.Contains(output, "auth") {
			t.Fatalf("expected auth in filtered list output, got %q", output)
		}
		if !strings.Contains(output, "v0.9.0") {
			t.Fatalf("expected compatible version in filtered list output, got %q", output)
		}
		if strings.Contains(output, "billing") {
			t.Fatalf("did not expect incompatible module in filtered list output, got %q", output)
		}
	})

	t.Run("info validates input and returns payload", func(t *testing.T) {
		cmd := &cobra.Command{}
		if err := runModuleInfoRemote(cmd, runtimeScope, runtimeOptions, "   ", "", false); err == nil || !strings.Contains(err.Error(), "module input is required") {
			t.Fatalf("expected module input required error, got %v", err)
		}

		if err := runModuleInfoRemote(cmd, runtimeScope, runtimeOptions, "legacy/auth", "", false); err == nil || !strings.Contains(err.Error(), "registry alias syntax is no longer supported") {
			t.Fatalf("expected legacy alias syntax error, got %v", err)
		}

		for _, input := range []string{"auth", "auth@latest"} {
			var out bytes.Buffer
			infoCmd := &cobra.Command{}
			infoCmd.SetOut(&out)

			if err := runModuleInfoRemote(infoCmd, runtimeScope, runtimeOptions, input, "v0.0.0-0", false); err != nil {
				t.Fatalf("runModuleInfoRemote(%s) error = %v", input, err)
			}
			payload := map[string]any{}
			if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
				t.Fatalf("unmarshal remote info payload for %s: %v (payload=%q)", input, err, out.String())
			}
			if payload["name"] != "auth" || payload["latestVersion"] != "v1.2.3" {
				t.Fatalf("unexpected remote info payload for %s: %#v", input, payload)
			}
		}

		invalidOptions := cliruntime.Options{ModuleCatalogIndexURL: "https://index.acme.dev/v1/catalog.json"}
		if err := runModuleInfoRemote(cmd, runtimeScope, invalidOptions, "auth", "v0.0.0-0", false); err == nil || !strings.Contains(err.Error(), "index.json") {
			t.Fatalf("expected invalid catalog index url validation error, got %v", err)
		}
	})
}

func TestModuleRemoteCompatibilityEdgeBranches(t *testing.T) {
	runtimeScope := &commandTestScope{ctx: context.Background(), cfg: &config.Config{}}

	compatibleCatalog := startRemoteRegistryCatalogServer(t, []remoteCatalogModule{
		{
			Name:          "auth",
			LatestVersion: "v2.0.0",
			Description:   "Authentication",
			Versions:      []string{"v1.0.0", "v2.0.0"},
			VersionCLIRanges: map[string]string{
				"v1.0.0": ">=1.0.0 <2.0.0",
				"v2.0.0": ">=2.0.0 <3.0.0",
			},
		},
	})
	defer compatibleCatalog.Close()

	runtimeOptions := cliruntime.Options{ModuleCatalogIndexURL: compatibleCatalog.URL + "/v1/index.json"}

	t.Run("list all allows unresolved compat version", func(t *testing.T) {
		var out bytes.Buffer
		listCmd := &cobra.Command{}
		listCmd.SetOut(&out)

		if err := runModuleListRemote(listCmd, runtimeScope, runtimeOptions, "", true); err != nil {
			t.Fatalf("runModuleListRemote(showAll unresolved) error = %v", err)
		}
		output := out.String()
		if !strings.Contains(output, "MODULE") || !strings.Contains(output, "CLI_RANGE") || !strings.Contains(output, "auth") {
			t.Fatalf("unexpected remote list output: %q", output)
		}
	})

	t.Run("list rejects invalid compat version", func(t *testing.T) {
		listCmd := &cobra.Command{}
		err := runModuleListRemote(listCmd, runtimeScope, runtimeOptions, "bad", false)
		if err == nil || !strings.Contains(err.Error(), "ERR_CLI_COMPAT_VERSION_INVALID") {
			t.Fatalf("runModuleListRemote(invalid compat) error = %v, want invalid-version error", err)
		}
	})

	t.Run("list fails when catalog item misses cli range", func(t *testing.T) {
		malformedCatalog := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"modules": map[string]any{
					"auth": map[string]any{
						"moduleId":      "auth",
						"latestVersion": "v1.0.0",
						"package":       "@acme/choysum-auth",
						"versions": map[string]any{
							"v1.0.0": map[string]any{
								"package":   "@acme/choysum-auth",
								"tarball":   "https://registry.npmjs.org/@acme/choysum-auth/-/choysum-auth-1.0.0.tgz",
								"integrity": "sha512-auth-1.0.0",
							},
						},
					},
				},
			})
		}))
		defer malformedCatalog.Close()

		err := runModuleListRemote(&cobra.Command{}, runtimeScope, cliruntime.Options{ModuleCatalogIndexURL: malformedCatalog.URL + "/v1/index.json"}, "v1.5.0", false)
		if err == nil || !strings.Contains(err.Error(), "ERR_MODULE_CLI_RANGE_MISSING") {
			t.Fatalf("runModuleListRemote(malformed cli range) error = %v, want missing-range error", err)
		}
	})

	t.Run("info rejects explicit version outside compatible set", func(t *testing.T) {
		err := runModuleInfoRemote(&cobra.Command{}, runtimeScope, runtimeOptions, "auth@v1.0.0", "v2.5.0", false)
		if err == nil || !strings.Contains(err.Error(), "ERR_MODULE_NO_COMPATIBLE_VERSION") {
			t.Fatalf("runModuleInfoRemote(explicit incompatible) error = %v, want no-compatible error", err)
		}
	})

	t.Run("info all allows unresolved compat version", func(t *testing.T) {
		var out bytes.Buffer
		infoCmd := &cobra.Command{}
		infoCmd.SetOut(&out)

		if err := runModuleInfoRemote(infoCmd, runtimeScope, runtimeOptions, "auth", "", true); err != nil {
			t.Fatalf("runModuleInfoRemote(showAll unresolved) error = %v", err)
		}
		if !strings.Contains(out.String(), `"name": "auth"`) {
			t.Fatalf("unexpected info payload: %q", out.String())
		}
	})
}

func newModuleQueryTestScope(t *testing.T) (*moduleQueryTestScope, *gorm.DB) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "module-query.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	return &moduleQueryTestScope{ctx: context.Background(), db: db}, db
}

type moduleQueryTestScope struct {
	ctx context.Context
	db  *gorm.DB
}

func (e *moduleQueryTestScope) Run(fn func(runtimeScope scope.Scope) error) error { return fn(e) }
func (e *moduleQueryTestScope) Transactor() scope.Transactor {
	return scopetest.NewPassthroughTransactor(e)
}
func (e *moduleQueryTestScope) Session() *scope.Session { return &scope.Session{DB: e.db} }
func (e *moduleQueryTestScope) WithContext(ctx context.Context) scope.Scope {
	return &moduleQueryTestScope{ctx: ctx, db: e.db}
}
func (e *moduleQueryTestScope) Context() context.Context {
	if e.ctx != nil {
		return e.ctx
	}
	return context.Background()
}
func (e *moduleQueryTestScope) Logger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
func (e *moduleQueryTestScope) Config() *config.Config { return &config.Config{} }
func (e *moduleQueryTestScope) FactoryInput() scope.FactoryInput {
	return scopetest.FactoryInputFromConfig(e.Config())
}

func TestQueryModuleIndexViewsBranches(t *testing.T) {
	t.Run("returns empty when module index table is missing", func(t *testing.T) {
		runtimeScope, _ := newModuleQueryTestScope(t)

		views, hasIndex, err := queryModuleIndexViews(runtimeScope, "")
		if err != nil {
			t.Fatalf("queryModuleIndexViews() error = %v", err)
		}
		if hasIndex {
			t.Fatal("expected hasIndex=false when module index table is missing")
		}
		if len(views) != 0 {
			t.Fatalf("expected empty views, got %#v", views)
		}
	})

	t.Run("queries index without module table join", func(t *testing.T) {
		runtimeScope, db := newModuleQueryTestScope(t)
		if err := db.AutoMigrate(&modmeta.ModuleIndex{}); err != nil {
			t.Fatalf("auto migrate module index: %v", err)
		}
		if err := db.Create(&modmeta.ModuleIndex{
			ModuleName: "auth",
			OriginType: "local",
			OriginRef:  "modules/auth",
			Available:  true,
		}).Error; err != nil {
			t.Fatalf("seed module index row: %v", err)
		}

		views, hasIndex, err := queryModuleIndexViews(runtimeScope, "")
		if err != nil {
			t.Fatalf("queryModuleIndexViews() error = %v", err)
		}
		if !hasIndex {
			t.Fatal("expected hasIndex=true")
		}
		if len(views) != 1 || views[0].ModuleName != "auth" {
			t.Fatalf("unexpected views without module join: %#v", views)
		}
		if views[0].InstallStatus.String != "" {
			t.Fatalf("expected empty install status without module join, got %#v", views[0].InstallStatus)
		}
	})

	t.Run("joins module install status when module table exists", func(t *testing.T) {
		runtimeScope, db := newModuleQueryTestScope(t)
		if err := db.AutoMigrate(&modmeta.ModuleIndex{}, &meta.Module{}); err != nil {
			t.Fatalf("auto migrate module tables: %v", err)
		}
		if err := db.Create(&modmeta.ModuleIndex{
			ModuleName: "auth",
			OriginType: "local",
			OriginRef:  "modules/auth",
			Available:  true,
		}).Error; err != nil {
			t.Fatalf("seed module index row: %v", err)
		}
		if err := db.Create(&meta.Module{
			Name:           "auth",
			ApplicationStr: "auth",
			Status:         meta.Installed,
			Version:        "v1.0.0",
			Path:           "auth",
		}).Error; err != nil {
			t.Fatalf("seed installed module: %v", err)
		}

		views, hasIndex, err := queryModuleIndexViews(runtimeScope, "auth")
		if err != nil {
			t.Fatalf("queryModuleIndexViews(auth) error = %v", err)
		}
		if !hasIndex || len(views) != 1 {
			t.Fatalf("expected one indexed view, got hasIndex=%v views=%#v", hasIndex, views)
		}
		if !views[0].InstallStatus.Valid || views[0].InstallStatus.String != string(meta.Installed) {
			t.Fatalf("unexpected install status: %#v", views[0].InstallStatus)
		}
	})
}

func TestEnsurePurgeModuleNotInstalledSkipsWhenModuleTableMissing(t *testing.T) {
	runtimeScope, _ := newModuleQueryTestScope(t)

	if err := ensurePurgeModuleNotInstalled(runtimeScope, "auth"); err != nil {
		t.Fatalf("ensurePurgeModuleNotInstalled() error = %v, want nil when module table is missing", err)
	}
}

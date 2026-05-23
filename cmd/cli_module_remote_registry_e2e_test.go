// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	internalorigin "github.com/choysum-dev/choysum/internal/module/origin"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/meta"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type remoteCatalogModule struct {
	Name          string   `json:"name"`
	LatestVersion string   `json:"latestVersion"`
	Description   string   `json:"description,omitempty"`
	Versions      []string `json:"versions,omitempty"`
}

func startRemoteRegistryCatalogServer(t *testing.T, modules []remoteCatalogModule) *httptest.Server {
	t.Helper()

	byName := make(map[string]remoteCatalogModule, len(modules))
	for _, item := range modules {
		item.Name = strings.TrimSpace(item.Name)
		if item.Name == "" {
			t.Fatalf("module name must not be empty")
		}
		sort.Strings(item.Versions)
		byName[item.Name] = item
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/modules", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		q := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
		items := make([]remoteCatalogModule, 0, len(byName))
		for _, item := range byName {
			if q != "" && !strings.Contains(strings.ToLower(item.Name), q) {
				continue
			}
			items = append(items, remoteCatalogModule{
				Name:          item.Name,
				LatestVersion: item.LatestVersion,
				Description:   item.Description,
			})
		}
		sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"modules": items})
	})
	mux.HandleFunc("/api/v1/modules/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		name := path.Base(r.URL.Path)
		item, ok := byName[name]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(item)
	})

	return httptest.NewServer(mux)
}

func TestCLIModuleRemoteSearchListInfo(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	configPath := writeTempConfigWithDSN(t, "sqlite", writeTempSqliteDB(t), "")

	srv := startRemoteRegistryCatalogServer(t, []remoteCatalogModule{
		{Name: "auth", LatestVersion: "v1.2.3", Description: "Authentication", Versions: []string{"v1.0.0", "v1.2.3"}},
		{Name: "partner", LatestVersion: "v0.9.0", Description: "Partner management", Versions: []string{"v0.9.0"}},
	})
	defer srv.Close()

	if output, code := runCLI(t, "registry", "add", "corp", srv.URL, "--config", configPath); code != 0 {
		t.Fatalf("registry add failed, code=%d output=%s", code, output)
	}

	listOutput, listCode := runCLI(t, "module", "list", "--remote", "--registry", "corp", "--config", configPath)
	if listCode != 0 {
		t.Fatalf("module list --remote failed, code=%d output=%s", listCode, listOutput)
	}
	if !strings.Contains(listOutput, "auth") || !strings.Contains(listOutput, "partner") {
		t.Fatalf("expected remote module names in list output, got %q", listOutput)
	}
	if !strings.Contains(listOutput, "v1.2.3") {
		t.Fatalf("expected latest version in list output, got %q", listOutput)
	}

	searchOutput, searchCode := runCLI(t, "module", "search", "au", "--remote", "--registry", "corp", "--config", configPath)
	if searchCode != 0 {
		t.Fatalf("module search --remote failed, code=%d output=%s", searchCode, searchOutput)
	}
	if !strings.Contains(searchOutput, "auth") {
		t.Fatalf("expected auth in search output, got %q", searchOutput)
	}
	if strings.Contains(searchOutput, "partner") {
		t.Fatalf("did not expect partner in filtered search output, got %q", searchOutput)
	}

	infoOutput, infoCode := runCLI(t, "module", "info", "auth", "--remote", "--registry", "corp", "--config", configPath)
	if infoCode != 0 {
		t.Fatalf("module info --remote failed, code=%d output=%s", infoCode, infoOutput)
	}
	if !strings.Contains(infoOutput, `"name": "auth"`) || !strings.Contains(infoOutput, `"latestVersion": "v1.2.3"`) {
		t.Fatalf("unexpected info output: %q", infoOutput)
	}
}

func TestCLIModuleRemoteInfoNotFound(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	configPath := writeTempConfigWithDSN(t, "sqlite", writeTempSqliteDB(t), "")

	srv := startRemoteRegistryCatalogServer(t, []remoteCatalogModule{{Name: "auth", LatestVersion: "v1.0.0"}})
	defer srv.Close()

	if output, code := runCLI(t, "registry", "add", "corp", srv.URL, "--config", configPath); code != 0 {
		t.Fatalf("registry add failed, code=%d output=%s", code, output)
	}

	output, code := runCLI(t, "module", "info", "missing", "--remote", "--registry", "corp", "--config", configPath)
	if code == 0 {
		t.Fatalf("expected non-zero exit code for missing module, output=%s", output)
	}
	if !strings.Contains(output, "not found") {
		t.Fatalf("expected not found message, got %q", output)
	}
}

func TestCLIInstallLocalMissingGuidance(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	configPath := writeTempConfigWithDSN(t, "sqlite", writeTempSqliteDB(t), "")

	output, code := runCLI(t, "install", "missing", "--config", configPath)
	if code == 0 {
		t.Fatalf("expected install to fail for missing local module, output=%s", output)
	}
	if !strings.Contains(output, "module missing not found in addons path") {
		t.Fatalf("expected local missing error in output, got %q", output)
	}
	if !strings.Contains(output, "choysum module fetch <registry>/<module>@<version>") {
		t.Fatalf("expected module fetch guidance in output, got %q", output)
	}
	if !strings.Contains(output, "choysum install <registry>/<module>@<version>") {
		t.Fatalf("expected registry install guidance in output, got %q", output)
	}
}

func TestCLIUninstallMissingModuleReportsFailure(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	configPath := writeTempConfigWithDSN(t, "sqlite", writeTempSqliteDB(t), "")

	const missingModule = "__missing_copilot_module__"
	output, code := runCLI(t, "uninstall", missingModule, "--config", configPath)
	if code == 0 {
		t.Fatalf("expected uninstall to fail for missing module, output=%s", output)
	}
	if !strings.Contains(output, "module uninstall failed") {
		t.Fatalf("expected uninstall failure wrapper, got %q", output)
	}
	if !strings.Contains(output, missingModule) {
		t.Fatalf("expected missing module name in output, got %q", output)
	}
}

func TestCLIUpgradeMissingModuleReportsFailure(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	configPath := writeTempConfigWithDSN(t, "sqlite", writeTempSqliteDB(t), "")

	const missingModule = "__missing_copilot_module__"
	output, code := runCLI(t, "upgrade", missingModule, "--config", configPath)
	if code == 0 {
		t.Fatalf("expected upgrade to fail for missing module, output=%s", output)
	}
	if !strings.Contains(output, "module upgrade failed") {
		t.Fatalf("expected upgrade failure wrapper, got %q", output)
	}
	if !strings.Contains(output, missingModule) {
		t.Fatalf("expected missing module input in output, got %q", output)
	}
}

func TestCLIUpgradeRegistryRefRequiresAlias(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	configPath := writeTempConfigWithDSN(t, "sqlite", writeTempSqliteDB(t), "")

	output, code := runCLI(t, "upgrade", "corp/demo@v1.0.0", "--config", configPath)
	if code == 0 {
		t.Fatalf("expected registry upgrade to fail without alias config, output=%s", output)
	}
	if !strings.Contains(output, "registry alias corp not found") {
		t.Fatalf("expected registry alias error, got %q", output)
	}
}

func TestCLIModulePurgeRequiresUninstallWhenInstalled(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	workspaceRoot := t.TempDir()
	addonsPath := filepath.Join(workspaceRoot, "addons")
	if err := os.MkdirAll(addonsPath, 0o755); err != nil {
		t.Fatalf("create addons path: %v", err)
	}

	writeCommandManifest(t, addonsPath, "demo", `{
		"name": "demo",
		"description": "demo module",
		"application": "demo",
		"category": "test",
		"depends": [],
		"externalDependencies": {"application": {}, "node_module": {}, "binary": {}},
		"entryPoints": {"service": "./service/index.ts", "web": "./web/index.ts"},
		"version": "v0.1.0",
		"license": "Apache 2.0",
		"author": "test"
	}`)

	dbPath := writeTempSqliteDB(t)
	seedModuleStatusForCLI(t, dbPath, "demo", meta.Installed)
	configPath := writeTempConfigWithDSN(t, "sqlite", dbPath, addonsPath)

	if output, code := runCLI(t, "module", "fetch", "demo", "--config", configPath); code != 0 {
		t.Fatalf("module fetch failed, code=%d output=%s", code, output)
	}

	output, code := runCLI(t, "module", "purge", "demo", "--config", configPath)
	if code == 0 {
		t.Fatalf("expected purge to fail for installed module, output=%s", output)
	}
	if !strings.Contains(output, "run 'choysum uninstall demo' before purge") {
		t.Fatalf("expected uninstall guidance in purge output, got %q", output)
	}
	if _, err := os.Stat(filepath.Join(addonsPath, "demo")); err != nil {
		t.Fatalf("expected module directory to remain after blocked purge, err=%v", err)
	}
}

func TestCLIRegistryAddFetchUninstallPurgeFlow(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	workspaceRoot := t.TempDir()
	addonsPath := filepath.Join(workspaceRoot, "addons")
	if err := os.MkdirAll(addonsPath, 0o755); err != nil {
		t.Fatalf("create addons path: %v", err)
	}

	writeCommandManifest(t, addonsPath, "demo", `{
		"name": "demo",
		"description": "demo module",
		"application": "demo",
		"category": "test",
		"depends": [],
		"externalDependencies": {"application": {}, "node_module": {}, "binary": {}},
		"entryPoints": {"service": "./service/index.ts", "web": "./web/index.ts"},
		"version": "v0.1.0",
		"license": "Apache 2.0",
		"author": "test"
	}`)

	dbPath := writeTempSqliteDB(t)
	seedModuleStatusForCLI(t, dbPath, "demo", meta.Installed)
	configPath := writeTempConfigWithDSN(t, "sqlite", dbPath, addonsPath)

	srv := startRemoteRegistryCatalogServer(t, []remoteCatalogModule{{Name: "demo", LatestVersion: "v0.1.0"}})
	defer srv.Close()

	if output, code := runCLI(t, "registry", "add", "corp", srv.URL, "--config", configPath); code != 0 {
		t.Fatalf("registry add failed, code=%d output=%s", code, output)
	}
	if output, code := runCLI(t, "module", "fetch", "demo", "--config", configPath); code != 0 {
		t.Fatalf("module fetch failed, code=%d output=%s", code, output)
	}
	if output, code := runCLI(t, "uninstall", "demo", "--config", configPath); code != 0 {
		t.Fatalf("uninstall failed, code=%d output=%s", code, output)
	}
	if output, code := runCLI(t, "module", "purge", "demo", "--config", configPath); code != 0 {
		t.Fatalf("module purge failed, code=%d output=%s", code, output)
	}

	if _, err := os.Stat(filepath.Join(addonsPath, "demo")); !os.IsNotExist(err) {
		t.Fatalf("expected purged module dir to be removed, stat err=%v", err)
	}
	workspaceStateRoot := filepath.Dir(configPath)
	cfg, err := config.NewConfig(configPath)
	if err != nil {
		t.Fatalf("load config for lock lookup failed: %v", err)
	}
	if _, ok, err := internalorigin.NewLockStore(internalorigin.WithLockStoreDefaultChoysumPath(cfg.DefaultChoysumPath)).LookupBinding(workspaceStateRoot, "demo"); err != nil {
		t.Fatalf("lookup binding after purge failed: %v", err)
	} else if ok {
		t.Fatal("expected module binding to be removed after purge")
	}
}

func seedModuleStatusForCLI(t *testing.T, dbPath, moduleName string, status meta.Status) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&meta.IrModule{}); err != nil {
		t.Fatalf("auto-migrate ir module: %v", err)
	}

	var count int64
	if err := db.Model(&meta.IrModule{}).Where("name = ?", moduleName).Count(&count).Error; err != nil {
		t.Fatalf("count module fixture failed: %v", err)
	}
	if count == 0 {
		record := &meta.IrModule{
			Name:           moduleName,
			ApplicationStr: moduleName,
			Status:         status,
			Version:        "v0.1.0",
			Path:           moduleName,
		}
		if err := db.Create(record).Error; err != nil {
			t.Fatalf("create module status fixture: %v", err)
		}
		return
	}

	var existing meta.IrModule
	err = db.Where("name = ?", moduleName).Take(&existing).Error
	if err != nil {
		t.Fatalf("query module fixture failed: %v", err)
	}

	existing.Status = status
	if err := db.Save(&existing).Error; err != nil {
		t.Fatalf("update module status fixture: %v", err)
	}
}

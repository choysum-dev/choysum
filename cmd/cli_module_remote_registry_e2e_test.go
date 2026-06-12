// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cmd

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
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

func buildRemoteRegistryTarGz(t *testing.T, files map[string]string) []byte {
	t.Helper()

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)

	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	for _, path := range paths {
		content := files[path]
		hdr := &tar.Header{Name: path, Mode: 0o644, Size: int64(len(content))}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("write tar header: %v", err)
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			t.Fatalf("write tar content: %v", err)
		}
	}

	if err := tw.Close(); err != nil {
		t.Fatalf("close tar writer: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("close gzip writer: %v", err)
	}

	return buf.Bytes()
}

func startRemoteRegistryCatalogAndNPMServer(t *testing.T, moduleName, latestVersion string) *httptest.Server {
	t.Helper()

	moduleName = strings.TrimSpace(moduleName)
	if moduleName == "" {
		t.Fatal("module name must not be empty")
	}

	latestVersion = strings.TrimSpace(latestVersion)
	if latestVersion == "" {
		latestVersion = "v0.2.0"
	}
	if !strings.HasPrefix(latestVersion, "v") {
		latestVersion = "v" + latestVersion
	}

	npmVersion := strings.TrimPrefix(latestVersion, "v")
	packageName := "@acme/choysum-" + moduleName
	metadataPathEscaped := "/" + url.PathEscape(packageName)
	metadataPathPlain := "/" + packageName
	tarballPath := fmt.Sprintf("/tarballs/choysum-%s-%s.tgz", moduleName, npmVersion)
	integrity := "sha512-" + moduleName + "-" + npmVersion

	packageJSON := fmt.Sprintf(`{"name":"%s","version":"%s","description":"%s module","license":"Apache-2.0","author":"test","choysum":{"moduleName":"%s","application":"%s","category":"test","depends":[]}}`, packageName, npmVersion, moduleName, moduleName, moduleName)
	tarballBody := buildRemoteRegistryTarGz(t, map[string]string{"package/package.json": packageJSON})

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		baseURL := "http://" + r.Host
		switch {
		case r.URL.Path == "/v1/index.json":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"modules": map[string]any{
					moduleName: map[string]any{
						"moduleId":      moduleName,
						"description":   moduleName + " module",
						"latestVersion": latestVersion,
						"package":       packageName,
						"source": map[string]any{
							"type":      "npm",
							"registry":  baseURL,
							"package":   packageName,
							"version":   latestVersion,
							"tarball":   baseURL + tarballPath,
							"integrity": integrity,
						},
						"versions": map[string]any{
							latestVersion: map[string]any{
								"registry":  baseURL,
								"package":   packageName,
								"tarball":   baseURL + tarballPath,
								"integrity": integrity,
							},
						},
					},
				},
			})
		case r.URL.EscapedPath() == metadataPathEscaped || r.URL.Path == metadataPathPlain:
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"dist-tags": map[string]any{"latest": npmVersion},
				"versions": map[string]any{
					npmVersion: map[string]any{
						"name":        packageName,
						"version":     npmVersion,
						"description": moduleName + " module",
						"license":     "Apache-2.0",
						"author":      "test",
						"choysum": map[string]any{
							"moduleName":  moduleName,
							"application": moduleName,
							"category":    "test",
							"depends":     []string{},
						},
						"dist": map[string]any{
							"tarball":   baseURL + tarballPath,
							"integrity": integrity,
						},
					},
				},
			})
		case r.URL.Path == tarballPath:
			w.Header().Set("Content-Type", "application/octet-stream")
			_, _ = w.Write(tarballBody)
		default:
			http.NotFound(w, r)
		}
	}))
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
	mux.HandleFunc("/v1/index.json", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		modulesPayload := map[string]any{}
		for _, item := range byName {
			versions := item.Versions
			if len(versions) == 0 {
				versions = []string{item.LatestVersion}
			}
			versionPayload := map[string]any{}
			for _, version := range versions {
				version = strings.TrimSpace(version)
				if version == "" {
					continue
				}
				versionPayload[version] = map[string]any{
					"tarball":   "https://registry.npmjs.org/@acme/choysum-" + item.Name + "/-/choysum-" + item.Name + "-" + strings.TrimPrefix(version, "v") + ".tgz",
					"integrity": "sha512-" + item.Name + "-" + strings.TrimPrefix(version, "v"),
					"package":   "@acme/choysum-" + item.Name,
				}
			}
			modulesPayload[item.Name] = map[string]any{
				"moduleId":      item.Name,
				"description":   item.Description,
				"latestVersion": item.LatestVersion,
				"package":       "@acme/choysum-" + item.Name,
				"versions":      versionPayload,
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"modules": modulesPayload})
	})

	return httptest.NewServer(mux)
}

func setModuleCatalogIndexURLForCLIConfig(t *testing.T, configPath, indexURL string) {
	t.Helper()

	body, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	body = append(body, []byte(fmt.Sprintf("module_catalog_index_url: %q\n", indexURL))...)
	if err := os.WriteFile(configPath, body, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

func setLegacyRegistryURLConfigForCLI(t *testing.T, configPath string) {
	t.Helper()

	body, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	body = append(body, []byte("registry_index_url: \"https://index.legacy.dev/v1/index.json\"\n")...)
	body = append(body, []byte("registries:\n  official:\n    url: \"https://index.legacy.dev/v1/index.json\"\n")...)
	if err := os.WriteFile(configPath, body, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
}

func TestCLIModuleRemoteSearchListInfo(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	configPath := writeTempConfigWithDSN(t, "sqlite", writeTempSqliteDB(t), "")

	srv := startRemoteRegistryCatalogServer(t, []remoteCatalogModule{
		{Name: "auth", LatestVersion: "v1.2.3", Description: "Authentication", Versions: []string{"v1.0.0", "v1.2.3"}},
		{Name: "partner", LatestVersion: "v0.9.0", Description: "Partner management", Versions: []string{"v0.9.0"}},
	})
	defer srv.Close()
	setModuleCatalogIndexURLForCLIConfig(t, configPath, srv.URL+"/v1/index.json")

	listOutput, listCode := runCLI(t, "module", "list", "--remote", "--config", configPath)
	if listCode != 0 {
		t.Fatalf("module list --remote failed, code=%d output=%s", listCode, listOutput)
	}
	if !strings.Contains(listOutput, "auth") || !strings.Contains(listOutput, "partner") {
		t.Fatalf("expected remote module names in list output, got %q", listOutput)
	}
	if !strings.Contains(listOutput, "v1.2.3") {
		t.Fatalf("expected latest version in list output, got %q", listOutput)
	}

	searchOutput, searchCode := runCLI(t, "module", "search", "au", "--remote", "--config", configPath)
	if searchCode != 0 {
		t.Fatalf("module search --remote failed, code=%d output=%s", searchCode, searchOutput)
	}
	if !strings.Contains(searchOutput, "auth") {
		t.Fatalf("expected auth in search output, got %q", searchOutput)
	}
	if strings.Contains(searchOutput, "partner") {
		t.Fatalf("did not expect partner in filtered search output, got %q", searchOutput)
	}

	infoOutput, infoCode := runCLI(t, "module", "info", "auth", "--remote", "--config", configPath)
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
	setModuleCatalogIndexURLForCLIConfig(t, configPath, srv.URL+"/v1/index.json")

	output, code := runCLI(t, "module", "info", "missing", "--remote", "--config", configPath)
	if code == 0 {
		t.Fatalf("expected non-zero exit code for missing module, output=%s", output)
	}
	if !strings.Contains(output, "not found") {
		t.Fatalf("expected not found message, got %q", output)
	}
}

func TestCLIModuleRemoteRejectsLegacyRegistryURLConfig(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	configPath := writeTempConfigWithDSN(t, "sqlite", writeTempSqliteDB(t), "")
	setLegacyRegistryURLConfigForCLI(t, configPath)

	output, code := runCLI(t, "module", "list", "--remote", "--config", configPath)
	if code == 0 {
		t.Fatalf("expected non-zero exit code for legacy registry config, output=%s", output)
	}
	if !strings.Contains(output, "legacy module catalog config keys are no longer supported") {
		t.Fatalf("expected legacy module catalog rejection, got %q", output)
	}
	if !strings.Contains(output, "module_catalog_index_url") {
		t.Fatalf("expected module_catalog_index_url guidance, got %q", output)
	}
}

func TestCLIInstallLocalMissingGuidance(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	configPath := writeTempConfigWithDSN(t, "sqlite", writeTempSqliteDB(t), "")

	output, code := runCLI(t, "install", "missing", "--config", configPath)
	if code == 0 {
		t.Fatalf("expected install to fail for missing local module, output=%s", output)
	}
	if !strings.Contains(output, "module missing not found in modules path") {
		t.Fatalf("expected local missing error in output, got %q", output)
	}
	if !strings.Contains(output, "choysum module fetch <module>@<version>") {
		t.Fatalf("expected module fetch guidance in output, got %q", output)
	}
	if !strings.Contains(output, "choysum install <module>@<version>") {
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

func TestCLIUpgradeRejectsLegacyAliasSyntax(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	configPath := writeTempConfigWithDSN(t, "sqlite", writeTempSqliteDB(t), "")

	output, code := runCLI(t, "upgrade", "corp/demo@v1.0.0", "--config", configPath)
	if code == 0 {
		t.Fatalf("expected registry upgrade to fail for legacy alias syntax, output=%s", output)
	}
	if !strings.Contains(output, "is no longer supported") {
		t.Fatalf("expected legacy alias syntax error, got %q", output)
	}
}

func TestCLIUpgradeFlowWithGlobalRegistryIndex(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	workspaceRoot := t.TempDir()
	modulesPath := filepath.Join(workspaceRoot, "modules")
	if err := os.MkdirAll(modulesPath, 0o755); err != nil {
		t.Fatalf("create modules path: %v", err)
	}

	writeCommandPackage(t, modulesPath, "demo", `{
		"name": "@choysum-dev/demo",
		"version": "0.1.0",
		"description": "demo module",
		"license": "Apache-2.0",
		"author": "test",
		"choysum": {
			"moduleName": "demo",
			"application": "demo",
			"category": "test",
			"depends": []
		}
	}`)

	dbPath := writeTempSqliteDB(t)
	seedModuleStatusForCLI(t, dbPath, "demo", meta.Installed)
	configPath := writeTempConfigWithDSN(t, "sqlite", dbPath, modulesPath)

	srv := startRemoteRegistryCatalogAndNPMServer(t, "demo", "v0.2.0")
	defer srv.Close()
	setModuleCatalogIndexURLForCLIConfig(t, configPath, srv.URL+"/v1/index.json")

	output, code := runCLI(t, "upgrade", "demo@latest", "--config", configPath)
	if code != 0 {
		t.Fatalf("upgrade failed, code=%d output=%s", code, output)
	}

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	var upgraded meta.IrModule
	if err := db.Where("name = ?", "demo").Take(&upgraded).Error; err != nil {
		t.Fatalf("query upgraded module failed: %v", err)
	}
	if upgraded.Version != "v0.2.0" {
		t.Fatalf("unexpected upgraded version: %q", upgraded.Version)
	}
	if upgraded.Status != meta.Installed {
		t.Fatalf("unexpected upgraded status: %v", upgraded.Status)
	}

	cfg, err := config.NewConfig(configPath)
	if err != nil {
		t.Fatalf("load config for lock lookup failed: %v", err)
	}
	workspaceStateRoot := filepath.Dir(configPath)
	binding, ok, err := internalorigin.NewLockStore(internalorigin.WithLockStoreDefaultChoysumPath(cfg.DefaultChoysumPath)).LookupBinding(workspaceStateRoot, "demo")
	if err != nil {
		t.Fatalf("lookup lock binding failed: %v", err)
	}
	if !ok {
		t.Fatal("expected lock binding for upgraded module")
	}
	if binding.OriginType != internalorigin.OriginTypeRegistry {
		t.Fatalf("unexpected origin type: %q", binding.OriginType)
	}
	if binding.OriginRef != "demo@v0.2.0" {
		t.Fatalf("unexpected origin ref: %q", binding.OriginRef)
	}
	if binding.ResolvedVersion != "v0.2.0" {
		t.Fatalf("unexpected resolved version: %q", binding.ResolvedVersion)
	}

	packageJSONRaw, err := os.ReadFile(filepath.Join(modulesPath, "demo", "package.json"))
	if err != nil {
		t.Fatalf("read fetched package.json failed: %v", err)
	}
	packageJSON := string(packageJSONRaw)
	if !strings.Contains(packageJSON, `"version":"0.2.0"`) && !strings.Contains(packageJSON, `"version": "0.2.0"`) {
		t.Fatalf("expected fetched package.json version 0.2.0, got %q", packageJSON)
	}
}

func TestCLIModulePurgeRequiresUninstallWhenInstalled(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	workspaceRoot := t.TempDir()
	modulesPath := filepath.Join(workspaceRoot, "modules")
	if err := os.MkdirAll(modulesPath, 0o755); err != nil {
		t.Fatalf("create modules path: %v", err)
	}

	writeCommandPackage(t, modulesPath, "demo", `{
		"name": "@choysum-dev/demo",
		"version": "0.1.0",
		"description": "demo module",
		"license": "Apache-2.0",
		"author": "test",
		"type": "module",
		"main": "index.ts",
		"choysum": {
			"moduleName": "demo",
			"application": "demo",
			"category": "test",
			"depends": [],
			"entryPoints": {"service": "./service/index.ts", "web": "./web/index.ts"}
		}
	}`)

	dbPath := writeTempSqliteDB(t)
	seedModuleStatusForCLI(t, dbPath, "demo", meta.Installed)
	configPath := writeTempConfigWithDSN(t, "sqlite", dbPath, modulesPath)

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
	if _, err := os.Stat(filepath.Join(modulesPath, "demo")); err != nil {
		t.Fatalf("expected module directory to remain after blocked purge, err=%v", err)
	}
}

func TestCLIFetchUninstallPurgeFlowWithGlobalRegistryIndex(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	workspaceRoot := t.TempDir()
	modulesPath := filepath.Join(workspaceRoot, "modules")
	if err := os.MkdirAll(modulesPath, 0o755); err != nil {
		t.Fatalf("create modules path: %v", err)
	}

	writeCommandPackage(t, modulesPath, "demo", `{
		"name": "@choysum-dev/demo",
		"version": "0.1.0",
		"description": "demo module",
		"license": "Apache-2.0",
		"author": "test",
		"type": "module",
		"main": "index.ts",
		"choysum": {
			"moduleName": "demo",
			"application": "demo",
			"category": "test",
			"depends": [],
			"entryPoints": {"service": "./service/index.ts", "web": "./web/index.ts"}
		}
	}`)

	dbPath := writeTempSqliteDB(t)
	seedModuleStatusForCLI(t, dbPath, "demo", meta.Installed)
	configPath := writeTempConfigWithDSN(t, "sqlite", dbPath, modulesPath)

	srv := startRemoteRegistryCatalogServer(t, []remoteCatalogModule{{Name: "demo", LatestVersion: "v0.1.0"}})
	defer srv.Close()
	setModuleCatalogIndexURLForCLIConfig(t, configPath, srv.URL+"/v1/index.json")

	if output, code := runCLI(t, "module", "fetch", "demo", "--config", configPath); code != 0 {
		t.Fatalf("module fetch failed, code=%d output=%s", code, output)
	}
	if output, code := runCLI(t, "uninstall", "demo", "--config", configPath); code != 0 {
		t.Fatalf("uninstall failed, code=%d output=%s", code, output)
	}
	if output, code := runCLI(t, "module", "purge", "demo", "--config", configPath); code != 0 {
		t.Fatalf("module purge failed, code=%d output=%s", code, output)
	}

	if _, err := os.Stat(filepath.Join(modulesPath, "demo")); !os.IsNotExist(err) {
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

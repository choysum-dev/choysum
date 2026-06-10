// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package registry

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/choysum-dev/choysum/internal/module/origin/contract"
	cfg "github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	cp "github.com/otiai10/copy"
	xfmt "golang.org/x/exp/errors/fmt"
)

type Provider interface {
	PeekManifest(ctx context.Context, registryURL, moduleName, packageName, version string) (*meta.IrModule, error)
	Fetch(ctx context.Context, registryURL, moduleName, packageName, version string) (*meta.IrModule, error)
}

type SourceRegistryProvider struct {
	runtimeScope scope.Scope
	client       *http.Client
}

type ProviderOption func(*SourceRegistryProvider)

func WithHTTPClient(client *http.Client) ProviderOption {
	return func(p *SourceRegistryProvider) {
		if client != nil {
			p.client = client
		}
	}
}

func NewProvider(runtimeScope scope.Scope, opts ...ProviderOption) *SourceRegistryProvider {
	p := &SourceRegistryProvider{runtimeScope: runtimeScope, client: http.DefaultClient}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

func normalizeModuleNameVersion(moduleName, version string) (string, string, error) {
	moduleName = strings.TrimSpace(moduleName)
	if moduleName == "" {
		return "", "", xfmt.Errorf("module name is empty")
	}
	version = strings.TrimSpace(version)
	if version == "" {
		version = "latest"
	}
	return moduleName, version, nil
}

func (p *SourceRegistryProvider) httpGet(ctx context.Context, requestURL string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, err
	}
	client := p.client
	if client == nil {
		client = http.DefaultClient
	}
	return client.Do(req)
}

type npmVersionDist struct {
	Tarball   string `json:"tarball"`
	Integrity string `json:"integrity"`
}

type npmPackageMetadata struct {
	DistTags map[string]string          `json:"dist-tags"`
	Versions map[string]json.RawMessage `json:"versions"`
}

type npmVersionEnvelope struct {
	Dist npmVersionDist `json:"dist"`
}

func normalizePackageName(moduleName, packageName string) (string, error) {
	moduleName = strings.TrimSpace(moduleName)
	if moduleName == "" {
		return "", xfmt.Errorf("module name is empty")
	}
	packageName = strings.TrimSpace(packageName)
	if packageName != "" {
		return packageName, nil
	}
	return moduleName, nil
}

func normalizeDefaultRegistryMetadataBaseURL(defaultRegistryURL string) (string, error) {
	defaultRegistryURL = strings.TrimSpace(defaultRegistryURL)
	if defaultRegistryURL == "" {
		defaultRegistryURL = cfg.DefaultNPMRegistryURL
	}

	parsed, err := url.Parse(defaultRegistryURL)
	if err != nil {
		return "", xfmt.Errorf("invalid default npm registry url %q: %w", defaultRegistryURL, err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", xfmt.Errorf("unsupported default npm registry url: %s", defaultRegistryURL)
	}
	if strings.TrimSpace(parsed.Host) == "" {
		return "", xfmt.Errorf("unsupported default npm registry url: %s", defaultRegistryURL)
	}

	parsed.RawQuery = ""
	parsed.Fragment = ""
	parsed.Path = strings.TrimRight(parsed.Path, "/")

	return strings.TrimRight(parsed.String(), "/"), nil
}

func normalizeRegistryMetadataBaseURL(registryURL, defaultRegistryURL string) (string, error) {
	fallbackBaseURL, err := normalizeDefaultRegistryMetadataBaseURL(defaultRegistryURL)
	if err != nil {
		return "", err
	}

	registryURL = strings.TrimSpace(registryURL)
	if registryURL == "" {
		return fallbackBaseURL, nil
	}

	parsed, err := url.Parse(registryURL)
	if err != nil {
		return "", xfmt.Errorf("invalid registry url %q: %w", registryURL, err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", xfmt.Errorf("unsupported registry url: %s", registryURL)
	}
	if strings.TrimSpace(parsed.Host) == "" {
		return "", xfmt.Errorf("unsupported registry url: %s", registryURL)
	}

	pathLower := strings.ToLower(parsed.Path)
	hostLower := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(parsed.Hostname())), ".")
	if hostLower == "github.com" || hostLower == "www.github.com" || strings.HasSuffix(pathLower, ".json") || strings.Contains(pathLower, "/api/v1/modules") || strings.Contains(pathLower, "/api/modules") {
		return "", xfmt.Errorf("legacy catalog source registry %q is no longer supported; use module_catalog_index_url for catalog index and set source.registry to npm registry base URL", registryURL)
	}

	parsed.RawQuery = ""
	parsed.Fragment = ""
	parsed.Path = strings.TrimRight(parsed.Path, "/")

	return strings.TrimRight(parsed.String(), "/"), nil
}

func registryPackageMetadataURL(registryURL, moduleName, packageName, defaultRegistryURL string) (string, string, error) {
	baseURL, err := normalizeRegistryMetadataBaseURL(registryURL, defaultRegistryURL)
	if err != nil {
		return "", "", err
	}
	packageName, err = normalizePackageName(moduleName, packageName)
	if err != nil {
		return "", "", err
	}
	metadataURL := strings.TrimRight(baseURL, "/") + "/" + url.PathEscape(packageName)
	return metadataURL, packageName, nil
}

func (p *SourceRegistryProvider) defaultRegistryURL() string {
	if p == nil {
		return cfg.DefaultNPMRegistryURL
	}
	configured := strings.TrimSpace(runtimeOptionsFromScope(p.runtimeScope).npmRegistryURL)
	if configured != "" {
		return configured
	}
	return cfg.DefaultNPMRegistryURL
}

func (p *SourceRegistryProvider) fetchPackageMetadata(ctx context.Context, registryURL, moduleName, packageName string) (*npmPackageMetadata, string, error) {
	metadataURL, packageName, err := registryPackageMetadataURL(registryURL, moduleName, packageName, p.defaultRegistryURL())
	if err != nil {
		return nil, "", err
	}
	resp, err := p.httpGet(ctx, metadataURL)
	if err != nil {
		return nil, "", xfmt.Errorf("get npm metadata: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", xfmt.Errorf("error getting npm metadata url: %s status code: %d", metadataURL, resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", xfmt.Errorf("read npm metadata: %w", err)
	}

	metadata := &npmPackageMetadata{}
	if err := json.Unmarshal(body, metadata); err != nil {
		return nil, "", xfmt.Errorf("error decoding npm metadata: %w", err)
	}
	if len(metadata.Versions) == 0 {
		return nil, "", xfmt.Errorf("npm metadata has no versions for package %q", packageName)
	}
	return metadata, packageName, nil
}

func resolveNPMVersion(metadata *npmPackageMetadata, requestedVersion string) (string, json.RawMessage, error) {
	if metadata == nil {
		return "", nil, xfmt.Errorf("npm metadata is nil")
	}
	versionKey := strings.TrimSpace(requestedVersion)
	if versionKey == "" {
		versionKey = "latest"
	}

	if raw, ok := metadata.Versions[versionKey]; ok {
		return versionKey, raw, nil
	}

	if trimmed := strings.TrimPrefix(versionKey, "v"); trimmed != versionKey {
		if raw, ok := metadata.Versions[trimmed]; ok {
			return trimmed, raw, nil
		}
	}

	if tagged, ok := metadata.DistTags[versionKey]; ok {
		tagged = strings.TrimSpace(tagged)
		if raw, ok := metadata.Versions[tagged]; ok {
			return tagged, raw, nil
		}
		if raw, ok := metadata.Versions[strings.TrimPrefix(tagged, "v")]; ok {
			return strings.TrimPrefix(tagged, "v"), raw, nil
		}
	}

	if versionKey == "latest" {
		if tagged := strings.TrimSpace(metadata.DistTags["latest"]); tagged != "" {
			if raw, ok := metadata.Versions[tagged]; ok {
				return tagged, raw, nil
			}
			if raw, ok := metadata.Versions[strings.TrimPrefix(tagged, "v")]; ok {
				return strings.TrimPrefix(tagged, "v"), raw, nil
			}
		}
		if len(metadata.Versions) == 1 {
			for k, raw := range metadata.Versions {
				return k, raw, nil
			}
		}
		return "", nil, xfmt.Errorf("npm metadata missing latest dist-tag")
	}

	return "", nil, xfmt.Errorf("version %q not found in npm metadata", requestedVersion)
}

func normalizePackageAuthor(raw []byte) ([]byte, error) {
	payload := map[string]any{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, xfmt.Errorf("decode package.json payload: %w", err)
	}
	if author, ok := payload["author"]; ok {
		switch typed := author.(type) {
		case string:
			payload["author"] = strings.TrimSpace(typed)
		case map[string]any:
			if name, ok := typed["name"].(string); ok {
				payload["author"] = strings.TrimSpace(name)
			} else {
				delete(payload, "author")
			}
		default:
			delete(payload, "author")
		}
	}
	normalized, err := json.Marshal(payload)
	if err != nil {
		return nil, xfmt.Errorf("encode normalized package.json: %w", err)
	}
	return normalized, nil
}

func parseModuleFromPackageJSON(raw []byte, moduleName, modulePath string) (*meta.IrModule, error) {
	normalizedRaw, err := normalizePackageAuthor(raw)
	if err != nil {
		return nil, err
	}
	result, err := contract.ParsePackageJSONToIrModule(normalizedRaw, modulePath, nil)
	if err != nil {
		return nil, xfmt.Errorf("parse package.json: %w", err)
	}
	module := result.Module
	if strings.TrimSpace(module.Name) != strings.TrimSpace(moduleName) {
		return nil, xfmt.Errorf("package.json choysum.moduleName %q does not match requested module %q", strings.TrimSpace(module.Name), strings.TrimSpace(moduleName))
	}
	module.Name = moduleName
	module.Path = modulePath
	return module, nil
}

func extractTarballURL(versionRaw json.RawMessage) (string, string, error) {
	envelope := npmVersionEnvelope{}
	if err := json.Unmarshal(versionRaw, &envelope); err != nil {
		return "", "", xfmt.Errorf("decode npm version dist: %w", err)
	}
	downloadURL := strings.TrimSpace(envelope.Dist.Tarball)
	if downloadURL == "" {
		return "", "", xfmt.Errorf("no tarball url found in npm metadata")
	}
	return downloadURL, strings.TrimSpace(envelope.Dist.Integrity), nil
}

type packageInspection struct {
	module      *meta.IrModule
	downloadURL string
	integrity   string
}

func (p *SourceRegistryProvider) inspectRegistryPackage(ctx context.Context, registryURL, moduleName, packageName, version string) (*packageInspection, error) {
	metadata, packageName, err := p.fetchPackageMetadata(ctx, registryURL, moduleName, packageName)
	if err != nil {
		return nil, err
	}
	resolvedVersion, versionRaw, err := resolveNPMVersion(metadata, version)
	if err != nil {
		return nil, xfmt.Errorf("inspect package %q: %w", packageName, err)
	}
	module, err := parseModuleFromPackageJSON(versionRaw, moduleName, "")
	if err != nil {
		return nil, xfmt.Errorf("inspect package %q version %q: %w", packageName, resolvedVersion, err)
	}
	downloadURL, integrity, err := extractTarballURL(versionRaw)
	if err != nil {
		return nil, xfmt.Errorf("inspect package %q version %q: %w", packageName, resolvedVersion, err)
	}
	module.Tarball = downloadURL
	module.Integrity = integrity
	return &packageInspection{module: module, downloadURL: downloadURL, integrity: integrity}, nil
}

func isUnsafeTarPath(name string) bool {
	if name == "" {
		return true
	}
	if filepath.IsAbs(name) {
		return true
	}
	if runtime.GOOS == "windows" && strings.Contains(name, ":") {
		return true
	}
	clean := filepath.Clean(name)
	if clean == "." {
		return false
	}
	if strings.HasPrefix(clean, ".."+string(os.PathSeparator)) || clean == ".." {
		return true
	}
	if strings.HasPrefix(clean, "../") {
		return true
	}
	return false
}

func safeJoin(baseDir, name string) (string, error) {
	if isUnsafeTarPath(name) {
		return "", xfmt.Errorf("unsafe tar path: %q", name)
	}
	joined := filepath.Join(baseDir, filepath.Clean(name))
	rel, err := filepath.Rel(baseDir, joined)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", xfmt.Errorf("tar path escapes target dir: %q", name)
	}
	return joined, nil
}

func validateTarballURL(downloadURL string) error {
	parsed, err := url.Parse(downloadURL)
	if err != nil {
		return err
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return xfmt.Errorf("invalid url scheme")
	}
	if !strings.HasSuffix(parsed.Path, ".tar.gz") && !strings.HasSuffix(parsed.Path, ".tgz") {
		return xfmt.Errorf("invalid file type")
	}
	return nil
}

func scorePackageJSONPath(path, moduleName string) int {
	path = filepath.ToSlash(path)
	score := 0
	if strings.Contains(path, "/modules/"+moduleName+"/") {
		score += 4
	}
	if strings.HasSuffix(path, "/package/package.json") {
		score += 3
	}
	if strings.HasSuffix(path, "/"+moduleName+"/package.json") {
		score += 2
	}
	if strings.HasSuffix(path, "/package.json") {
		score += 1
	}
	return score
}

func findBestPackageJSONPath(rootDir, moduleName string) (string, error) {
	moduleName = strings.TrimSpace(moduleName)
	bestPath := ""
	bestScore := -1
	bestDepth := 1 << 30

	err := filepath.WalkDir(rootDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Base(path) != "package.json" {
			return nil
		}
		rel, err := filepath.Rel(rootDir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		score := scorePackageJSONPath(rel, moduleName)
		depth := strings.Count(rel, "/")
		if score > bestScore || (score == bestScore && depth < bestDepth) {
			bestPath = path
			bestScore = score
			bestDepth = depth
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if bestPath == "" {
		return "", xfmt.Errorf("package.json not found in tarball")
	}
	return bestPath, nil
}

func (p *SourceRegistryProvider) extractTarballToDir(ctx context.Context, downloadURL, targetDir string) error {
	if err := validateTarballURL(downloadURL); err != nil {
		return err
	}
	resp, err := p.httpGet(ctx, downloadURL)
	if err != nil {
		return xfmt.Errorf("download tarball: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return xfmt.Errorf("unexpected HTTP status %s", resp.Status)
	}

	gr, err := gzip.NewReader(resp.Body)
	if err != nil {
		return xfmt.Errorf("gzip reader: %w", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return xfmt.Errorf("read tar: %w", err)
		}
		if h == nil {
			continue
		}
		if isUnsafeTarPath(h.Name) {
			return xfmt.Errorf("read tar: unsafe path %q", h.Name)
		}
		outPath, err := safeJoin(targetDir, h.Name)
		if err != nil {
			return xfmt.Errorf("read tar: %w", err)
		}

		switch h.Typeflag {
		case tar.TypeSymlink, tar.TypeLink:
			return xfmt.Errorf("read tar: link entry not allowed: %q", h.Name)
		case tar.TypeDir:
			if err := os.MkdirAll(outPath, os.FileMode(h.Mode)); err != nil {
				return xfmt.Errorf("read tar: mkdir failed: %w", err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
				return xfmt.Errorf("read tar: mkdirall failed: %w", err)
			}
			f, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(h.Mode))
			if err != nil {
				return xfmt.Errorf("read tar: create failed: %w", err)
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return xfmt.Errorf("read tar: copy failed: %w", err)
			}
			if err := f.Close(); err != nil {
				return xfmt.Errorf("read tar: close failed: %w", err)
			}
		}
	}

	return nil
}

func (p *SourceRegistryProvider) PeekManifest(ctx context.Context, registryURL, moduleName, packageName, version string) (*meta.IrModule, error) {
	if p == nil || p.runtimeScope == nil {
		return nil, xfmt.Errorf("registry provider env is nil")
	}
	var err error
	moduleName, version, err = normalizeModuleNameVersion(moduleName, version)
	if err != nil {
		return nil, err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	inspection, err := p.inspectRegistryPackage(ctx, registryURL, moduleName, packageName, version)
	if err != nil {
		return nil, err
	}
	if inspection.module == nil {
		return nil, xfmt.Errorf("empty package inspection result")
	}
	return inspection.module, nil
}

func (p *SourceRegistryProvider) Fetch(ctx context.Context, registryURL, moduleName, packageName, version string) (*meta.IrModule, error) {
	if p == nil || p.runtimeScope == nil {
		return nil, xfmt.Errorf("registry provider env is nil")
	}
	var err error
	moduleName, version, err = normalizeModuleNameVersion(moduleName, version)
	if err != nil {
		return nil, err
	}
	if ctx == nil {
		ctx = context.Background()
	}

	inspection, err := p.inspectRegistryPackage(ctx, registryURL, moduleName, packageName, version)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(inspection.downloadURL) == "" {
		return nil, xfmt.Errorf("no tarball url found in npm metadata")
	}

	tmpDir, err := os.MkdirTemp(os.TempDir(), "choysum-source-fetch-")
	if err != nil {
		return nil, xfmt.Errorf("create temp dir failed: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := p.extractTarballToDir(ctx, inspection.downloadURL, tmpDir); err != nil {
		return nil, err
	}

	packageJSONPath, err := findBestPackageJSONPath(tmpDir, moduleName)
	if err != nil {
		return nil, err
	}
	moduleSourcePath := filepath.Dir(packageJSONPath)

	modulesPath := runtimeOptionsFromScope(p.runtimeScope).modulesPath
	if strings.TrimSpace(modulesPath) == "" {
		return nil, xfmt.Errorf("modules path is empty")
	}
	moduleTargetPath := filepath.Join(modulesPath, moduleName)
	if _, err := os.Stat(moduleTargetPath); err == nil {
		return nil, os.ErrExist
	} else if !os.IsNotExist(err) {
		return nil, xfmt.Errorf("stat module target path failed: %w", err)
	}

	if err := cp.Copy(moduleSourcePath, moduleTargetPath, cp.Options{
		Skip: func(srcinfo os.FileInfo, src, dest string) (bool, error) {
			if srcinfo.IsDir() && srcinfo.Name() == ".git" {
				return true, nil
			}
			return false, nil
		},
	}); err != nil {
		return nil, xfmt.Errorf("copy module to modules failed: %w", err)
	}

	raw, err := os.ReadFile(filepath.Join(moduleTargetPath, "package.json"))
	if err != nil {
		return nil, xfmt.Errorf("read package.json: %w", err)
	}
	module, err := parseModuleFromPackageJSON(raw, moduleName, moduleTargetPath)
	if err != nil {
		return nil, err
	}
	module.Tarball = inspection.downloadURL
	module.Integrity = inspection.integrity
	return module, nil
}

// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package registry

import (
	"archive/tar"
	"bytes"
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
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	cp "github.com/otiai10/copy"
	xfmt "golang.org/x/exp/errors/fmt"
)

type Provider interface {
	PeekManifest(ctx context.Context, registryURL, moduleName, version string) (*meta.IrModule, error)
	Fetch(ctx context.Context, registryURL, moduleName, version string) (*meta.IrModule, error)
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

// NewLegacyFetcherProvider is kept for migration compatibility.
func NewLegacyFetcherProvider(runtimeScope scope.Scope) *SourceRegistryProvider {
	return NewProvider(runtimeScope)
}

func decodeModuleManifest(r io.Reader) (*meta.IrModule, error) {
	module := &meta.IrModule{}
	if err := json.NewDecoder(r).Decode(module); err != nil {
		return nil, err
	}
	module.Status = meta.ToInstall
	return module, nil
}

func applyEntryPoints(module *meta.IrModule) error {
	if module == nil {
		return nil
	}
	entryPointsMap := make(map[string]string)
	if module.EntryPoints != nil {
		if err := json.Unmarshal(module.EntryPoints, &entryPointsMap); err != nil {
			return xfmt.Errorf("error unmarshalling entry points: %w", err)
		}
		if webEntryPoint, ok := entryPointsMap["web"]; ok {
			module.WebEntryPoint = webEntryPoint
		}
		if serviceEntryPoint, ok := entryPointsMap["service"]; ok {
			module.ServiceEntryPoint = serviceEntryPoint
		}
	}
	return nil
}

func validateAndNormalizeManifestSemVer(mod *meta.IrModule, manifestHint string) error {
	if mod == nil {
		return nil
	}
	ver := strings.TrimSpace(mod.Version)
	if ver == "" {
		return xfmt.Errorf("empty manifest version (module=%q, manifest=%q)", strings.TrimSpace(mod.Name), strings.TrimSpace(manifestHint))
	}
	normalized, err := contract.NormalizeVersion(ver)
	if err != nil {
		return xfmt.Errorf("invalid manifest version %q (module=%q, manifest=%q); expected SemVer like v0.1.0", ver, strings.TrimSpace(mod.Name), strings.TrimSpace(manifestHint))
	}
	mod.Version = normalized
	return nil
}

func looksLikeModuleManifest(mod *meta.IrModule) bool {
	if mod == nil {
		return false
	}
	return strings.TrimSpace(mod.ApplicationStr) != "" ||
		len(mod.DependsStr) > 0 ||
		mod.EntryPoints != nil ||
		strings.TrimSpace(mod.WebEntryPoint) != "" ||
		strings.TrimSpace(mod.ServiceEntryPoint) != ""
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

func registryManifestURL(registryURL, moduleName, version string) (string, error) {
	registryURL = strings.TrimSpace(registryURL)
	if registryURL == "" {
		registryURL = DefaultRegistryURL
	}
	if !strings.HasPrefix(registryURL, "https://github.com/") {
		return "", xfmt.Errorf("unsupported registry url: %s", registryURL)
	}
	ownerAndRepo := strings.Split(strings.TrimPrefix(registryURL, "https://github.com/"), "/")
	if len(ownerAndRepo) < 2 || strings.TrimSpace(ownerAndRepo[0]) == "" || strings.TrimSpace(ownerAndRepo[1]) == "" {
		return "", xfmt.Errorf("invalid registry url: %s", registryURL)
	}
	owner := strings.TrimSpace(ownerAndRepo[0])
	repo := strings.TrimSpace(ownerAndRepo[1])
	return "https://raw.githubusercontent.com/" + owner + "/" + repo + "/main/addons/" + moduleName + "/" + version + "/manifest.json", nil
}

type manifestInspection struct {
	module      *meta.IrModule
	downloadURL string
}

func (p *SourceRegistryProvider) inspectRegistryManifest(ctx context.Context, registryURL, moduleName, version string) (*manifestInspection, error) {
	manifestURL, err := registryManifestURL(registryURL, moduleName, version)
	if err != nil {
		return nil, err
	}
	resp, err := p.httpGet(ctx, manifestURL)
	if err != nil {
		return nil, xfmt.Errorf("get registry manifest: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, xfmt.Errorf("error getting registry manifest url: %s status code: %d", manifestURL, resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, xfmt.Errorf("read registry manifest: %w", err)
	}

	manifestMap := make(map[string]interface{})
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&manifestMap); err != nil {
		return nil, xfmt.Errorf("error decoding registry manifest: %w", err)
	}

	result := &manifestInspection{}
	if modFromRegistry, err := decodeModuleManifest(bytes.NewReader(body)); err == nil && looksLikeModuleManifest(modFromRegistry) {
		modFromRegistry.Name = moduleName
		modFromRegistry.Path = ""
		if err := applyEntryPoints(modFromRegistry); err != nil {
			return nil, err
		}
		if err := validateAndNormalizeManifestSemVer(modFromRegistry, "registry:manifest.json"); err != nil {
			return nil, err
		}
		result.module = modFromRegistry
	}

	if v, ok := manifestMap["tarball"]; ok {
		if s, ok := v.(string); ok {
			result.downloadURL = strings.TrimSpace(s)
		}
	} else if v, ok := manifestMap["repository"]; ok {
		if s, ok := v.(string); ok {
			repoURL := strings.TrimSpace(s)
			if repoURL != "" {
				result.downloadURL = repoURL + "/archive/refs/tags/" + version + ".tar.gz"
			}
		}
	}

	return result, nil
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

func scoreManifestPath(path, moduleName string) int {
	path = filepath.ToSlash(path)
	score := 0
	if strings.Contains(path, "/addons/"+moduleName+"/") {
		score += 3
	}
	if strings.HasSuffix(path, "/"+moduleName+"/manifest.json") {
		score += 2
	}
	if strings.Count(path, "/") <= 4 {
		score += 1
	}
	return score
}

func findBestManifestPath(rootDir, moduleName string) (string, error) {
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
		if filepath.Base(path) != "manifest.json" {
			return nil
		}
		rel, err := filepath.Rel(rootDir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		score := scoreManifestPath(rel, moduleName)
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
		return "", xfmt.Errorf("manifest.json not found in tarball")
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

func (p *SourceRegistryProvider) peekManifestFromTarball(ctx context.Context, moduleName, downloadURL string) (*meta.IrModule, error) {
	tmpDir, err := os.MkdirTemp(os.TempDir(), "choysum-source-peek-")
	if err != nil {
		return nil, xfmt.Errorf("create temp dir failed: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := p.extractTarballToDir(ctx, downloadURL, tmpDir); err != nil {
		return nil, err
	}
	manifestPath, err := findBestManifestPath(tmpDir, moduleName)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(manifestPath)
	if err != nil {
		return nil, xfmt.Errorf("open manifest: %w", err)
	}
	defer f.Close()
	mod, err := decodeModuleManifest(f)
	if err != nil {
		return nil, xfmt.Errorf("decode manifest from tar: %w", err)
	}
	mod.Name = moduleName
	mod.Path = ""
	if err := applyEntryPoints(mod); err != nil {
		return nil, err
	}
	if err := validateAndNormalizeManifestSemVer(mod, "tar:manifest.json"); err != nil {
		return nil, err
	}
	return mod, nil
}

func (p *SourceRegistryProvider) PeekManifest(ctx context.Context, registryURL, moduleName, version string) (*meta.IrModule, error) {
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
	inspection, err := p.inspectRegistryManifest(ctx, registryURL, moduleName, version)
	if err != nil {
		return nil, err
	}
	if inspection.module != nil {
		return inspection.module, nil
	}
	if strings.TrimSpace(inspection.downloadURL) == "" {
		return nil, xfmt.Errorf("no download url found in registry manifest")
	}
	return p.peekManifestFromTarball(ctx, moduleName, inspection.downloadURL)
}

func (p *SourceRegistryProvider) Fetch(ctx context.Context, registryURL, moduleName, version string) (*meta.IrModule, error) {
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

	inspection, err := p.inspectRegistryManifest(ctx, registryURL, moduleName, version)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(inspection.downloadURL) == "" {
		return nil, xfmt.Errorf("no download url found in registry manifest")
	}

	tmpDir, err := os.MkdirTemp(os.TempDir(), "choysum-source-fetch-")
	if err != nil {
		return nil, xfmt.Errorf("create temp dir failed: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := p.extractTarballToDir(ctx, inspection.downloadURL, tmpDir); err != nil {
		return nil, err
	}

	manifestPath, err := findBestManifestPath(tmpDir, moduleName)
	if err != nil {
		return nil, err
	}
	moduleSourcePath := filepath.Dir(manifestPath)

	addonsPath := runtimeOptionsFromScope(p.runtimeScope).addonsPath
	if strings.TrimSpace(addonsPath) == "" {
		return nil, xfmt.Errorf("addons path is empty")
	}
	moduleTargetPath := filepath.Join(addonsPath, moduleName)
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
		return nil, xfmt.Errorf("copy module to addons failed: %w", err)
	}

	f, err := os.Open(filepath.Join(moduleTargetPath, "manifest.json"))
	if err != nil {
		return nil, xfmt.Errorf("open manifest: %w", err)
	}
	defer f.Close()
	mod, err := decodeModuleManifest(f)
	if err != nil {
		return nil, xfmt.Errorf("decode manifest: %w", err)
	}
	mod.Name = moduleName
	mod.Path = moduleTargetPath
	if err := applyEntryPoints(mod); err != nil {
		return nil, err
	}
	if err := validateAndNormalizeManifestSemVer(mod, filepath.Join(moduleTargetPath, "manifest.json")); err != nil {
		return nil, err
	}
	return mod, nil
}

// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package registry

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/choysum-dev/choysum/pkg/scope"
	xfmt "golang.org/x/exp/errors/fmt"
	"golang.org/x/mod/semver"
)

type CatalogModule struct {
	Name          string   `json:"name"`
	LatestVersion string   `json:"latestVersion,omitempty"`
	Description   string   `json:"description,omitempty"`
	Versions      []string `json:"versions,omitempty"`
}

type Catalog struct {
	runtimeScope scope.Scope
	provider     Provider
	client       *http.Client
}

type CatalogOption func(*Catalog)

func WithCatalogHTTPClient(client *http.Client) CatalogOption {
	return func(c *Catalog) {
		if client != nil {
			c.client = client
		}
	}
}

func WithCatalogProvider(provider Provider) CatalogOption {
	return func(c *Catalog) {
		if provider != nil {
			c.provider = provider
		}
	}
}

func NewCatalog(runtimeScope scope.Scope, opts ...CatalogOption) *Catalog {
	c := &Catalog{
		runtimeScope: runtimeScope,
		provider:     NewProvider(runtimeScope),
		client:       http.DefaultClient,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *Catalog) List(ctx context.Context, registryURL, query string) ([]CatalogModule, error) {
	registryURL = strings.TrimSpace(registryURL)
	if registryURL == "" {
		registryURL = DefaultRegistryURL
	}
	query = strings.TrimSpace(query)
	if isGitHubRegistryURL(registryURL) {
		return c.listFromGitHub(ctx, registryURL, query)
	}
	return c.listFromRemoteAPI(ctx, registryURL, query)
}

func (c *Catalog) Info(ctx context.Context, registryURL, moduleName string) (*CatalogModule, error) {
	registryURL = strings.TrimSpace(registryURL)
	if registryURL == "" {
		registryURL = DefaultRegistryURL
	}
	moduleName = strings.TrimSpace(moduleName)
	if moduleName == "" {
		return nil, xfmt.Errorf("module name is empty")
	}
	if isGitHubRegistryURL(registryURL) {
		return c.infoFromGitHub(ctx, registryURL, moduleName)
	}
	return c.infoFromRemoteAPI(ctx, registryURL, moduleName)
}

func (c *Catalog) listFromRemoteAPI(ctx context.Context, registryURL, query string) ([]CatalogModule, error) {
	u := strings.TrimRight(registryURL, "/") + "/api/v1/modules"
	parsed, err := url.Parse(u)
	if err != nil {
		return nil, xfmt.Errorf("invalid registry url %q: %w", registryURL, err)
	}
	if query != "" {
		q := parsed.Query()
		q.Set("q", query)
		parsed.RawQuery = q.Encode()
	}
	payload, err := c.fetchJSON(ctx, parsed.String())
	if err != nil {
		return nil, err
	}

	type listEnvelope struct {
		Modules []CatalogModule `json:"modules"`
	}
	env := listEnvelope{}
	if err := json.Unmarshal(payload, &env); err == nil && env.Modules != nil {
		normalizeCatalogModules(env.Modules)
		return env.Modules, nil
	}

	items := []CatalogModule{}
	if err := json.Unmarshal(payload, &items); err != nil {
		return nil, xfmt.Errorf("decode registry catalog list failed: %w", err)
	}
	normalizeCatalogModules(items)
	return items, nil
}

func (c *Catalog) infoFromRemoteAPI(ctx context.Context, registryURL, moduleName string) (*CatalogModule, error) {
	u := strings.TrimRight(registryURL, "/") + "/api/v1/modules/" + url.PathEscape(moduleName)
	payload, err := c.fetchJSON(ctx, u)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, xfmt.Errorf("remote module %q not found", moduleName)
		}
		return nil, err
	}
	item := &CatalogModule{}
	if err := json.Unmarshal(payload, item); err != nil {
		return nil, xfmt.Errorf("decode registry module info failed: %w", err)
	}
	if strings.TrimSpace(item.Name) == "" {
		item.Name = moduleName
	}
	normalizeCatalogModule(item)
	return item, nil
}

func (c *Catalog) listFromGitHub(ctx context.Context, registryURL, query string) ([]CatalogModule, error) {
	owner, repo, err := parseGitHubOwnerRepo(registryURL)
	if err != nil {
		return nil, err
	}
	entries, err := c.listGitHubDirectory(ctx, owner, repo, "addons")
	if err != nil {
		return nil, err
	}
	out := make([]CatalogModule, 0, len(entries))
	for _, entry := range entries {
		if entry.Type != "dir" {
			continue
		}
		name := strings.TrimSpace(entry.Name)
		if name == "" {
			continue
		}
		if query != "" && !strings.Contains(strings.ToLower(name), strings.ToLower(query)) {
			continue
		}
		latest, _, err := c.fetchGitHubLatestVersion(ctx, owner, repo, name)
		if err != nil {
			latest = ""
		}
		out = append(out, CatalogModule{Name: name, LatestVersion: latest})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (c *Catalog) infoFromGitHub(ctx context.Context, registryURL, moduleName string) (*CatalogModule, error) {
	owner, repo, err := parseGitHubOwnerRepo(registryURL)
	if err != nil {
		return nil, err
	}
	latest, versions, err := c.fetchGitHubLatestVersion(ctx, owner, repo, moduleName)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, xfmt.Errorf("remote module %q not found", moduleName)
		}
		return nil, err
	}
	item := &CatalogModule{Name: moduleName, LatestVersion: latest, Versions: versions}
	if c.provider != nil && latest != "" {
		if mod, peekErr := c.provider.PeekManifest(ctx, registryURL, moduleName, latest); peekErr == nil && mod != nil {
			item.Description = strings.TrimSpace(mod.Description)
		}
	}
	normalizeCatalogModule(item)
	return item, nil
}

type githubContentItem struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

func (c *Catalog) listGitHubDirectory(ctx context.Context, owner, repo, dir string) ([]githubContentItem, error) {
	u := "https://api.github.com/repos/" + owner + "/" + repo + "/contents/" + strings.TrimPrefix(dir, "/")
	payload, err := c.fetchJSON(ctx, u)
	if err != nil {
		return nil, err
	}
	items := []githubContentItem{}
	if err := json.Unmarshal(payload, &items); err != nil {
		return nil, xfmt.Errorf("decode github directory listing failed: %w", err)
	}
	return items, nil
}

func (c *Catalog) fetchGitHubLatestVersion(ctx context.Context, owner, repo, moduleName string) (string, []string, error) {
	entries, err := c.listGitHubDirectory(ctx, owner, repo, path.Join("addons", moduleName))
	if err != nil {
		return "", nil, err
	}
	versions := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.Type != "dir" {
			continue
		}
		ver := strings.TrimSpace(entry.Name)
		if ver == "" {
			continue
		}
		versions = append(versions, ver)
	}
	if len(versions) == 0 {
		return "", nil, os.ErrNotExist
	}
	sort.Strings(versions)
	latest := pickLatestVersion(versions)
	return latest, versions, nil
}

func pickLatestVersion(versions []string) string {
	if len(versions) == 0 {
		return ""
	}
	best := ""
	bestRaw := ""
	for _, raw := range versions {
		normalized, ok := normalizeSemVer(raw)
		if !ok {
			continue
		}
		if best == "" || semver.Compare(normalized, best) > 0 {
			best = normalized
			bestRaw = raw
		}
	}
	if bestRaw != "" {
		return bestRaw
	}
	return versions[len(versions)-1]
}

func normalizeSemVer(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}
	candidate := raw
	if !strings.HasPrefix(candidate, "v") {
		candidate = "v" + candidate
	}
	if !semver.IsValid(candidate) {
		return "", false
	}
	return candidate, true
}

func isGitHubRegistryURL(registryURL string) bool {
	u, err := url.Parse(strings.TrimSpace(registryURL))
	if err != nil {
		return false
	}
	return strings.EqualFold(u.Scheme, "https") && strings.EqualFold(u.Host, "github.com")
}

func parseGitHubOwnerRepo(registryURL string) (string, string, error) {
	u, err := url.Parse(strings.TrimSpace(registryURL))
	if err != nil {
		return "", "", xfmt.Errorf("invalid github registry url %q: %w", registryURL, err)
	}
	if !strings.EqualFold(u.Scheme, "https") || !strings.EqualFold(u.Host, "github.com") {
		return "", "", xfmt.Errorf("unsupported github registry url: %s", registryURL)
	}
	parts := strings.Split(strings.Trim(strings.TrimSpace(u.Path), "/"), "/")
	if len(parts) < 2 {
		return "", "", xfmt.Errorf("invalid github registry url: %s", registryURL)
	}
	owner := strings.TrimSpace(parts[0])
	repo := strings.TrimSpace(parts[1])
	if owner == "" || repo == "" {
		return "", "", xfmt.Errorf("invalid github registry url: %s", registryURL)
	}
	return owner, repo, nil
}

func (c *Catalog) fetchJSON(ctx context.Context, requestURL string) ([]byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, xfmt.Errorf("build request failed: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	client := c.client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, xfmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, os.ErrNotExist
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, xfmt.Errorf("unexpected HTTP status %s", resp.Status)
	}
	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, xfmt.Errorf("read response failed: %w", err)
	}
	return payload, nil
}

func normalizeCatalogModules(items []CatalogModule) {
	for i := range items {
		normalizeCatalogModule(&items[i])
	}
}

func normalizeCatalogModule(item *CatalogModule) {
	if item == nil {
		return
	}
	item.Name = strings.TrimSpace(item.Name)
	item.LatestVersion = strings.TrimSpace(item.LatestVersion)
	item.Description = strings.TrimSpace(item.Description)
	if len(item.Versions) > 0 {
		versions := make([]string, 0, len(item.Versions))
		for _, version := range item.Versions {
			version = strings.TrimSpace(version)
			if version == "" {
				continue
			}
			versions = append(versions, version)
		}
		sort.Strings(versions)
		item.Versions = versions
		if item.LatestVersion == "" {
			item.LatestVersion = pickLatestVersion(versions)
		}
	}
}

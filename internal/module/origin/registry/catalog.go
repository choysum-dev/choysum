// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package registry

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
	xfmt "golang.org/x/exp/errors/fmt"
	"golang.org/x/mod/semver"
)

type CatalogModule struct {
	Name             string            `json:"name"`
	LatestVersion    string            `json:"latestVersion,omitempty"`
	Description      string            `json:"description,omitempty"`
	Versions         []string          `json:"versions,omitempty"`
	VersionCLIRanges map[string]string `json:"versionCLIRanges,omitempty"`
	NPMPackage       string            `json:"npmPackage,omitempty"`
	Source           *CatalogSource    `json:"source,omitempty"`
}

type CatalogSource struct {
	Type      string `json:"type,omitempty"`
	Registry  string `json:"registry,omitempty"`
	Package   string `json:"package,omitempty"`
	Version   string `json:"version,omitempty"`
	Tarball   string `json:"tarball,omitempty"`
	Integrity string `json:"integrity,omitempty"`
}

func (m CatalogModule) ResolvedNPMPackage() string {
	if m.Source != nil {
		if pkg := strings.TrimSpace(m.Source.Package); pkg != "" {
			return pkg
		}
	}
	return strings.TrimSpace(m.NPMPackage)
}

func (m CatalogModule) ResolvedNPMRegistry(defaultRegistry string) string {
	if m.Source != nil {
		if registry := strings.TrimSpace(m.Source.Registry); registry != "" {
			return registry
		}
	}
	return strings.TrimSpace(defaultRegistry)
}

func (m CatalogModule) CLIRangeForVersion(version string) (string, bool) {
	version = strings.TrimSpace(version)
	if version == "" || len(m.VersionCLIRanges) == 0 {
		return "", false
	}

	if value, ok := m.VersionCLIRanges[version]; ok {
		value = strings.TrimSpace(value)
		if value != "" {
			return value, true
		}
	}

	targetNormalized, targetHasSemVer := normalizeSemVer(version)
	keys := make([]string, 0, len(m.VersionCLIRanges))
	for rawVersion := range m.VersionCLIRanges {
		keys = append(keys, rawVersion)
	}
	sort.Strings(keys)

	for _, rawVersion := range keys {
		rawRange := m.VersionCLIRanges[rawVersion]
		rawRange = strings.TrimSpace(rawRange)
		if rawRange == "" {
			continue
		}
		trimmedVersion := strings.TrimSpace(rawVersion)
		if trimmedVersion == version {
			return rawRange, true
		}
		if !targetHasSemVer {
			continue
		}
		if rawNormalized, ok := normalizeSemVer(trimmedVersion); ok && rawNormalized == targetNormalized {
			return rawRange, true
		}
	}

	return "", false
}

func (m CatalogModule) LatestCLIRange() (string, bool) {
	return m.CLIRangeForVersion(m.LatestVersion)
}

type catalogIndexDocument struct {
	Modules map[string]catalogIndexModule `json:"modules"`
}

type catalogIndexModule struct {
	ModuleID      string                       `json:"moduleId,omitempty"`
	Name          string                       `json:"name,omitempty"`
	Description   string                       `json:"description,omitempty"`
	Package       string                       `json:"package,omitempty"`
	LatestVersion string                       `json:"latestVersion,omitempty"`
	Versions      map[string]catalogIndexEntry `json:"versions,omitempty"`
	Source        *CatalogSource               `json:"source,omitempty"`
}

type catalogIndexEntry struct {
	Registry  string                   `json:"registry,omitempty"`
	Package   string                   `json:"package,omitempty"`
	Tarball   string                   `json:"tarball,omitempty"`
	Integrity string                   `json:"integrity,omitempty"`
	Choysum   catalogIndexEntryChoysum `json:"choysum,omitempty"`
	Source    *CatalogSource           `json:"source,omitempty"`
}

type catalogIndexEntryChoysum struct {
	CLI string `json:"cli,omitempty"`
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
		client:       newDefaultHTTPClient(),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *Catalog) List(ctx context.Context, indexURL, query string) ([]CatalogModule, error) {
	index, err := c.loadIndex(ctx, indexURL)
	if err != nil {
		return nil, err
	}
	query = strings.ToLower(strings.TrimSpace(query))

	items := make([]CatalogModule, 0, len(index.Modules))
	for moduleName, module := range index.Modules {
		if query != "" {
			name := resolveCatalogModuleName(moduleName, module)
			if !strings.Contains(strings.ToLower(name), query) {
				continue
			}
		}
		item := buildCatalogModule(moduleName, module)
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	return items, nil
}

func (c *Catalog) Info(ctx context.Context, indexURL, moduleName string) (*CatalogModule, error) {
	moduleName = strings.TrimSpace(moduleName)
	if moduleName == "" {
		return nil, xfmt.Errorf("module name is empty")
	}

	index, err := c.loadIndex(ctx, indexURL)
	if err != nil {
		return nil, err
	}

	if module, ok := index.Modules[moduleName]; ok {
		item := buildCatalogModule(moduleName, module)
		return &item, nil
	}

	keys := make([]string, 0, len(index.Modules))
	for k := range index.Modules {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		module := index.Modules[k]
		if strings.EqualFold(strings.TrimSpace(k), moduleName) || strings.EqualFold(strings.TrimSpace(module.ModuleID), moduleName) || strings.EqualFold(strings.TrimSpace(module.Name), moduleName) {
			item := buildCatalogModule(k, module)
			return &item, nil
		}
	}

	return nil, xfmt.Errorf("remote module %q not found", moduleName)
}

func (c *Catalog) loadIndex(ctx context.Context, indexURL string) (*catalogIndexDocument, error) {
	indexURL = strings.TrimSpace(indexURL)
	if indexURL == "" {
		indexURL = config.DefaultModuleCatalogIndexURL
	}
	payload, err := c.fetchJSON(ctx, indexURL)
	if err != nil {
		return nil, err
	}
	index := &catalogIndexDocument{}
	if err := json.Unmarshal(payload, index); err != nil {
		return nil, xfmt.Errorf("decode module catalog index failed: %w", err)
	}
	if index.Modules == nil {
		index.Modules = map[string]catalogIndexModule{}
	}
	return index, nil
}

func buildCatalogModule(moduleName string, module catalogIndexModule) CatalogModule {
	name := resolveCatalogModuleName(moduleName, module)

	item := CatalogModule{
		Name:             name,
		Description:      strings.TrimSpace(module.Description),
		NPMPackage:       strings.TrimSpace(module.Package),
		LatestVersion:    strings.TrimSpace(module.LatestVersion),
		VersionCLIRanges: map[string]string{},
	}
	if module.Source != nil {
		sourceCopy := *module.Source
		item.Source = &sourceCopy
	}

	versions := make([]string, 0, len(module.Versions))
	for version, entry := range module.Versions {
		trimmedVersion := strings.TrimSpace(version)
		versions = append(versions, trimmedVersion)
		cliRange := strings.TrimSpace(entry.Choysum.CLI)
		if cliRange != "" {
			item.VersionCLIRanges[trimmedVersion] = cliRange
		}
	}
	sortCatalogVersions(versions)
	item.Versions = versions
	if item.LatestVersion == "" {
		item.LatestVersion = pickLatestVersion(versions)
	}

	entry, ok := resolveCatalogVersionEntry(module.Versions, item.LatestVersion)
	if ok {
		if entry.Source != nil {
			if item.Source == nil {
				item.Source = cloneCatalogSource(entry.Source)
			} else {
				mergeCatalogSource(item.Source, entry.Source)
			}
		}
		if item.Source == nil {
			item.Source = &CatalogSource{}
		}
		if item.Source.Package == "" {
			item.Source.Package = strings.TrimSpace(entry.Package)
		}
		if item.Source.Registry == "" {
			item.Source.Registry = strings.TrimSpace(entry.Registry)
		}
		if item.Source.Tarball == "" {
			item.Source.Tarball = strings.TrimSpace(entry.Tarball)
		}
		if item.Source.Integrity == "" {
			item.Source.Integrity = strings.TrimSpace(entry.Integrity)
		}
		if item.Source.Version == "" {
			item.Source.Version = strings.TrimSpace(item.LatestVersion)
		}
		if item.Source.Type == "" && (item.Source.Package != "" || item.Source.Registry != "" || item.Source.Tarball != "") {
			item.Source.Type = "npm"
		}
		if item.NPMPackage == "" {
			item.NPMPackage = strings.TrimSpace(item.Source.Package)
		}
	}

	if item.Source != nil && item.Source.Package == "" {
		item.Source.Package = item.NPMPackage
	}
	if len(item.VersionCLIRanges) == 0 {
		item.VersionCLIRanges = nil
	}

	normalizeCatalogModule(&item)
	return item
}

func resolveCatalogModuleName(moduleName string, module catalogIndexModule) string {
	name := strings.TrimSpace(moduleName)
	if moduleID := strings.TrimSpace(module.ModuleID); moduleID != "" {
		name = moduleID
	}
	if name == "" {
		name = strings.TrimSpace(module.Name)
	}
	return name
}

func cloneCatalogSource(src *CatalogSource) *CatalogSource {
	if src == nil {
		return nil
	}
	clone := *src
	return &clone
}

func mergeCatalogSource(dst *CatalogSource, src *CatalogSource) {
	if dst == nil || src == nil {
		return
	}
	if value := strings.TrimSpace(src.Type); value != "" {
		dst.Type = value
	}
	if value := strings.TrimSpace(src.Registry); value != "" {
		dst.Registry = value
	}
	if value := strings.TrimSpace(src.Package); value != "" {
		dst.Package = value
	}
	if value := strings.TrimSpace(src.Version); value != "" {
		dst.Version = value
	}
	if value := strings.TrimSpace(src.Tarball); value != "" {
		dst.Tarball = value
	}
	if value := strings.TrimSpace(src.Integrity); value != "" {
		dst.Integrity = value
	}
}

func resolveCatalogVersionEntry(versions map[string]catalogIndexEntry, targetVersion string) (catalogIndexEntry, bool) {
	targetVersion = strings.TrimSpace(targetVersion)
	if targetVersion == "" {
		return catalogIndexEntry{}, false
	}
	if entry, ok := versions[targetVersion]; ok {
		return entry, true
	}
	keys := make([]string, 0, len(versions))
	for rawVersion := range versions {
		keys = append(keys, rawVersion)
	}
	sort.Strings(keys)

	for _, rawVersion := range keys {
		entry := versions[rawVersion]
		trimmedVersion := strings.TrimSpace(rawVersion)
		if trimmedVersion == targetVersion {
			return entry, true
		}
	}

	targetNormalized, targetHasSemVer := normalizeSemVer(targetVersion)
	if !targetHasSemVer {
		return catalogIndexEntry{}, false
	}
	for _, rawVersion := range keys {
		entry := versions[rawVersion]
		trimmedVersion := strings.TrimSpace(rawVersion)
		if rawNormalized, ok := normalizeSemVer(trimmedVersion); ok && rawNormalized == targetNormalized {
			return entry, true
		}
	}
	return catalogIndexEntry{}, false
}

func sortCatalogVersions(versions []string) {
	sort.Slice(versions, func(i, j int) bool {
		leftSemVer, leftOK := normalizeSemVer(versions[i])
		rightSemVer, rightOK := normalizeSemVer(versions[j])
		if leftOK && rightOK {
			if cmp := semver.Compare(leftSemVer, rightSemVer); cmp != 0 {
				return cmp < 0
			}
			return versions[i] < versions[j]
		}
		if leftOK != rightOK {
			return !leftOK
		}
		return versions[i] < versions[j]
	})
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
		client = newDefaultHTTPClient()
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
	item.NPMPackage = strings.TrimSpace(item.NPMPackage)
	if item.Source != nil {
		item.Source.Type = strings.TrimSpace(item.Source.Type)
		item.Source.Registry = strings.TrimSpace(item.Source.Registry)
		item.Source.Package = strings.TrimSpace(item.Source.Package)
		item.Source.Version = strings.TrimSpace(item.Source.Version)
		item.Source.Tarball = strings.TrimSpace(item.Source.Tarball)
		item.Source.Integrity = strings.TrimSpace(item.Source.Integrity)
		if item.Source.Type == "" && item.Source.Registry == "" && item.Source.Package == "" && item.Source.Version == "" && item.Source.Tarball == "" && item.Source.Integrity == "" {
			item.Source = nil
		}
	}
	if len(item.Versions) > 0 {
		versions := make([]string, 0, len(item.Versions))
		for _, version := range item.Versions {
			version = strings.TrimSpace(version)
			if version == "" {
				continue
			}
			versions = append(versions, version)
		}
		sortCatalogVersions(versions)
		item.Versions = versions
		if item.LatestVersion == "" {
			item.LatestVersion = pickLatestVersion(versions)
		}
	}
	if len(item.VersionCLIRanges) > 0 {
		normalizedRanges := make(map[string]string, len(item.VersionCLIRanges))
		for version, cliRange := range item.VersionCLIRanges {
			version = strings.TrimSpace(version)
			cliRange = strings.TrimSpace(cliRange)
			if version == "" || cliRange == "" {
				continue
			}
			normalizedRanges[version] = cliRange
		}
		if len(normalizedRanges) == 0 {
			item.VersionCLIRanges = nil
		} else {
			item.VersionCLIRanges = normalizedRanges
		}
	}
}

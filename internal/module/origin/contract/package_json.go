// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package contract

import (
	"encoding/json"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/choysum-dev/choysum/pkg/meta"
	xfmt "golang.org/x/exp/errors/fmt"
	"golang.org/x/mod/semver"
)

var strictSemVerV = regexp.MustCompile(`^v\d+\.\d+\.\d+([\-\+].+)?$`)

// PackageJSON defines the canonical addon package contract for Phase A.
type PackageJSON struct {
	Name             string            `json:"name"`
	Version          string            `json:"version"`
	Description      string            `json:"description,omitempty"`
	License          string            `json:"license,omitempty"`
	Author           string            `json:"author,omitempty"`
	PeerDependencies map[string]string `json:"peerDependencies,omitempty"`
	Choysum          ChoysumMeta       `json:"choysum"`
}

type ChoysumMeta struct {
	ModuleName    string            `json:"moduleName"`
	Application   string            `json:"application"`
	Category      string            `json:"category,omitempty"`
	Depends       []string          `json:"depends,omitempty"`
	EntryPoints   map[string]string `json:"entryPoints,omitempty"`
	Data          []string          `json:"data,omitempty"`
	Demo          []string          `json:"demo,omitempty"`
	Compatibility CompatibilityMeta `json:"compatibility,omitempty"`
}

type CompatibilityMeta struct {
	Choysum string `json:"choysum,omitempty"`
}

// PackageToModuleResult is the canonical output of package.json parsing.
// PeerDependencies are returned as a dedicated map because they are not part of
// ExternalDependencies.node_module anymore.
type PackageToModuleResult struct {
	Module           *meta.IrModule
	PeerDependencies map[string]string
}

// DecodePackageJSON decodes raw package.json bytes into the canonical contract.
func DecodePackageJSON(raw []byte) (*PackageJSON, error) {
	if len(raw) == 0 {
		return nil, xfmt.Errorf("empty package.json content")
	}
	pkg := &PackageJSON{}
	if err := json.Unmarshal(raw, pkg); err != nil {
		return nil, xfmt.Errorf("decode package.json: %w", err)
	}
	return pkg, nil
}

// ParsePackageJSONToIrModule decodes, validates, and converts package.json into
// the internal meta.IrModule representation.
func ParsePackageJSONToIrModule(raw []byte, modulePath string, binaryDependencies map[string]string) (*PackageToModuleResult, error) {
	pkg, err := DecodePackageJSON(raw)
	if err != nil {
		return nil, err
	}
	return PackageJSONToIrModule(pkg, modulePath, binaryDependencies)
}

// PackageJSONToIrModule converts a validated PackageJSON into meta.IrModule.
func PackageJSONToIrModule(pkg *PackageJSON, modulePath string, binaryDependencies map[string]string) (*PackageToModuleResult, error) {
	if err := ValidatePackageJSON(pkg); err != nil {
		return nil, err
	}

	normalizedVersion, err := NormalizeVersion(pkg.Version)
	if err != nil {
		return nil, err
	}

	mod := &meta.IrModule{
		Name:           strings.TrimSpace(pkg.Choysum.ModuleName),
		Version:        normalizedVersion,
		Description:    strings.TrimSpace(pkg.Description),
		License:        strings.TrimSpace(pkg.License),
		Author:         strings.TrimSpace(pkg.Author),
		ApplicationStr: strings.TrimSpace(pkg.Choysum.Application),
		Category:       strings.TrimSpace(pkg.Choysum.Category),
		Path:           strings.TrimSpace(modulePath),
		Status:         meta.ToInstall,
	}

	if len(pkg.Choysum.Depends) > 0 {
		dependsJSON, err := json.Marshal(pkg.Choysum.Depends)
		if err != nil {
			return nil, xfmt.Errorf("marshal depends: %w", err)
		}
		mod.DependsStr = dependsJSON
	}

	if len(pkg.Choysum.EntryPoints) > 0 {
		entryPointsJSON, err := json.Marshal(pkg.Choysum.EntryPoints)
		if err != nil {
			return nil, xfmt.Errorf("marshal entryPoints: %w", err)
		}
		mod.EntryPoints = entryPointsJSON
		if webEntry, ok := pkg.Choysum.EntryPoints["web"]; ok {
			mod.WebEntryPoint = strings.TrimSpace(webEntry)
		}
		if serviceEntry, ok := pkg.Choysum.EntryPoints["service"]; ok {
			mod.ServiceEntryPoint = strings.TrimSpace(serviceEntry)
		}
	}

	if len(pkg.Choysum.Data) > 0 {
		dataJSON, err := json.Marshal(pkg.Choysum.Data)
		if err != nil {
			return nil, xfmt.Errorf("marshal data: %w", err)
		}
		mod.DataStr = dataJSON
	}

	if len(pkg.Choysum.Demo) > 0 {
		demoJSON, err := json.Marshal(pkg.Choysum.Demo)
		if err != nil {
			return nil, xfmt.Errorf("marshal demo: %w", err)
		}
		mod.DemoStr = demoJSON
	}

	externalDependenciesJSON, err := json.Marshal(BuildExternalDependencies(binaryDependencies))
	if err != nil {
		return nil, xfmt.Errorf("marshal externalDependencies: %w", err)
	}
	mod.ExternalDependencies = externalDependenciesJSON

	peerDependencies := CanonicalPeerDependencies(pkg.PeerDependencies)

	return &PackageToModuleResult{
		Module:           mod,
		PeerDependencies: peerDependencies,
	}, nil
}

// NormalizeVersion converts a SemVer version into the internal v-prefixed form.
func NormalizeVersion(version string) (string, error) {
	version = strings.TrimSpace(version)
	if version == "" {
		return "", xfmt.Errorf("empty package version")
	}
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}
	if !strictSemVerV.MatchString(version) || !semver.IsValid(version) {
		return "", xfmt.Errorf("invalid package version %q; expected SemVer like 0.1.0 or v0.1.0", strings.TrimSpace(strings.TrimPrefix(version, "v")))
	}
	return version, nil
}

// ValidatePackageJSON validates mandatory contract fields and value shapes.
func ValidatePackageJSON(pkg *PackageJSON) error {
	if pkg == nil {
		return xfmt.Errorf("package contract is nil")
	}
	if strings.TrimSpace(pkg.Name) == "" {
		return xfmt.Errorf("package name is required")
	}
	if _, err := NormalizeVersion(pkg.Version); err != nil {
		return err
	}
	if strings.TrimSpace(pkg.Choysum.ModuleName) == "" {
		return xfmt.Errorf("choysum.moduleName is required")
	}
	if strings.TrimSpace(pkg.Choysum.Application) == "" {
		return xfmt.Errorf("choysum.application is required")
	}

	for _, dep := range pkg.Choysum.Depends {
		if strings.TrimSpace(dep) == "" {
			return xfmt.Errorf("choysum.depends contains empty module name")
		}
	}

	if len(pkg.Choysum.EntryPoints) > 0 {
		for k, v := range pkg.Choysum.EntryPoints {
			key := strings.TrimSpace(k)
			if key != "service" && key != "web" {
				return xfmt.Errorf("unsupported choysum.entryPoints key %q; allowed keys are service and web", k)
			}
			if err := validateAddonRelativePath(v, "choysum.entryPoints."+key); err != nil {
				return err
			}
		}
	}

	for i, p := range pkg.Choysum.Data {
		if err := validateAddonRelativePath(p, xfmt.Sprintf("choysum.data[%d]", i)); err != nil {
			return err
		}
	}

	for i, p := range pkg.Choysum.Demo {
		if err := validateAddonRelativePath(p, xfmt.Sprintf("choysum.demo[%d]", i)); err != nil {
			return err
		}
	}

	return nil
}

func validateAddonRelativePath(raw, field string) error {
	v := strings.TrimSpace(raw)
	if v == "" {
		return xfmt.Errorf("%s cannot be empty", field)
	}

	// Normalize path separators for deterministic rule checks.
	normalized := strings.ReplaceAll(v, "\\", "/")

	if strings.HasPrefix(normalized, "/") || filepath.IsAbs(v) || windowsDrivePrefix.MatchString(normalized) {
		return xfmt.Errorf("%s must be a relative path, got %q", field, raw)
	}

	segments := strings.Split(normalized, "/")
	for _, seg := range segments {
		if seg == ".." {
			return xfmt.Errorf("%s cannot contain parent traversal, got %q", field, raw)
		}
	}

	clean := path.Clean(normalized)
	if clean == "." || clean == "" {
		return xfmt.Errorf("%s must point to a file path, got %q", field, raw)
	}

	return nil
}

var windowsDrivePrefix = regexp.MustCompile(`^[a-zA-Z]:[/\\]`)

// CanonicalPeerDependencies returns a stable key-order copy for deterministic output.
func CanonicalPeerDependencies(peer map[string]string) map[string]string {
	if len(peer) == 0 {
		return map[string]string{}
	}
	keys := make([]string, 0, len(peer))
	for k := range peer {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make(map[string]string, len(peer))
	for _, k := range keys {
		out[k] = peer[k]
	}
	return out
}

// BuildExternalDependencies keeps only the binary channel.
// JS dependencies must come from package.json peerDependencies.
func BuildExternalDependencies(binary map[string]string) map[string]map[string]string {
	out := map[string]map[string]string{
		"binary": {},
	}
	if len(binary) == 0 {
		return out
	}
	keys := make([]string, 0, len(binary))
	for k := range binary {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		out["binary"][k] = binary[k]
	}
	return out
}

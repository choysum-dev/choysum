// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package contract

import (
	"encoding/json"
	"regexp"
	"sort"
	"strings"

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
			if strings.TrimSpace(v) == "" {
				return xfmt.Errorf("choysum.entryPoints.%s cannot be empty", key)
			}
		}
	}

	return nil
}

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

// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package esmresolver

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type packageJSONPins struct {
	Dependencies     map[string]string `json:"dependencies"`
	PeerDependencies map[string]string `json:"peerDependencies"`
}

// ExactPinsFromPackageJSON returns bare-import pins for exact versions listed in
// moduleRoot/package.json peerDependencies and dependencies. Ranges (^/~/>=)
// and dist-tags are skipped so WithBareImportPins can consume the map safely.
// Missing package.json yields (nil, nil).
func ExactPinsFromPackageJSON(moduleRoot string) (map[string]string, error) {
	moduleRoot = strings.TrimSpace(moduleRoot)
	if moduleRoot == "" {
		return nil, nil
	}
	pkgPath := filepath.Join(moduleRoot, "package.json")
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read package.json: %w", err)
	}
	var pkg packageJSONPins
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, fmt.Errorf("parse package.json: %w", err)
	}
	out := make(map[string]string)
	// Peers first so a module's own exact dependency version wins when both
	// blocks pin the same package.
	collectExactPins(out, pkg.PeerDependencies)
	collectExactPins(out, pkg.Dependencies)
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

func collectExactPins(dst map[string]string, src map[string]string) {
	for name, ver := range src {
		name = strings.TrimSpace(name)
		ver = strings.TrimSpace(ver)
		if name == "" || !isExactPinVersion(ver) {
			continue
		}
		dst[name] = strings.TrimPrefix(ver, "v")
	}
}

// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package vue

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// GoldenCoder loads committed language-core goldens for .vue fixtures.
// It does not invoke Node or QuickJS.
type GoldenCoder struct {
	GoldenDir string
}

type goldenMeta struct {
	EmbeddedID string        `json:"embeddedId"`
	ScriptKind string        `json:"scriptKind"`
	Mappings   []SpanMapping `json:"mappings"`
}

// NewGoldenCoder returns a Coder that reads service scripts from goldenDir.
func NewGoldenCoder(goldenDir string) *GoldenCoder {
	return &GoldenCoder{GoldenDir: goldenDir}
}

// CreateServiceScript loads <basename>.service.ts and <basename>.mappings.json
// from GoldenDir. The path argument may be any absolute/relative .vue path;
// only the base name is used for lookup.
func (c *GoldenCoder) CreateServiceScript(path, source string, _ CodegenOptions) (ServiceScript, error) {
	_ = source // golden is authoritative for PR-3; content hash matching lands with QuickJS.
	if c == nil || strings.TrimSpace(c.GoldenDir) == "" {
		return ServiceScript{}, fmt.Errorf("vue: GoldenCoder.GoldenDir is required")
	}
	base := filepath.Base(path)
	if base == "" || base == "." || base == string(filepath.Separator) {
		return ServiceScript{}, fmt.Errorf("vue: invalid vue path %q", path)
	}
	servicePath := filepath.Join(c.GoldenDir, base+".service.ts")
	metaPath := filepath.Join(c.GoldenDir, base+".mappings.json")
	content, err := os.ReadFile(servicePath)
	if err != nil {
		return ServiceScript{}, fmt.Errorf("vue: read golden %s: %w", servicePath, err)
	}
	metaBytes, err := os.ReadFile(metaPath)
	if err != nil {
		return ServiceScript{}, fmt.Errorf("vue: read golden %s: %w", metaPath, err)
	}
	var meta goldenMeta
	if err := json.Unmarshal(metaBytes, &meta); err != nil {
		return ServiceScript{}, fmt.Errorf("vue: parse golden %s: %w", metaPath, err)
	}
	return ServiceScript{
		EmbeddedID: meta.EmbeddedID,
		ScriptKind: meta.ScriptKind,
		Content:    string(content),
		Mappings:   meta.Mappings,
	}, nil
}

// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package vue

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// GoldenCoder loads committed language-core goldens for .vue fixtures.
// It is fixture-oriented (basename lookup) and verifies sourceSHA256 when
// source is non-empty. It does not invoke Node or QuickJS.
type GoldenCoder struct {
	GoldenDir string
}

type goldenMeta struct {
	EmbeddedID   string        `json:"embeddedId"`
	ScriptKind   string        `json:"scriptKind"`
	SourceSHA256 string        `json:"sourceSHA256"`
	Mappings     []SpanMapping `json:"mappings"`
}

// NewGoldenCoder returns a Coder that reads service scripts from goldenDir.
func NewGoldenCoder(goldenDir string) *GoldenCoder {
	return &GoldenCoder{GoldenDir: goldenDir}
}

// CreateServiceScript loads <basename>.service.txt and <basename>.mappings.json
// from GoldenDir. Lookup is by basename (fixture goldens). The .txt suffix keeps
// language-core output out of JS/TS static analysis. When source is non-empty
// it must match the golden sourceSHA256.
func (c *GoldenCoder) CreateServiceScript(path, source string, _ CodegenOptions) (ServiceScript, error) {
	if c == nil || strings.TrimSpace(c.GoldenDir) == "" {
		return ServiceScript{}, fmt.Errorf("vue: GoldenCoder.GoldenDir is required")
	}
	base := filepath.Base(path)
	if base == "" || base == "." || base == string(filepath.Separator) {
		return ServiceScript{}, fmt.Errorf("vue: invalid vue path %q", path)
	}
	servicePath := filepath.Join(c.GoldenDir, base+".service.txt")
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
	if source != "" && meta.SourceSHA256 != "" {
		sum := sha256.Sum256([]byte(source))
		got := hex.EncodeToString(sum[:])
		if got != meta.SourceSHA256 {
			return ServiceScript{}, fmt.Errorf("vue: source SHA-256 mismatch for %s (got %s want %s); refresh goldens or use a real Coder", base, got, meta.SourceSHA256)
		}
	}
	return ServiceScript{
		EmbeddedID:    meta.EmbeddedID,
		ScriptKind:    meta.ScriptKind,
		Content:       string(content),
		SourceContent: source,
		Mappings:      meta.Mappings,
	}, nil
}

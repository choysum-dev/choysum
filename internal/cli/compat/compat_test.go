// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package compat

import (
	"context"
	"reflect"
	"strings"
	"testing"

	sourceregistry "github.com/choysum-dev/choysum/internal/module/origin/registry"
)

func TestNormalizeAndParseCLICompatVersion(t *testing.T) {
	t.Run("normalize", func(t *testing.T) {
		tests := []struct {
			name   string
			input  string
			want   string
			wantOK bool
		}{
			{name: "trim and add prefix", input: " 1.7.0 ", want: "v1.7.0", wantOK: true},
			{name: "keep prefixed", input: "v1.7.0", want: "v1.7.0", wantOK: true},
			{name: "invalid", input: "foo", want: "", wantOK: false},
			{name: "empty", input: "   ", want: "", wantOK: false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, ok := NormalizeCLICompatVersion(tt.input)
				if got != tt.want || ok != tt.wantOK {
					t.Fatalf("NormalizeCLICompatVersion(%q) = (%q,%v), want (%q,%v)", tt.input, got, ok, tt.want, tt.wantOK)
				}
			})
		}
	})

	t.Run("parse rejects invalid", func(t *testing.T) {
		if _, err := ParseCLICompatVersion("not-semver"); err == nil || !strings.Contains(err.Error(), "ERR_CLI_COMPAT_VERSION_INVALID") {
			t.Fatalf("ParseCLICompatVersion(invalid) error = %v, want invalid-version error", err)
		}
	})
}

func TestResolveCLICompatVersion(t *testing.T) {
	t.Run("flag takes precedence", func(t *testing.T) {
		resolved, err := ResolveCLICompatVersion("1.7.0", "v1.8.0", "v1.6.0")
		if err != nil {
			t.Fatalf("ResolveCLICompatVersion() error = %v", err)
		}
		if resolved.Version != "v1.7.0" || resolved.Source != "flag" {
			t.Fatalf("ResolveCLICompatVersion() = %#v, want flag v1.7.0", resolved)
		}
	})

	t.Run("env used when flag missing", func(t *testing.T) {
		resolved, err := ResolveCLICompatVersion("", "v1.8.0", "1.6.0")
		if err != nil {
			t.Fatalf("ResolveCLICompatVersion() error = %v", err)
		}
		if resolved.Version != "v1.6.0" || resolved.Source != "env" {
			t.Fatalf("ResolveCLICompatVersion() = %#v, want env v1.6.0", resolved)
		}
	})

	t.Run("runtime used when flag env missing", func(t *testing.T) {
		resolved, err := ResolveCLICompatVersion("", "1.8.0", "")
		if err != nil {
			t.Fatalf("ResolveCLICompatVersion() error = %v", err)
		}
		if resolved.Version != "v1.8.0" || resolved.Source != "runtime" {
			t.Fatalf("ResolveCLICompatVersion() = %#v, want runtime v1.8.0", resolved)
		}
	})

	t.Run("returns empty when unresolved", func(t *testing.T) {
		resolved, err := ResolveCLICompatVersion("", "invalid", "")
		if err != nil {
			t.Fatalf("ResolveCLICompatVersion() error = %v", err)
		}
		if resolved != (ResolvedCLICompatVersion{}) {
			t.Fatalf("ResolveCLICompatVersion() = %#v, want empty", resolved)
		}
	})

	t.Run("invalid flag fails", func(t *testing.T) {
		if _, err := ResolveCLICompatVersion("bad", "v1.8.0", ""); err == nil || !strings.Contains(err.Error(), "ERR_CLI_COMPAT_VERSION_INVALID") {
			t.Fatalf("ResolveCLICompatVersion(flag invalid) error = %v, want invalid-version error", err)
		}
	})

	t.Run("invalid env fails", func(t *testing.T) {
		if _, err := ResolveCLICompatVersion("", "v1.8.0", "bad"); err == nil || !strings.Contains(err.Error(), "ERR_CLI_COMPAT_VERSION_INVALID") {
			t.Fatalf("ResolveCLICompatVersion(env invalid) error = %v, want invalid-version error", err)
		}
	})
}

func TestCatalogVersionHelpers(t *testing.T) {
	versions := CatalogCandidateVersions(&sourceregistry.CatalogModule{Versions: []string{" v1.0.0 ", "", "v1.2.0"}})
	if !reflect.DeepEqual(versions, []string{"v1.0.0", "v1.2.0"}) {
		t.Fatalf("CatalogCandidateVersions() = %#v", versions)
	}

	latestOnly := CatalogCandidateVersions(&sourceregistry.CatalogModule{LatestVersion: " v2.0.0 "})
	if !reflect.DeepEqual(latestOnly, []string{"v2.0.0"}) {
		t.Fatalf("CatalogCandidateVersions(latest only) = %#v", latestOnly)
	}

	if !ContainsCatalogVersion([]string{"v1.2.3"}, "1.2.3") {
		t.Fatal("ContainsCatalogVersion() should match equivalent semver with or without v prefix")
	}
	if !ContainsCatalogVersion([]string{"snapshot"}, "snapshot") {
		t.Fatal("ContainsCatalogVersion() should match non-semver values by exact string")
	}
	if ContainsCatalogVersion([]string{"v1.2.3"}, "") {
		t.Fatal("ContainsCatalogVersion() should reject empty target")
	}
}

func TestCompatibleCatalogVersionsAndSelection(t *testing.T) {
	t.Run("nil module", func(t *testing.T) {
		if _, err := CompatibleCatalogVersions(nil, "v1.0.0"); err == nil || !strings.Contains(err.Error(), "remote module is nil") {
			t.Fatalf("CompatibleCatalogVersions(nil) error = %v, want nil-module error", err)
		}
	})

	t.Run("invalid cli version", func(t *testing.T) {
		item := &sourceregistry.CatalogModule{Name: "demo", Versions: []string{"v1.0.0"}, VersionCLIRanges: map[string]string{"v1.0.0": ">=1.0.0 <2.0.0"}}
		if _, err := CompatibleCatalogVersions(item, "bad"); err == nil || !strings.Contains(err.Error(), "ERR_CLI_COMPAT_VERSION_INVALID") {
			t.Fatalf("CompatibleCatalogVersions(invalid cli) error = %v, want invalid-version error", err)
		}
	})

	t.Run("missing range", func(t *testing.T) {
		item := &sourceregistry.CatalogModule{Versions: []string{"v1.0.0"}}
		if _, err := CompatibleCatalogVersions(item, "v1.0.0"); err == nil || !strings.Contains(err.Error(), "ERR_MODULE_CLI_RANGE_MISSING") {
			t.Fatalf("CompatibleCatalogVersions(missing range) error = %v, want missing-range error", err)
		}
	})

	t.Run("invalid range", func(t *testing.T) {
		item := &sourceregistry.CatalogModule{Name: "demo", Versions: []string{"v1.0.0"}, VersionCLIRanges: map[string]string{"v1.0.0": "not-a-range"}}
		if _, err := CompatibleCatalogVersions(item, "v1.0.0"); err == nil || !strings.Contains(err.Error(), "ERR_MODULE_CLI_RANGE_INVALID") {
			t.Fatalf("CompatibleCatalogVersions(invalid range) error = %v, want invalid-range error", err)
		}
	})

	t.Run("no compatible version", func(t *testing.T) {
		item := &sourceregistry.CatalogModule{Name: "demo", Versions: []string{"v2.0.0"}, VersionCLIRanges: map[string]string{"v2.0.0": ">=2.0.0 <3.0.0"}}
		if _, err := CompatibleCatalogVersions(item, "v1.0.0"); err == nil || !strings.Contains(err.Error(), "ERR_MODULE_NO_COMPATIBLE_VERSION") {
			t.Fatalf("CompatibleCatalogVersions(no compatible) error = %v, want no-compatible error", err)
		}
	})

	t.Run("returns compatible versions and latest", func(t *testing.T) {
		item := &sourceregistry.CatalogModule{
			Name:     "demo",
			Versions: []string{"v1.0.0", "v1.3.0", "v2.0.0"},
			VersionCLIRanges: map[string]string{
				"v1.0.0": ">=1.0.0 <2.0.0",
				"v1.3.0": ">=1.2.0 <2.0.0",
				"v2.0.0": ">=2.0.0 <3.0.0",
			},
		}

		compatible, err := CompatibleCatalogVersions(item, "v1.5.0")
		if err != nil {
			t.Fatalf("CompatibleCatalogVersions() error = %v", err)
		}
		if !reflect.DeepEqual(compatible, []string{"v1.0.0", "v1.3.0"}) {
			t.Fatalf("CompatibleCatalogVersions() = %#v, want [v1.0.0 v1.3.0]", compatible)
		}

		latest, err := SelectLatestCompatibleCatalogVersion(item, "v1.5.0")
		if err != nil {
			t.Fatalf("SelectLatestCompatibleCatalogVersion() error = %v", err)
		}
		if latest != "v1.3.0" {
			t.Fatalf("SelectLatestCompatibleCatalogVersion() = %q, want %q", latest, "v1.3.0")
		}
	})

	t.Run("selects latest compatible from unsorted versions", func(t *testing.T) {
		item := &sourceregistry.CatalogModule{
			Name:     "demo",
			Versions: []string{"v1.9.0", "v1.10.0", "v1.2.0"},
			VersionCLIRanges: map[string]string{
				"v1.9.0":  ">=1.0.0 <2.0.0",
				"v1.10.0": ">=1.0.0 <2.0.0",
				"v1.2.0":  ">=1.0.0 <2.0.0",
			},
		}

		latest, err := SelectLatestCompatibleCatalogVersion(item, "v1.5.0")
		if err != nil {
			t.Fatalf("SelectLatestCompatibleCatalogVersion() error = %v", err)
		}
		if latest != "v1.10.0" {
			t.Fatalf("SelectLatestCompatibleCatalogVersion() = %q, want %q", latest, "v1.10.0")
		}

		filtered, err := FilterCatalogModuleByCompatibility(item, "v1.5.0")
		if err != nil {
			t.Fatalf("FilterCatalogModuleByCompatibility() error = %v", err)
		}
		if filtered.LatestVersion != "v1.10.0" {
			t.Fatalf("FilterCatalogModuleByCompatibility().LatestVersion = %q, want %q", filtered.LatestVersion, "v1.10.0")
		}
	})
}

func TestCompatHelpersAdditionalCoverage(t *testing.T) {
	t.Run("resolve compatible latest rejects nil runtime scope", func(t *testing.T) {
		if _, err := ResolveCompatibleRegistryLatestVersion(context.Background(), nil, "https://example.com/v1/index.json", "demo", "v1.0.0"); err == nil || !strings.Contains(err.Error(), "runtime scope is nil") {
			t.Fatalf("ResolveCompatibleRegistryLatestVersion(nil scope) error = %v, want nil-scope error", err)
		}
	})

	t.Run("registry origin binding rejects nil runtime scope", func(t *testing.T) {
		if _, err := HasRegistryOriginBinding(nil, "/tmp/.choysum", "demo"); err == nil || !strings.Contains(err.Error(), "runtime scope is nil") {
			t.Fatalf("HasRegistryOriginBinding(nil scope) error = %v, want nil-scope error", err)
		}
	})

	t.Run("latestCatalogVersion keeps first non-semver fallback", func(t *testing.T) {
		got := latestCatalogVersion([]string{"snapshot-a", "snapshot-b", "snapshot-c"})
		if got != "snapshot-a" {
			t.Fatalf("latestCatalogVersion(non-semver) = %q, want %q", got, "snapshot-a")
		}
	})

	t.Run("latestCatalogVersion prefers semver over non-semver fallback", func(t *testing.T) {
		got := latestCatalogVersion([]string{"snapshot-a", "v1.2.0", "snapshot-b", "v1.1.0"})
		if got != "v1.2.0" {
			t.Fatalf("latestCatalogVersion(mixed) = %q, want %q", got, "v1.2.0")
		}
	})

	t.Run("catalog candidates for nil module", func(t *testing.T) {
		if candidates := CatalogCandidateVersions(nil); candidates != nil {
			t.Fatalf("CatalogCandidateVersions(nil) = %#v, want nil", candidates)
		}
	})

	t.Run("containsCatalogVersion non-semver target mismatch", func(t *testing.T) {
		if ContainsCatalogVersion([]string{"v1.2.3"}, "snapshot") {
			t.Fatal("ContainsCatalogVersion() should return false for non-semver mismatch")
		}
	})

	t.Run("filterCatalogModuleByCompatibility nil module", func(t *testing.T) {
		if _, err := FilterCatalogModuleByCompatibility(nil, "v1.0.0"); err == nil || !strings.Contains(err.Error(), "remote module is nil") {
			t.Fatalf("FilterCatalogModuleByCompatibility(nil) error = %v, want nil-module error", err)
		}
	})
}

func TestCompatFilterSkippedWarning(t *testing.T) {
	warning := CompatFilterSkippedWarning()
	if !strings.Contains(warning, "WARN_CLI_COMPAT_FILTER_SKIPPED") {
		t.Fatalf("CompatFilterSkippedWarning() = %q, want warning code", warning)
	}
}

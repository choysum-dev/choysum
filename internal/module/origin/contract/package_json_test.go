// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package contract

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDecodePackageJSON(t *testing.T) {
	t.Parallel()

	raw := []byte(`{
		"name":"@acme/choysum-sale",
		"version":"0.1.0",
		"peerDependencies":{"vue":"^3.4.29"},
		"choysum":{
			"moduleName":"sale",
			"application":"sale",
			"depends":["core"],
			"entryPoints":{"web":"./web/index.ts"}
		}
	}`)

	pkg, err := DecodePackageJSON(raw)
	if err != nil {
		t.Fatalf("DecodePackageJSON() error = %v", err)
	}
	if pkg.Name != "@acme/choysum-sale" {
		t.Fatalf("name = %q, want %q", pkg.Name, "@acme/choysum-sale")
	}
	if pkg.Choysum.ModuleName != "sale" {
		t.Fatalf("moduleName = %q, want %q", pkg.Choysum.ModuleName, "sale")
	}
}

func TestNormalizeVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "no v prefix", input: "0.1.0", want: "v0.1.0"},
		{name: "with v prefix", input: "v1.2.3", want: "v1.2.3"},
		{name: "with pre release", input: "1.0.0-rc.1", want: "v1.0.0-rc.1"},
		{name: "invalid", input: "1", wantErr: true},
		{name: "empty", input: "", wantErr: true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := NormalizeVersion(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("NormalizeVersion(%q) expected error", tc.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("NormalizeVersion(%q) error = %v", tc.input, err)
			}
			if got != tc.want {
				t.Fatalf("NormalizeVersion(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestValidatePackageJSON(t *testing.T) {
	t.Parallel()

	valid := &PackageJSON{
		Name:    "@acme/choysum-sale",
		Version: "0.1.0",
		Choysum: ChoysumMeta{
			ModuleName:  "sale",
			Application: "sale",
			Depends:     []string{"core"},
			EntryPoints: map[string]string{"web": "./web/index.ts"},
		},
	}
	if err := ValidatePackageJSON(valid); err != nil {
		t.Fatalf("ValidatePackageJSON(valid) error = %v", err)
	}

	missingName := *valid
	missingName.Name = ""
	if err := ValidatePackageJSON(&missingName); err == nil || !strings.Contains(err.Error(), "package name is required") {
		t.Fatalf("missing name expected error, got %v", err)
	}

	missingModuleName := *valid
	missingModuleName.Choysum.ModuleName = ""
	if err := ValidatePackageJSON(&missingModuleName); err == nil || !strings.Contains(err.Error(), "choysum.moduleName is required") {
		t.Fatalf("missing moduleName expected error, got %v", err)
	}

	badEntry := *valid
	badEntry.Choysum.EntryPoints = map[string]string{"worker": "./worker/index.ts"}
	if err := ValidatePackageJSON(&badEntry); err == nil || !strings.Contains(err.Error(), "unsupported choysum.entryPoints key") {
		t.Fatalf("bad entry key expected error, got %v", err)
	}

	emptyDep := *valid
	emptyDep.Choysum.Depends = []string{"core", ""}
	if err := ValidatePackageJSON(&emptyDep); err == nil || !strings.Contains(err.Error(), "contains empty module name") {
		t.Fatalf("empty depends expected error, got %v", err)
	}
}

func TestBuildExternalDependencies(t *testing.T) {
	t.Parallel()

	deps := BuildExternalDependencies(map[string]string{"wkhtmltopdf": ">=0.12.0"})
	if _, ok := deps["node_module"]; ok {
		t.Fatalf("node_module channel must not exist")
	}
	binary, ok := deps["binary"]
	if !ok {
		t.Fatalf("binary channel missing")
	}
	if binary["wkhtmltopdf"] != ">=0.12.0" {
		t.Fatalf("binary dependency mismatch: %#v", binary)
	}

	raw, err := json.Marshal(deps)
	if err != nil {
		t.Fatalf("marshal deps: %v", err)
	}
	if strings.Contains(string(raw), "node_module") {
		t.Fatalf("marshaled deps contains unexpected node_module: %s", string(raw))
	}
}

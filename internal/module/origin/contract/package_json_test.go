// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package contract

import (
	"encoding/json"
	"reflect"
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
			"cli":">=0.0.0-0 <0.0.0",
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
	if pkg.Choysum.CLI != ">=0.0.0-0 <0.0.0" {
		t.Fatalf("cli = %q, want %q", pkg.Choysum.CLI, ">=0.0.0-0 <0.0.0")
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
			CLI:         ">=0.0.0-0 <0.0.0",
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

	invalidCLI := *valid
	invalidCLI.Choysum.CLI = ">=0.0.0 <"
	if err := ValidatePackageJSON(&invalidCLI); err == nil || !strings.Contains(err.Error(), "invalid choysum.cli range") {
		t.Fatalf("invalid cli range expected error, got %v", err)
	}

	badEntry := *valid
	badEntry.Choysum.EntryPoints = map[string]string{"worker": "./worker/index.ts"}
	if err := ValidatePackageJSON(&badEntry); err == nil || !strings.Contains(err.Error(), "unsupported choysum.entryPoints key") {
		t.Fatalf("bad entry key expected error, got %v", err)
	}

	spacePaddedEntry := *valid
	spacePaddedEntry.Choysum.EntryPoints = map[string]string{" service ": "./service/index.ts"}
	if err := ValidatePackageJSON(&spacePaddedEntry); err == nil || !strings.Contains(err.Error(), "unsupported choysum.entryPoints key") {
		t.Fatalf("space padded entry key expected error, got %v", err)
	}

	emptyDep := *valid
	emptyDep.Choysum.Depends = []string{"core", ""}
	if err := ValidatePackageJSON(&emptyDep); err == nil || !strings.Contains(err.Error(), "contains empty module name") {
		t.Fatalf("empty depends expected error, got %v", err)
	}

	absEntry := *valid
	absEntry.Choysum.EntryPoints = map[string]string{"web": "/tmp/web/index.ts"}
	if err := ValidatePackageJSON(&absEntry); err == nil || !strings.Contains(err.Error(), "must be a relative path") {
		t.Fatalf("absolute entry path expected error, got %v", err)
	}

	traversalData := *valid
	traversalData.Choysum.Data = []string{"../secret.json"}
	if err := ValidatePackageJSON(&traversalData); err == nil || !strings.Contains(err.Error(), "cannot contain parent traversal") {
		t.Fatalf("traversal data path expected error, got %v", err)
	}

	windowsDemo := *valid
	windowsDemo.Choysum.Demo = []string{"C:/tmp/demo.json"}
	if err := ValidatePackageJSON(&windowsDemo); err == nil || !strings.Contains(err.Error(), "must be a relative path") {
		t.Fatalf("windows absolute path expected error, got %v", err)
	}

	validPaths := *valid
	validPaths.Choysum.Data = []string{"data/bootstrap.json"}
	validPaths.Choysum.Demo = []string{"demo/demo.json"}
	if err := ValidatePackageJSON(&validPaths); err != nil {
		t.Fatalf("valid paths should pass, got %v", err)
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

func TestParsePackageJSONToIrModule(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		raw := []byte(`{
			"name":"@acme/choysum-sale",
			"version":"0.1.0",
			"description":"Sales module",
			"license":"LGPL-3.0-or-later",
			"author":"Acme",
			"peerDependencies":{"pinia":"^2.1.7","vue":"^3.4.29"},
			"choysum":{
				"moduleName":"sale",
				"application":"sale",
				"cli":">=0.0.0-0 <0.0.0",
				"depends":["core","base"],
				"entryPoints":{"service":"./service/index.ts","web":"./web/index.ts"},
				"data":["data/bootstrap.json"],
				"demo":["demo/demo.json"]
			}
		}`)

		result, err := ParsePackageJSONToIrModule(raw, "/tmp/modules/sale", map[string]string{"wkhtmltopdf": ">=0.12.0"})
		if err != nil {
			t.Fatalf("ParsePackageJSONToIrModule() error = %v", err)
		}
		if result == nil || result.Module == nil {
			t.Fatalf("expected non-nil conversion result")
		}
		if result.Module.Name != "sale" {
			t.Fatalf("module name = %q, want %q", result.Module.Name, "sale")
		}
		if result.Module.Version != "v0.1.0" {
			t.Fatalf("module version = %q, want %q", result.Module.Version, "v0.1.0")
		}
		if result.Module.WebEntryPoint != "./web/index.ts" || result.Module.ServiceEntryPoint != "./service/index.ts" {
			t.Fatalf("unexpected entry points: web=%q service=%q", result.Module.WebEntryPoint, result.Module.ServiceEntryPoint)
		}
		if !reflect.DeepEqual(result.PeerDependencies, map[string]string{"pinia": "^2.1.7", "vue": "^3.4.29"}) {
			t.Fatalf("unexpected peerDependencies: %#v", result.PeerDependencies)
		}

		ext := map[string]map[string]string{}
		if err := json.Unmarshal(result.Module.ExternalDependencies, &ext); err != nil {
			t.Fatalf("unmarshal externalDependencies: %v", err)
		}
		if _, ok := ext["node_module"]; ok {
			t.Fatalf("node_module channel must not exist: %#v", ext)
		}
		if ext["binary"]["wkhtmltopdf"] != ">=0.12.0" {
			t.Fatalf("unexpected binary dependencies: %#v", ext)
		}
	})

	t.Run("type error in depends", func(t *testing.T) {
		t.Parallel()
		raw := []byte(`{
			"name":"@acme/choysum-sale",
			"version":"0.1.0",
			"choysum":{
				"moduleName":"sale",
				"application":"sale",
				"depends":[1]
			}
		}`)
		if _, err := ParsePackageJSONToIrModule(raw, "", nil); err == nil {
			t.Fatalf("expected type error for depends")
		}
	})
}

func TestPackageJSONToIrModuleIdempotent(t *testing.T) {
	t.Parallel()

	pkg := &PackageJSON{
		Name:    "@acme/choysum-sale",
		Version: "0.1.0",
		Choysum: ChoysumMeta{
			ModuleName:  "sale",
			Application: "sale",
			CLI:         ">=0.0.0-0 <0.0.0",
			Depends:     []string{"core", "base"},
			EntryPoints: map[string]string{"service": "./service/index.ts", "web": "./web/index.ts"},
			Data:        []string{"data/bootstrap.json"},
			Demo:        []string{"demo/demo.json"},
		},
		PeerDependencies: map[string]string{"vue": "^3.4.29", "pinia": "^2.1.7"},
	}

	res1, err := PackageJSONToIrModule(pkg, "/tmp/modules/sale", map[string]string{"wkhtmltopdf": ">=0.12.0"})
	if err != nil {
		t.Fatalf("first conversion error = %v", err)
	}
	res2, err := PackageJSONToIrModule(pkg, "/tmp/modules/sale", map[string]string{"wkhtmltopdf": ">=0.12.0"})
	if err != nil {
		t.Fatalf("second conversion error = %v", err)
	}

	if !reflect.DeepEqual(res1, res2) {
		t.Fatalf("conversion result is not idempotent\nres1=%#v\nres2=%#v", res1, res2)
	}
}

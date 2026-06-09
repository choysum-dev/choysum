// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/choysum-dev/choysum/pkg/meta"
)

func TestWebServiceGenerate(t *testing.T) {
	runtimeScope := newGeneratorScope(t)
	if NewWebServiceGenerator(runtimeScope, &meta.IrModule{Name: "base"}) == nil {
		t.Fatal("expected web service generator constructor to return non-nil")
	}
	webServiceDir := t.TempDir()
	serviceResults, err := (&webServiceGenerator{runtimeScope: runtimeScope, module: &meta.IrModule{ApplicationStr: "crm"}, modulesWebDir: webServiceDir}).generate(testApp())
	if err != nil {
		t.Fatalf("web service generate() error = %v", err)
	}
	if len(serviceResults) != 1 || serviceResults[0].Name != "webservice" {
		t.Fatalf("unexpected web service results: %#v", serviceResults)
	}
	serviceContent, err := os.ReadFile(filepath.Join(webServiceDir, "service.ts"))
	if err != nil {
		t.Fatalf("read web service.ts: %v", err)
	}
	if !strings.Contains(string(serviceContent), "CreateWebApiService") || !strings.Contains(string(serviceContent), "CreatePartner") || !strings.Contains(string(serviceContent), "@/core/web/rpc") {
		t.Fatalf("unexpected web service.ts content: %s", string(serviceContent))
	}
}

func TestWebServiceGenerateEmptyApp(t *testing.T) {
	runtimeScope := newGeneratorScope(t)
	results, err := (&webServiceGenerator{runtimeScope: runtimeScope, module: &meta.IrModule{ApplicationStr: "crm"}, modulesWebDir: t.TempDir()}).generate(&meta.IrApplication{Name: "crm"})
	if err != nil {
		t.Fatalf("generate(empty app) error = %v", err)
	}
	if results != nil {
		t.Fatalf("expected nil results for app without models, got %#v", results)
	}
}

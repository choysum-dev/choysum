// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package lifecycle

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestCommitClosures_DoNotCallHookRunners guards against regressing hook JS into
// install/upgrade/uninstall Commit TX bodies (pre_init must stay outside Required).
func TestCommitClosures_DoNotCallHookRunners(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	dir := filepath.Dir(thisFile)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	banned := map[string]bool{
		"runInstallHookPhase":     true,
		"runUpgradeHookPhase":     true,
		"runUninstallHookPhase":   true,
		"runInstallPreInit":       true,
		"uninstallHooksNewRunner": true,
		"upgradeHooksNewRunner":   true,
		"hooksNewRunner":          true,
		"NewRunner":               true,
		"RunPhase":                true,
	}
	fset := token.NewFileSet()
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		path := filepath.Join(dir, name)
		src, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		file, err := parser.ParseFile(fset, path, src, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name == nil || fn.Body == nil {
				continue
			}
			fnName := fn.Name.Name
			if fnName != "commitInstall" && fnName != "commitUpgrade" && fnName != "commitUninstall" {
				continue
			}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				switch fun := call.Fun.(type) {
				case *ast.Ident:
					if banned[fun.Name] {
						t.Errorf("%s calls banned hook helper %q", fnName, fun.Name)
					}
				case *ast.SelectorExpr:
					if fun.Sel != nil && banned[fun.Sel.Name] {
						t.Errorf("%s calls banned hook helper %q", fnName, fun.Sel.Name)
					}
				}
				return true
			})
		}
	}
}

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
	fset := token.NewFileSet()
	for _, name := range []string{"installer.go", "upgrader.go", "uninstaller.go"} {
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
			if !ok || fn.Name == nil {
				continue
			}
			fnName := fn.Name.Name
			if fnName != "commitInstall" && fnName != "commitUpgrade" && fnName != "commitUninstall" {
				continue
			}
			start := fset.Position(fn.Pos()).Offset
			end := fset.Position(fn.End()).Offset
			body := string(src[start:end])
			for _, banned := range []string{
				"runInstallHookPhase(",
				"runUpgradeHookPhase(",
				"runUninstallHookPhase(",
				"hooks.NewRunner(",
				"uninstallHooksNewRunner(",
				"upgradeHooksNewRunner(",
				".RunPhase(",
			} {
				if strings.Contains(body, banned) {
					t.Errorf("%s contains banned hook call %q", fnName, banned)
				}
			}
		}
	}
}

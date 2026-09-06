// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package typecheck

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	testingpathing "github.com/choysum-dev/choysum/internal/testing/tmpdir"
)

func TestTypecheckApp_AdditionalPaths(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	t.Run("requires modules path", func(t *testing.T) {
		err := TypecheckApp(context.Background(), RunOptions{RepoRoot: t.TempDir()}, "auth")
		if err == nil || !strings.Contains(err.Error(), "modules_path is required") {
			t.Fatalf("expected modules path error, got %v", err)
		}
	})

	t.Run("returns context error before doing work", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		err := TypecheckApp(ctx, RunOptions{ModulesPath: t.TempDir(), RepoRoot: t.TempDir()}, "auth")
		if err != context.Canceled {
			t.Fatalf("expected context canceled, got %v", err)
		}
	})

	t.Run("requires app name", func(t *testing.T) {
		err := TypecheckApp(context.Background(), RunOptions{ModulesPath: t.TempDir(), RepoRoot: t.TempDir()}, " ")
		if err == nil || !strings.Contains(err.Error(), "missing app name") {
			t.Fatalf("expected missing app name error, got %v", err)
		}
	})

	t.Run("uses current working directory as repo root when omitted", func(t *testing.T) {
		repoRoot := t.TempDir()
		modulesPath := t.TempDir()
		makeDir(t, filepath.Join(modulesPath, "auth", "service"))
		writeFile(t, filepath.Join(modulesPath, "auth", "service", "index.ts"), "export const auth = 1\n")

		originalWD, err := os.Getwd()
		if err != nil {
			t.Fatalf("Getwd returned error: %v", err)
		}
		if err := os.Chdir(repoRoot); err != nil {
			t.Fatalf("Chdir(%q): %v", repoRoot, err)
		}
		defer func() { _ = os.Chdir(originalWD) }()

		err = TypecheckApp(context.Background(), RunOptions{ModulesPath: modulesPath}, "auth")
		if err != nil {
			t.Fatalf("TypecheckApp returned error: %v", err)
		}
	})

	t.Run("accepts relative modules path", func(t *testing.T) {
		repoRoot := t.TempDir()
		modulesPath := filepath.Join(repoRoot, "modules")
		makeDir(t, filepath.Join(modulesPath, "auth", "service"))
		writeFile(t, filepath.Join(modulesPath, "auth", "service", "index.ts"), "export const auth = 1\n")

		originalWD, err := os.Getwd()
		if err != nil {
			t.Fatalf("Getwd returned error: %v", err)
		}
		if err := os.Chdir(repoRoot); err != nil {
			t.Fatalf("Chdir(%q): %v", repoRoot, err)
		}
		defer func() { _ = os.Chdir(originalWD) }()

		err = TypecheckApp(context.Background(), RunOptions{
			ModulesPath: "modules",
			RepoRoot:    repoRoot,
			Stderr:      &strings.Builder{},
		}, "auth")
		if err != nil {
			t.Fatalf("TypecheckApp(relative modules path) returned error: %v", err)
		}
	})

	t.Run("keeps diagnostics dump when keep is enabled", func(t *testing.T) {
		repoRoot := t.TempDir()
		modulesPath := t.TempDir()
		makeDir(t, filepath.Join(modulesPath, "auth", "service"))
		writeFile(t, filepath.Join(modulesPath, "auth", "service", "index.ts"), "export const auth = 1\n")
		tmpPath := t.TempDir()

		var stderr strings.Builder
		err := TypecheckApp(context.Background(), RunOptions{
			ModulesPath: modulesPath,
			RepoRoot:    repoRoot,
			TmpPath:     tmpPath,
			Keep:        true,
			Stderr:      &stderr,
		}, "auth")
		if err != nil {
			t.Fatalf("TypecheckApp returned error: %v", err)
		}
		if !strings.Contains(stderr.String(), "kept artifacts dir:") {
			t.Fatalf("expected keep notice, got %q", stderr.String())
		}
		wantRoot, err := testingpathing.ResolveTestingTmpDir(repoRoot, tmpPath, "typecheck")
		if err != nil {
			t.Fatalf("ResolveTestingTmpDir: %v", err)
		}
		keepDir := filepath.Join(wantRoot, sanitizeAppToken("auth"))
		if _, err := os.Stat(filepath.Join(keepDir, "diagnostics.txt")); err != nil {
			t.Fatalf("expected diagnostics dump under keep dir: %v", err)
		}
	})

	t.Run("wraps diagnostic failures with TS codes on stderr", func(t *testing.T) {
		repoRoot := t.TempDir()
		modulesPath := t.TempDir()
		makeDir(t, filepath.Join(modulesPath, "auth", "service"))
		writeFile(t, filepath.Join(modulesPath, "auth", "service", "bad.ts"), "const x: number = 'nope'\n")

		var stderr strings.Builder
		err := TypecheckApp(context.Background(), RunOptions{
			ModulesPath: modulesPath,
			RepoRoot:    repoRoot,
			TmpPath:     t.TempDir(),
			Stderr:      &stderr,
		}, "auth")
		if err == nil || !strings.Contains(err.Error(), "typecheck failed for auth") {
			t.Fatalf("expected wrapped error, got %v", err)
		}
		if !strings.Contains(stderr.String(), "TS") {
			t.Fatalf("expected TS diagnostics on stderr, got %q", stderr.String())
		}
	})

	t.Run("prints soft guidance when mapped type assets are missing", func(t *testing.T) {
		repoRoot := t.TempDir()
		modulesPath := t.TempDir()
		makeDir(t, filepath.Join(modulesPath, "auth", "service"))
		writeFile(t, filepath.Join(modulesPath, "auth", "service", "index.ts"), "export const auth = 1\n")
		writeFile(t, filepath.Join(modulesPath, "auth", "package.json"), `{"dependencies":{"lodash":"^4.17.21"}}`)
		writeFile(t, filepath.Join(modulesPath, "tsconfig.json"), `{
		  "compilerOptions": {
		    "paths": {
		      "lodash": ["./.choysum/pkg/types/lodash@4.17.21.d.ts"]
		    }
		  }
		}`)

		var stderr strings.Builder
		err := TypecheckApp(context.Background(), RunOptions{
			ModulesPath: modulesPath,
			RepoRoot:    repoRoot,
			TmpPath:     t.TempDir(),
			Stderr:      &stderr,
		}, "auth")
		if err != nil {
			t.Fatalf("TypecheckApp returned error: %v", err)
		}
		got := stderr.String()
		if !strings.Contains(got, "Warning: type declarations may be incomplete") {
			t.Fatalf("expected soft precheck warning, got %q", got)
		}
		if !strings.Contains(got, "recommended action:\n  go run . type-fetch auth") {
			t.Fatalf("expected type-fetch command hint, got %q", got)
		}
	})

	t.Run("suppresses soft guidance when tsconfig paths have no matching entry", func(t *testing.T) {
		repoRoot := t.TempDir()
		modulesPath := t.TempDir()
		makeDir(t, filepath.Join(modulesPath, "auth", "service"))
		writeFile(t, filepath.Join(modulesPath, "auth", "service", "index.ts"), "export const auth = 1\n")
		writeFile(t, filepath.Join(modulesPath, "auth", "package.json"), `{"dependencies":{"lodash":"^4.17.21"}}`)
		writeFile(t, filepath.Join(modulesPath, "tsconfig.json"), `{
		  "compilerOptions": {
		    "paths": {
		      "moment": ["./.choysum/pkg/types/moment@2.29.4.d.ts"]
		    }
		  }
		}`)

		var stderr strings.Builder
		err := TypecheckApp(context.Background(), RunOptions{
			ModulesPath: modulesPath,
			RepoRoot:    repoRoot,
			TmpPath:     t.TempDir(),
			Stderr:      &stderr,
		}, "auth")
		if err != nil {
			t.Fatalf("TypecheckApp returned error: %v", err)
		}
		if strings.Contains(stderr.String(), "Warning: type declarations may be incomplete") {
			t.Fatalf("did not expect soft precheck warning, got %q", stderr.String())
		}
	})

	t.Run("suppresses soft guidance when wildcard-mapped type asset exists", func(t *testing.T) {
		repoRoot := t.TempDir()
		modulesPath := t.TempDir()
		makeDir(t, filepath.Join(modulesPath, "auth", "service"))
		writeFile(t, filepath.Join(modulesPath, "auth", "service", "index.ts"), "export const auth = 1\n")
		writeFile(t, filepath.Join(modulesPath, "auth", "package.json"), `{"dependencies":{"lodash":"^4.17.21"}}`)

		cachedTypePath := filepath.Join(modulesPath, ".choysum", "pkg", "types", "lodash@4.17.21.d.ts")
		makeDir(t, filepath.Dir(cachedTypePath))
		writeFile(t, cachedTypePath, "declare const lodash: any\nexport default lodash\n")
		writeFile(t, filepath.Join(modulesPath, "tsconfig.json"), `{
		  "compilerOptions": {
		    "paths": {
		      "lodash": ["./.choysum/pkg/types/lodash@*.d.ts"]
		    }
		  }
		}`)

		var stderr strings.Builder
		err := TypecheckApp(context.Background(), RunOptions{
			ModulesPath: modulesPath,
			RepoRoot:    repoRoot,
			TmpPath:     t.TempDir(),
			Stderr:      &stderr,
		}, "auth")
		if err != nil {
			t.Fatalf("TypecheckApp returned error: %v", err)
		}
		if strings.Contains(stderr.String(), "Warning: type declarations may be incomplete") {
			t.Fatalf("did not expect soft precheck warning, got %q", stderr.String())
		}
	})

	t.Run("returns dependency collection error during soft precheck", func(t *testing.T) {
		repoRoot := t.TempDir()
		modulesPath := t.TempDir()
		makeDir(t, filepath.Join(modulesPath, "auth", "service"))
		writeFile(t, filepath.Join(modulesPath, "auth", "service", "index.ts"), "export const auth = 1\n")
		writeFile(t, filepath.Join(modulesPath, "auth", "package.json"), `{"dependencies":`)

		err := TypecheckApp(context.Background(), RunOptions{
			ModulesPath: modulesPath,
			RepoRoot:    repoRoot,
			TmpPath:     t.TempDir(),
		}, "auth")
		if err == nil || !strings.Contains(err.Error(), "typecheck: collect module dependencies:") {
			t.Fatalf("expected dependency collection error, got %v", err)
		}
	})

	t.Run("parses tsconfig paths from JSONC with comments and trailing commas", func(t *testing.T) {
		repoRoot := t.TempDir()
		modulesPath := t.TempDir()
		makeDir(t, filepath.Join(modulesPath, "auth", "service"))
		writeFile(t, filepath.Join(modulesPath, "auth", "service", "index.ts"), "export const auth = 1\n")
		writeFile(t, filepath.Join(modulesPath, "auth", "package.json"), `{"dependencies":{"lodash":"^4.17.21"}}`)
		writeFile(t, filepath.Join(modulesPath, "tsconfig.json"), `{
		  // comment
		  "compilerOptions": {
		    "paths": {
		      "lodash": ["./.choysum/pkg/types/lodash@4.17.21.d.ts"],
		    },
		  },
		}`)

		var stderr strings.Builder
		err := TypecheckApp(context.Background(), RunOptions{
			ModulesPath: modulesPath,
			RepoRoot:    repoRoot,
			TmpPath:     t.TempDir(),
			Stderr:      &stderr,
		}, "auth")
		if err != nil {
			t.Fatalf("TypecheckApp returned error: %v", err)
		}
		if !strings.Contains(stderr.String(), "Warning: type declarations may be incomplete") {
			t.Fatalf("expected soft precheck warning for JSONC paths, got %q", stderr.String())
		}
	})

	t.Run("suppresses soft guidance when tsconfig paths already point to cached type files", func(t *testing.T) {
		repoRoot := t.TempDir()
		modulesPath := t.TempDir()
		makeDir(t, filepath.Join(modulesPath, "auth", "service"))
		writeFile(t, filepath.Join(modulesPath, "auth", "service", "index.ts"), "export const auth = 1\n")
		writeFile(t, filepath.Join(modulesPath, "auth", "package.json"), `{"dependencies":{"lodash":"^4.17.21"}}`)
		cachedTypePath := filepath.Join(modulesPath, ".choysum", "pkg", "types", "lodash@4.17.21.d.ts")
		makeDir(t, filepath.Dir(cachedTypePath))
		writeFile(t, cachedTypePath, "declare const lodash: any\nexport default lodash\n")
		writeFile(t, filepath.Join(modulesPath, "tsconfig.json"), `{
		  "compilerOptions": {
		    "paths": {
		      "lodash": ["./.choysum/pkg/types/lodash@4.17.21.d.ts"]
		    }
		  }
		}`)

		var stderr strings.Builder
		err := TypecheckApp(context.Background(), RunOptions{
			ModulesPath: modulesPath,
			RepoRoot:    repoRoot,
			TmpPath:     t.TempDir(),
			Stderr:      &stderr,
		}, "auth")
		if err != nil {
			t.Fatalf("TypecheckApp returned error: %v", err)
		}
		if strings.Contains(stderr.String(), "Warning: type declarations may be incomplete") {
			t.Fatalf("did not expect soft precheck warning, got %q", stderr.String())
		}
	})

	t.Run("appends type-fetch guidance when diagnostics indicate missing modules", func(t *testing.T) {
		repoRoot := t.TempDir()
		modulesPath := t.TempDir()
		makeDir(t, filepath.Join(modulesPath, "auth", "service"))
		writeFile(t, filepath.Join(modulesPath, "auth", "service", "index.ts"),
			"import x from 'missing-lib-xyz'\nexport const auth = x\n")

		err := TypecheckApp(context.Background(), RunOptions{
			ModulesPath: modulesPath,
			RepoRoot:    repoRoot,
			TmpPath:     t.TempDir(),
			Stderr:      &strings.Builder{},
		}, "auth")
		if err == nil {
			t.Fatal("expected typecheck failure")
		}
		if !strings.Contains(err.Error(), "typecheck failed for auth") {
			t.Fatalf("expected wrapped typecheck failure, got %v", err)
		}
		if !strings.Contains(err.Error(), "go run . type-fetch auth") {
			t.Fatalf("expected type-fetch guidance in error, got %v", err)
		}
	})
}

func TestTypecheckApp_ServiceImportBoundary(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	repoRoot := t.TempDir()
	modulesPath := t.TempDir()

	makeDir(t, filepath.Join(modulesPath, "partner"))
	writeFile(t, filepath.Join(modulesPath, "partner", "package.json"), `{
  "name": "@test/partner",
  "version": "0.0.0-test",
  "choysum": {"moduleName":"partner","application":"partner"}
}`)
	makeDir(t, filepath.Join(modulesPath, "auth"))
	writeFile(t, filepath.Join(modulesPath, "auth", "package.json"), `{
  "name": "@test/auth",
  "version": "0.0.0-test",
  "choysum": {"moduleName":"auth","application":"auth"}
}`)
	makeDir(t, filepath.Join(modulesPath, "partner", "service", "models"))
	writeFile(
		t,
		filepath.Join(modulesPath, "partner", "service", "models", "partner.ts"),
		"import Role from '@/auth/service/models/role';\nexport default {};\n",
	)

	err := TypecheckApp(context.Background(), RunOptions{
		ModulesPath: modulesPath,
		RepoRoot:    repoRoot,
		TmpPath:     t.TempDir(),
	}, "partner")
	if err == nil || !strings.Contains(err.Error(), "typecheck: cross-app service import boundary violation") {
		t.Fatalf("TypecheckApp() error = %v, want import boundary failure", err)
	}
	if !strings.Contains(err.Error(), "partner -> auth") {
		t.Fatalf("TypecheckApp() error = %v, want partner -> auth edge", err)
	}
}

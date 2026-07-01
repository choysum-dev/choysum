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
		repoRoot := t.TempDir()
		modulesPath := t.TempDir()
		npmPath, _, _ := makeFakeTypecheckTooling(t, repoRoot, "exit 0\n")
		err := TypecheckApp(context.Background(), RunOptions{ModulesPath: modulesPath, RepoRoot: repoRoot, NpmPath: npmPath}, " ")
		if err == nil || !strings.Contains(err.Error(), "missing app name") {
			t.Fatalf("expected missing app name error, got %v", err)
		}
	})

	t.Run("uses current working directory as repo root when omitted", func(t *testing.T) {
		repoRoot := t.TempDir()
		modulesPath := t.TempDir()
		makeDir(t, filepath.Join(modulesPath, "auth", "service"))
		writeFile(t, filepath.Join(modulesPath, "auth", "service", "index.ts"), "export const auth = 1\n")
		npmPath, _, _ := makeFakeTypecheckTooling(t, repoRoot, "exit 0\n")

		originalWD, err := os.Getwd()
		if err != nil {
			t.Fatalf("Getwd returned error: %v", err)
		}
		if err := os.Chdir(repoRoot); err != nil {
			t.Fatalf("Chdir(%q): %v", repoRoot, err)
		}
		defer func() {
			_ = os.Chdir(originalWD)
		}()

		err = TypecheckApp(context.Background(), RunOptions{ModulesPath: modulesPath, NpmPath: npmPath}, "auth")
		if err != nil {
			t.Fatalf("TypecheckApp returned error: %v", err)
		}
	})

	t.Run("accepts relative modules path", func(t *testing.T) {
		repoRoot := t.TempDir()
		modulesPath := filepath.Join(repoRoot, "modules")
		makeDir(t, filepath.Join(modulesPath, "auth", "service"))
		writeFile(t, filepath.Join(modulesPath, "auth", "service", "index.ts"), "export const auth = 1\n")
		npmPath, _, _ := makeFakeTypecheckTooling(t, repoRoot, "exit 0\n")

		originalWD, err := os.Getwd()
		if err != nil {
			t.Fatalf("Getwd returned error: %v", err)
		}
		if err := os.Chdir(repoRoot); err != nil {
			t.Fatalf("Chdir(%q): %v", repoRoot, err)
		}
		defer func() {
			_ = os.Chdir(originalWD)
		}()

		err = TypecheckApp(context.Background(), RunOptions{
			ModulesPath: "modules",
			NpmPath:     npmPath,
			RepoRoot:    repoRoot,
			Stderr:      &strings.Builder{},
		}, "auth")
		if err != nil {
			t.Fatalf("TypecheckApp(relative modules path) returned error: %v", err)
		}
	})

	t.Run("writes temp tsconfig and cleans it up on success", func(t *testing.T) {
		repoRoot := t.TempDir()
		modulesPath := t.TempDir()
		makeDir(t, filepath.Join(modulesPath, "auth", "service"))
		writeFile(t, filepath.Join(modulesPath, "auth", "service", "index.ts"), "export const auth = 1\n")
		tmpPath := t.TempDir()
		npmPath, copyPath, _ := makeFakeTypecheckTooling(t, repoRoot, "exit 0\n")
		tsconfigPathCapture := filepath.Join(t.TempDir(), "tsconfig-path.txt")
		t.Setenv("CHOYSUM_CAPTURE_TSCONFIG_PATH", tsconfigPathCapture)
		t.Setenv("CHOYSUM_COPY_TSCONFIG", copyPath)

		var stderr strings.Builder
		err := TypecheckApp(context.Background(), RunOptions{
			ModulesPath: modulesPath,
			NpmPath:     npmPath,
			RepoRoot:    repoRoot,
			TmpPath:     tmpPath,
			Stderr:      &stderr,
		}, "auth")
		if err != nil {
			t.Fatalf("TypecheckApp returned error: %v", err)
		}

		tsconfigPathRaw, err := os.ReadFile(tsconfigPathCapture)
		if err != nil {
			t.Fatalf("ReadFile(%q): %v", tsconfigPathCapture, err)
		}
		usedTsconfigPath := strings.TrimSpace(string(tsconfigPathRaw))
		if usedTsconfigPath == "" {
			t.Fatalf("expected captured tsconfig path to be non-empty")
		}
		wantDir, err := testingpathing.ResolveTestingTmpDir(repoRoot, tmpPath, "typecheck")
		if err != nil {
			t.Fatalf("ResolveTestingTmpDir(typecheck): %v", err)
		}
		wantDir = filepath.Join(wantDir, sanitizeAppToken("auth"))
		if gotDir := filepath.Dir(usedTsconfigPath); gotDir != wantDir {
			t.Fatalf("tsconfig dir = %q, want %q", gotDir, wantDir)
		}

		captured, err := os.ReadFile(copyPath)
		if err != nil {
			t.Fatalf("ReadFile(%q): %v", copyPath, err)
		}
		capturedText := string(captured)
		for _, fragment := range []string{
			filepath.ToSlash(filepath.Join(modulesPath, "auth", "**", "*.d.ts")),
			filepath.ToSlash(filepath.Join(modulesPath, "auth", "*.ts")),
			filepath.ToSlash(filepath.Join(modulesPath, "auth", "service", "**", "*.ts")),
			filepath.ToSlash(filepath.Join(modulesPath, "auth", "web", "**", "*.tsx")),
			filepath.ToSlash(filepath.Join(modulesPath, "auth", "web", "**", "*.vue")),
			filepath.ToSlash(filepath.Join(modulesPath, "*")),
			filepath.ToSlash(filepath.Join(repoRoot, "node_modules", "@types")),
			"\"types\": [\n      \"node\"\n    ]",
			"\"noEmit\": true",
		} {
			if !strings.Contains(capturedText, fragment) {
				t.Fatalf("expected captured tsconfig to contain %q, got %q", fragment, capturedText)
			}
		}
		if strings.Contains(capturedText, `"baseUrl"`) {
			t.Fatalf("expected captured tsconfig not to contain baseUrl, got %q", capturedText)
		}

		if !strings.Contains(stderr.String(), "# typecheck auth\n# typecheck auth ok\n") {
			t.Fatalf("unexpected stderr output: %q", stderr.String())
		}

		if _, err := os.Stat(usedTsconfigPath); !os.IsNotExist(err) {
			t.Fatalf("expected temporary tsconfig to be removed, stat err=%v", err)
		}
		if _, err := os.Stat(filepath.Dir(usedTsconfigPath)); !os.IsNotExist(err) {
			t.Fatalf("expected temporary tsconfig dir to be removed, stat err=%v", err)
		}
	})

	t.Run("includes repo vite client types for web apps", func(t *testing.T) {
		repoRoot := t.TempDir()
		modulesPath := t.TempDir()
		tmpPath := t.TempDir()
		npmPath, copyPath, _ := makeFakeTypecheckTooling(t, repoRoot, "exit 0\n")
		makeDir(t, filepath.Join(modulesPath, "auth", "web"))
		writeFile(t, filepath.Join(modulesPath, "auth", "web", "index.ts"), "export const auth = 1\n")
		makeDir(t, filepath.Join(repoRoot, "node_modules", "vite"))
		writeFile(t, filepath.Join(repoRoot, "node_modules", "vite", "client.d.ts"), "declare interface ImportMetaEnv {}\n")
		tsconfigPathCapture := filepath.Join(t.TempDir(), "tsconfig-path.txt")
		t.Setenv("CHOYSUM_CAPTURE_TSCONFIG_PATH", tsconfigPathCapture)
		t.Setenv("CHOYSUM_COPY_TSCONFIG", copyPath)

		err := TypecheckApp(context.Background(), RunOptions{
			ModulesPath: modulesPath,
			NpmPath:     npmPath,
			RepoRoot:    repoRoot,
			TmpPath:     tmpPath,
			Stderr:      &strings.Builder{},
		}, "auth")
		if err != nil {
			t.Fatalf("TypecheckApp returned error: %v", err)
		}

		captured, err := os.ReadFile(copyPath)
		if err != nil {
			t.Fatalf("ReadFile(%q): %v", copyPath, err)
		}
		capturedText := string(captured)
		viteClientPath := filepath.ToSlash(filepath.Join(repoRoot, "node_modules", "vite", "client.d.ts"))
		if !strings.Contains(capturedText, viteClientPath) {
			t.Fatalf("expected captured tsconfig to contain %q, got %q", viteClientPath, capturedText)
		}
		if !strings.Contains(capturedText, `"types":`) {
			t.Fatalf("expected captured tsconfig to include compilerOptions.types, got %q", capturedText)
		}
	})

	t.Run("includes modules node_modules vite client types when repo copy is absent", func(t *testing.T) {
		repoRoot := t.TempDir()
		modulesPath := t.TempDir()
		tmpPath := t.TempDir()
		npmPath, copyPath, _ := makeFakeTypecheckTooling(t, repoRoot, "exit 0\n")
		if err := os.RemoveAll(filepath.Join(repoRoot, "node_modules", "vite")); err != nil {
			t.Fatalf("RemoveAll repo vite: %v", err)
		}

		makeDir(t, filepath.Join(modulesPath, "auth", "web"))
		writeFile(t, filepath.Join(modulesPath, "auth", "web", "index.ts"), "export const auth = 1\n")
		makeDir(t, filepath.Join(modulesPath, "node_modules", "vite"))
		writeFile(t, filepath.Join(modulesPath, "node_modules", "vite", "package.json"), "{}\n")
		writeFile(t, filepath.Join(modulesPath, "node_modules", "vite", "client.d.ts"), "declare interface ImportMetaEnv {}\n")

		tsconfigPathCapture := filepath.Join(t.TempDir(), "tsconfig-path.txt")
		t.Setenv("CHOYSUM_CAPTURE_TSCONFIG_PATH", tsconfigPathCapture)
		t.Setenv("CHOYSUM_COPY_TSCONFIG", copyPath)

		err := TypecheckApp(context.Background(), RunOptions{
			ModulesPath: modulesPath,
			NpmPath:     npmPath,
			RepoRoot:    repoRoot,
			TmpPath:     tmpPath,
			Stderr:      &strings.Builder{},
		}, "auth")
		if err != nil {
			t.Fatalf("TypecheckApp returned error: %v", err)
		}

		captured, err := os.ReadFile(copyPath)
		if err != nil {
			t.Fatalf("ReadFile(%q): %v", copyPath, err)
		}
		capturedText := string(captured)
		moduleViteClientPath := filepath.ToSlash(filepath.Join(modulesPath, "node_modules", "vite", "client.d.ts"))
		if !strings.Contains(capturedText, moduleViteClientPath) {
			t.Fatalf("expected captured tsconfig to contain %q, got %q", moduleViteClientPath, capturedText)
		}
	})

	t.Run("keeps temp tsconfig when keep is enabled", func(t *testing.T) {
		repoRoot := t.TempDir()
		modulesPath := t.TempDir()
		makeDir(t, filepath.Join(modulesPath, "auth", "service"))
		writeFile(t, filepath.Join(modulesPath, "auth", "service", "index.ts"), "export const auth = 1\n")
		tmpPath := t.TempDir()
		npmPath, _, _ := makeFakeTypecheckTooling(t, repoRoot, "exit 0\n")
		tsconfigPathCapture := filepath.Join(t.TempDir(), "tsconfig-path.txt")
		t.Setenv("CHOYSUM_CAPTURE_TSCONFIG_PATH", tsconfigPathCapture)

		err := TypecheckApp(context.Background(), RunOptions{
			ModulesPath: modulesPath,
			NpmPath:     npmPath,
			RepoRoot:    repoRoot,
			TmpPath:     tmpPath,
			Keep:        true,
			Stderr:      &strings.Builder{},
		}, "auth")
		if err != nil {
			t.Fatalf("TypecheckApp returned error: %v", err)
		}

		tsconfigPathRaw, err := os.ReadFile(tsconfigPathCapture)
		if err != nil {
			t.Fatalf("ReadFile(%q): %v", tsconfigPathCapture, err)
		}
		usedTsconfigPath := strings.TrimSpace(string(tsconfigPathRaw))
		if usedTsconfigPath == "" {
			t.Fatalf("expected captured tsconfig path to be non-empty")
		}
		if _, err := os.Stat(usedTsconfigPath); err != nil {
			t.Fatalf("expected temporary tsconfig to be kept, stat err=%v", err)
		}
		if _, err := os.Stat(filepath.Dir(usedTsconfigPath)); err != nil {
			t.Fatalf("expected temporary tsconfig dir to be kept, stat err=%v", err)
		}
	})

	t.Run("forwards command output and wraps command failure", func(t *testing.T) {
		repoRoot := t.TempDir()
		modulesPath := t.TempDir()
		makeDir(t, filepath.Join(modulesPath, "auth", "service"))
		writeFile(t, filepath.Join(modulesPath, "auth", "service", "index.ts"), "export const auth = 1\n")
		tmpPath := t.TempDir()
		npmPath, _, _ := makeFakeTypecheckTooling(t, repoRoot, "printf 'compile failed'; exit 7\n")
		tsconfigPathCapture := filepath.Join(t.TempDir(), "tsconfig-path.txt")
		t.Setenv("CHOYSUM_CAPTURE_TSCONFIG_PATH", tsconfigPathCapture)

		var stderr strings.Builder
		err := TypecheckApp(context.Background(), RunOptions{
			ModulesPath: modulesPath,
			NpmPath:     npmPath,
			RepoRoot:    repoRoot,
			TmpPath:     tmpPath,
			Stderr:      &stderr,
		}, "auth")
		if err == nil || !strings.Contains(err.Error(), "typecheck failed for auth") {
			t.Fatalf("expected wrapped command error, got %v", err)
		}
		if !strings.Contains(stderr.String(), "# typecheck auth\ncompile failed\n") {
			t.Fatalf("expected forwarded command output with newline, got %q", stderr.String())
		}

		tsconfigPathRaw, readErr := os.ReadFile(tsconfigPathCapture)
		if readErr != nil {
			t.Fatalf("ReadFile(%q): %v", tsconfigPathCapture, readErr)
		}
		usedTsconfigPath := strings.TrimSpace(string(tsconfigPathRaw))
		if usedTsconfigPath == "" {
			t.Fatalf("expected captured tsconfig path to be non-empty")
		}
		wantDir, err := testingpathing.ResolveTestingTmpDir(repoRoot, tmpPath, "typecheck")
		if err != nil {
			t.Fatalf("ResolveTestingTmpDir(typecheck): %v", err)
		}
		wantDir = filepath.Join(wantDir, sanitizeAppToken("auth"))
		if gotDir := filepath.Dir(usedTsconfigPath); gotDir != wantDir {
			t.Fatalf("tsconfig dir = %q, want %q", gotDir, wantDir)
		}
		if _, err := os.Stat(usedTsconfigPath); !os.IsNotExist(err) {
			t.Fatalf("expected temporary tsconfig to be removed after failure, stat err=%v", err)
		}
		if _, err := os.Stat(filepath.Dir(usedTsconfigPath)); !os.IsNotExist(err) {
			t.Fatalf("expected temporary tsconfig dir to be removed after failure, stat err=%v", err)
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

		npmPath, _, _ := makeFakeTypecheckTooling(t, repoRoot, "exit 0\n")
		var stderr strings.Builder
		err := TypecheckApp(context.Background(), RunOptions{
			ModulesPath: modulesPath,
			NpmPath:     npmPath,
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

		npmPath, _, _ := makeFakeTypecheckTooling(t, repoRoot, "exit 0\n")
		var stderr strings.Builder
		err := TypecheckApp(context.Background(), RunOptions{
			ModulesPath: modulesPath,
			NpmPath:     npmPath,
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

		npmPath, _, _ := makeFakeTypecheckTooling(t, repoRoot, "exit 0\n")
		var stderr strings.Builder
		err := TypecheckApp(context.Background(), RunOptions{
			ModulesPath: modulesPath,
			NpmPath:     npmPath,
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

		npmPath, _, _ := makeFakeTypecheckTooling(t, repoRoot, "exit 0\n")
		err := TypecheckApp(context.Background(), RunOptions{
			ModulesPath: modulesPath,
			NpmPath:     npmPath,
			RepoRoot:    repoRoot,
			TmpPath:     t.TempDir(),
			Stderr:      &strings.Builder{},
		}, "auth")
		if err == nil || !strings.Contains(err.Error(), "typecheck: collect module dependencies:") {
			t.Fatalf("expected dependency collection error, got %v", err)
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

		npmPath, _, _ := makeFakeTypecheckTooling(t, repoRoot, "exit 0\n")
		var stderr strings.Builder
		err := TypecheckApp(context.Background(), RunOptions{
			ModulesPath: modulesPath,
			NpmPath:     npmPath,
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

	t.Run("appends type-fetch guidance when diagnostics indicate missing type declarations", func(t *testing.T) {
		repoRoot := t.TempDir()
		modulesPath := t.TempDir()
		makeDir(t, filepath.Join(modulesPath, "auth", "service"))
		writeFile(t, filepath.Join(modulesPath, "auth", "service", "index.ts"), "export const auth = 1\n")

		npmPath, _, _ := makeFakeTypecheckTooling(t, repoRoot, "printf \"error TS2307: Cannot find module 'missing-lib' or its corresponding type declarations.\"; exit 7\n")
		var stderr strings.Builder
		err := TypecheckApp(context.Background(), RunOptions{
			ModulesPath: modulesPath,
			NpmPath:     npmPath,
			RepoRoot:    repoRoot,
			TmpPath:     t.TempDir(),
			Stderr:      &stderr,
		}, "auth")
		if err == nil {
			t.Fatal("expected typecheck command error")
		}
		if !strings.Contains(err.Error(), "typecheck failed for auth") {
			t.Fatalf("expected wrapped typecheck failure, got %v", err)
		}
		if !strings.Contains(err.Error(), "go run . type-fetch auth") {
			t.Fatalf("expected type-fetch guidance in error, got %v", err)
		}
	})

	t.Run("returns clear error when web app vite client types are missing", func(t *testing.T) {
		repoRoot := t.TempDir()
		modulesPath := t.TempDir()
		// Set up fake tooling without vite to trigger the missing-vite error.
		binDir := filepath.Join(t.TempDir(), "bin")
		makeDir(t, binDir)
		npmPath := filepath.Join(binDir, "npm")
		npxPath := filepath.Join(binDir, "npx")
		vueTscPath := filepath.Join(binDir, "vue-tsc")
		writeFile(t, npmPath, "#!/bin/sh\nexit 0\n")
		writeFile(t, npxPath, "#!/bin/sh\nexit 0\n")
		writeFile(t, vueTscPath, "#!/bin/sh\nexit 0\n")
		t.Setenv("PATH", binDir)

		makeDir(t, filepath.Join(repoRoot, "node_modules", "vue-tsc"))
		writeFile(t, filepath.Join(repoRoot, "node_modules", "vue-tsc", "package.json"), "{}\n")

		makeDir(t, filepath.Join(modulesPath, "auth"))
		writeFile(t, filepath.Join(modulesPath, "auth", "package.json"), `{"dependencies":{"lodash":"^4.17.21"}}`)
		makeDir(t, filepath.Join(modulesPath, "auth", "web"))
		writeFile(t, filepath.Join(modulesPath, "auth", "web", "index.ts"), "export const auth = 1\n")

		var stderr strings.Builder

		err := TypecheckApp(context.Background(), RunOptions{
			ModulesPath: modulesPath,
			NpmPath:     npmPath,
			RepoRoot:    repoRoot,
			TmpPath:     t.TempDir(),
			Stderr:      &stderr,
		}, "auth")
		if err == nil || !strings.Contains(err.Error(), "missing 1 required module(s): vite") {
			t.Fatalf("expected vite missing error, got %v", err)
		}
		if !strings.Contains(err.Error(), "npm install -g vite") {
			t.Fatalf("expected npm global install hint, got %v", err)
		}
		if !strings.Contains(err.Error(), "install command:") {
			t.Fatalf("expected structured install command section, got %v", err)
		}
		if !strings.Contains(err.Error(), "retry:\n  go run . test typecheck auth") {
			t.Fatalf("expected retry hint in error, got %v", err)
		}
	})

	t.Run("allows local vite binary without global PATH entry", func(t *testing.T) {
		repoRoot := t.TempDir()
		modulesPath := t.TempDir()

		binDir := filepath.Join(t.TempDir(), "bin")
		makeDir(t, binDir)
		npmPath := filepath.Join(binDir, "npm")
		npxPath := filepath.Join(binDir, "npx")
		writeFile(t, npmPath, "#!/bin/sh\nexit 0\n")
		writeFile(t, npxPath, "#!/bin/sh\nexit 0\n")
		t.Setenv("PATH", binDir)

		makeDir(t, filepath.Join(repoRoot, "node_modules", "vite"))
		writeFile(t, filepath.Join(repoRoot, "node_modules", "vite", "client.d.ts"), "declare module 'vite/client' {}\n")
		makeDir(t, filepath.Join(repoRoot, "node_modules", "vue-tsc"))
		writeFile(t, filepath.Join(repoRoot, "node_modules", "vue-tsc", "package.json"), "{}\n")
		makeDir(t, filepath.Join(repoRoot, "node_modules", ".bin"))
		writeFile(t, filepath.Join(repoRoot, "node_modules", ".bin", "vite"), "#!/bin/sh\nexit 0\n")

		makeDir(t, filepath.Join(modulesPath, "auth", "web"))
		writeFile(t, filepath.Join(modulesPath, "auth", "web", "index.ts"), "export const auth = 1\n")

		err := TypecheckApp(context.Background(), RunOptions{
			ModulesPath: modulesPath,
			NpmPath:     npmPath,
			RepoRoot:    repoRoot,
			TmpPath:     t.TempDir(),
			Stderr:      &strings.Builder{},
		}, "auth")
		if err != nil {
			t.Fatalf("expected local vite binary to be accepted, got %v", err)
		}
	})

	t.Run("best-effort cleanup ignores non-empty tmp directory", func(t *testing.T) {
		repoRoot := t.TempDir()
		modulesPath := t.TempDir()
		makeDir(t, filepath.Join(modulesPath, "auth", "service"))
		writeFile(t, filepath.Join(modulesPath, "auth", "service", "index.ts"), "export const auth = 1\n")
		tmpPath := t.TempDir()
		npmPath, _, _ := makeFakeTypecheckTooling(t, repoRoot, "dir=$(dirname \"$4\")\nprintf 'keep' > \"$dir/sentinel.keep\"\nexit 0\n")

		err := TypecheckApp(context.Background(), RunOptions{
			ModulesPath: modulesPath,
			NpmPath:     npmPath,
			RepoRoot:    repoRoot,
			TmpPath:     tmpPath,
			Stderr:      &strings.Builder{},
		}, "auth")
		if err != nil {
			t.Fatalf("TypecheckApp returned error: %v", err)
		}

		workspaceDir, err := testingpathing.ResolveTestingTmpDir(repoRoot, tmpPath, "typecheck")
		if err != nil {
			t.Fatalf("ResolveTestingTmpDir(typecheck): %v", err)
		}
		sentinelPath := filepath.Join(workspaceDir, sanitizeAppToken("auth"), "sentinel.keep")
		if _, err := os.Stat(sentinelPath); err != nil {
			t.Fatalf("expected sentinel file to remain in workspace tmp dir, stat err=%v", err)
		}
	})
}

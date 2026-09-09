// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package e2e

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/choysum-dev/choysum/internal/esmresolver"
	"github.com/choysum-dev/choysum/pkg/jsengine/scripts/choysume2e"
	"github.com/evanw/esbuild/pkg/api"
	xfmt "golang.org/x/exp/errors/fmt"
)

var (
	e2eOsMkdirAll     = os.MkdirAll
	e2eOsWriteFile    = os.WriteFile
	e2eOsReadFile     = os.ReadFile
	e2eOsUserHome     = os.UserHomeDir
	e2eFilepathAbs    = filepath.Abs
	e2eEsbuildBuild   = api.Build
	e2eJSONMarshal    = json.Marshal
	e2eChoysumE2EPath = choysume2e.SourcePath
)

// E2EBundleOptions configures the QuickJS e2e host bundle.
type E2EBundleOptions struct {
	RepoRoot   string
	EntryPath  string
	Outfile    string
	WorkingDir string
	CacheDir   string
	// RunDir enables alias of generated pb under <runDir>/.choysum/generated.
	RunDir string
}

// E2EBundleResult is esbuild output for an e2e host bundle.
type E2EBundleResult struct {
	JS     string
	JSPath string
}

// WriteE2EEntry writes a generated entry that imports each absolute spec path.
func WriteE2EEntry(entryPath string, specFiles []string) error {
	if strings.TrimSpace(entryPath) == "" {
		return xfmt.Errorf("e2e bundle: empty entry path")
	}
	if err := e2eOsMkdirAll(filepath.Dir(entryPath), 0o755); err != nil {
		return xfmt.Errorf("e2e bundle: mkdir entry: %w", err)
	}
	var b strings.Builder
	for _, f := range specFiles {
		absFile, err := e2eFilepathAbs(filepath.Clean(f))
		if err != nil {
			return xfmt.Errorf("e2e bundle: resolve spec path: %w", err)
		}
		encoded, err := e2eJSONMarshal(filepath.ToSlash(absFile))
		if err != nil {
			return xfmt.Errorf("e2e bundle: encode import path: %w", err)
		}
		b.WriteString("import ")
		b.Write(encoded)
		b.WriteString(";\n")
	}
	if err := e2eOsWriteFile(entryPath, []byte(b.String()), 0o644); err != nil {
		return xfmt.Errorf("e2e bundle: write entry: %w", err)
	}
	return nil
}

// BuildE2EBundle bundles e2e specs with `@choysum/e2e` alias and esmresolver.
func BuildE2EBundle(opts E2EBundleOptions) (*E2EBundleResult, error) {
	repoRoot := strings.TrimSpace(opts.RepoRoot)
	entry := strings.TrimSpace(opts.EntryPath)
	if repoRoot == "" {
		return nil, xfmt.Errorf("e2e bundle: empty repo root")
	}
	if entry == "" {
		return nil, xfmt.Errorf("e2e bundle: empty entry path")
	}
	e2ePath, err := e2eChoysumE2EPath()
	if err != nil {
		return nil, xfmt.Errorf("e2e bundle: choysume2e path: %w", err)
	}
	e2ePath, err = e2eFilepathAbs(e2ePath)
	if err != nil {
		return nil, xfmt.Errorf("e2e bundle: abs choysume2e: %w", err)
	}

	cacheDir := strings.TrimSpace(opts.CacheDir)
	if cacheDir == "" {
		cacheDir = os.Getenv("CHOYSUM_HOME")
	}
	if cacheDir == "" {
		home, homeErr := e2eOsUserHome()
		if homeErr != nil {
			return nil, xfmt.Errorf("e2e bundle: cache dir: %w", homeErr)
		}
		cacheDir = filepath.Join(home, ".choysum")
	}

	outfile := strings.TrimSpace(opts.Outfile)
	if outfile == "" {
		outfile = filepath.Join(filepath.Dir(entry), "e2e.bundle.js")
	}
	if err := e2eOsMkdirAll(filepath.Dir(outfile), 0o755); err != nil {
		return nil, xfmt.Errorf("e2e bundle: mkdir: %w", err)
	}

	modulesDir := filepath.Join(repoRoot, "modules")
	alias := map[string]string{
		"@": modulesDir,
	}

	plugins := []api.Plugin{
		{
			Name: "choysum-e2e-alias",
			Setup: func(build api.PluginBuild) {
				build.OnResolve(api.OnResolveOptions{Filter: `^@choysum/e2e$`},
					func(args api.OnResolveArgs) (api.OnResolveResult, error) {
						return api.OnResolveResult{Path: e2ePath, Namespace: "file"}, nil
					})
			},
		},
	}

	runDir := strings.TrimSpace(opts.RunDir)
	if runDir != "" {
		generatedRoot := filepath.Join(runDir, ".choysum", "generated")
		plugins = append(plugins, api.Plugin{
			Name: "choysum-e2e-generated-pb",
			Setup: func(build api.PluginBuild) {
				build.OnResolve(api.OnResolveOptions{Filter: `.*_pb(\.ts)?$`},
					func(args api.OnResolveArgs) (api.OnResolveResult, error) {
						base := filepath.Base(args.Path)
						cand := filepath.Join(generatedRoot, "web", base)
						if _, err := os.Stat(cand); err == nil {
							return api.OnResolveResult{Path: cand, Namespace: "file"}, nil
						}
						found := ""
						_ = filepath.WalkDir(generatedRoot, func(path string, d os.DirEntry, err error) error {
							if err != nil || d.IsDir() {
								return err
							}
							if d.Name() == base {
								found = path
								return filepath.SkipAll
							}
							return nil
						})
						if found != "" {
							return api.OnResolveResult{Path: found, Namespace: "file"}, nil
						}
						return api.OnResolveResult{}, nil
					})
			},
		})
	}

	plugins = append(plugins, esmresolver.New(
		esmresolver.WithCacheDir(cacheDir),
		esmresolver.WithTarget("es2020"),
		esmresolver.WithModulePath(repoRoot),
	).Plugin())

	absWorkingDir := strings.TrimSpace(opts.WorkingDir)
	if absWorkingDir == "" {
		absWorkingDir = repoRoot
	}

	result := e2eEsbuildBuild(api.BuildOptions{
		EntryPoints:   []string{entry},
		Bundle:        true,
		Write:         true,
		Outfile:       outfile,
		Platform:      api.PlatformNeutral,
		Format:        api.FormatIIFE,
		Target:        api.ES2020,
		LogLevel:      api.LogLevelWarning,
		AbsWorkingDir: absWorkingDir,
		Alias:         alias,
		Plugins:       plugins,
		Loader: map[string]api.Loader{
			".ts":  api.LoaderTS,
			".tsx": api.LoaderTSX,
			".css": api.LoaderEmpty,
		},
		Define: map[string]string{
			"process.env.NODE_ENV": "'test'",
		},
	})
	if len(result.Errors) > 0 {
		var b strings.Builder
		for _, e := range result.Errors {
			b.WriteString(e.Text)
			b.WriteByte('\n')
		}
		return nil, xfmt.Errorf("e2e bundle: esbuild: %s", b.String())
	}
	jsBytes, err := e2eOsReadFile(outfile)
	if err != nil {
		return nil, xfmt.Errorf("e2e bundle: read outfile: %w", err)
	}
	return &E2EBundleResult{JS: string(jsBytes), JSPath: outfile}, nil
}

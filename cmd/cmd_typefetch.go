// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/choysum-dev/choysum/internal/esmresolver"
	logutil "github.com/choysum-dev/choysum/internal/logger"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/spf13/cobra"
	xfmt "golang.org/x/exp/errors/fmt"
)

type offlineTransport struct{}

func (*offlineTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, xfmt.Errorf("offline mode enabled: network request blocked")
}

func newTypeFetchCmd(envGetter func() scope.Scope) *cobra.Command {
	var all bool
	var withDepends bool
	var missingDepPolicy string
	var upstream string
	var offline bool

	cmd := &cobra.Command{
		Use:   "type-fetch [<app>]",
		Short: "Fetch type definitions (.d.ts) for module dependencies",
		Annotations: map[string]string{
			lightweightScopeAnnotation: "true",
		},
		Long: `Scans the module's package.json for dependencies, downloads their
type definitions (.d.ts) from the configured ESM upstream, and caches them
locally for IDE support.

When called without arguments, fetches types for all installed modules.
When <app> is specified, fetches types for that module only.`,
		Hidden: true,
		Args: func(cmd *cobra.Command, args []string) error {
			if all && len(args) > 0 {
				return xfmt.Errorf("type-fetch: --all cannot be used with an app argument")
			}
			if !all && len(args) > 1 {
				return xfmt.Errorf("type-fetch: accepts at most 1 app argument (or use --all)")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			baseScope := envGetter()
			if baseScope == nil {
				return xfmt.Errorf("type-fetch: invalid scope")
			}

			runtimeOpts, hasRuntimeOpts := scope.PathsRuntimeOptionsFromScope(baseScope)
			if !hasRuntimeOpts {
				return xfmt.Errorf("type-fetch: missing runtime options")
			}

			modulesPath := strings.TrimSpace(runtimeOpts.ModulesPath)
			if modulesPath == "" {
				return xfmt.Errorf("type-fetch: modules path is empty")
			}
			if err := os.MkdirAll(modulesPath, 0o755); err != nil {
				return xfmt.Errorf("type-fetch: ensure modules path: %w", err)
			}

			tsconfigPath := filepath.Join(modulesPath, "tsconfig.json")
			if err := esmresolver.UpdateTsconfigPaths(tsconfigPath, nil); err != nil {
				return xfmt.Errorf("type-fetch: ensure modules tsconfig: %w", err)
			}
			cmd.Printf("Ensured tsconfig exists: %s\n", tsconfigPath)

			defaultPath := strings.TrimSpace(runtimeOpts.DefaultChoysumPath)
			if defaultPath == "" {
				defaultPath = ".choysum"
			}
			typesDir := filepath.Join(defaultPath, "pkg", "types")

			if upstream == "" {
				upstream = strings.TrimSpace(runtimeOpts.ESMUpstreamURL)
				if upstream == "" {
					upstream = config.DefaultESMUpstreamURL
				}
			}

			var client *http.Client
			if offline {
				client = &http.Client{Transport: &offlineTransport{}}
			} else {
				client = esmresolver.NewTypeFetchHTTPClient(30 * time.Second)
				if transport, ok := client.Transport.(*http.Transport); ok {
					defer transport.CloseIdleConnections()
				}
			}

			var appNames []string
			if all || len(args) == 0 {
				// Scan all modules.
				entries, err := os.ReadDir(modulesPath)
				if err != nil {
					return xfmt.Errorf("type-fetch: read modules dir: %w", err)
				}
				for _, entry := range entries {
					if entry.IsDir() {
						pkgPath := filepath.Join(modulesPath, entry.Name(), "package.json")
						if _, err := os.Stat(pkgPath); err == nil {
							appNames = append(appNames, entry.Name())
						}
					}
				}
			} else {
				appNames = []string{args[0]}
			}

			singleModuleMode := !all && len(args) == 1
			policy, err := resolveTypeFetchMissingDepPolicy(missingDepPolicy, singleModuleMode)
			if err != nil {
				return err
			}
			if withDepends && singleModuleMode {
				expanded, missingDepends, err := resolveTypeFetchDependsClosure(modulesPath, args[0])
				if err != nil {
					return xfmt.Errorf("type-fetch: resolve module depends closure: %w", err)
				}
				appNames = expanded
				if len(missingDepends) > 0 {
					if policy == typeFetchMissingDepPolicyError {
						return xfmt.Errorf("type-fetch: missing depends modules: %s", strings.Join(missingDepends, ", "))
					}
					cmd.Printf("Warning: missing depends modules (skipped): %s\n", strings.Join(missingDepends, ", "))
				}
			} else if withDepends {
				missingDepends, err := validateTypeFetchDependsCompleteness(modulesPath, appNames)
				if err != nil {
					return xfmt.Errorf("type-fetch: validate module depends completeness: %w", err)
				}
				if len(missingDepends) > 0 {
					if policy == typeFetchMissingDepPolicyError {
						return xfmt.Errorf("type-fetch: missing depends modules: %s", strings.Join(missingDepends, ", "))
					}
					cmd.Printf("Warning: missing depends modules (skipped): %s\n", strings.Join(missingDepends, ", "))
				}
			}

			if len(appNames) == 0 {
				cmd.Println("No modules found with package.json.")
				return nil
			}

			compilerTypeTargets, err := resolveTypeFetchCompilerTypeTargets(tsconfigPath, modulesPath)
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			var spinnerTicker *logutil.ProgressTicker
			if progressLine := logutil.NewProgressLine(os.Stderr); progressLine != nil && progressLine.IsTTY() {
				spinnerTicker = logutil.NewProgressTicker(progressLine, logutil.ProgressTickerOptions{Interval: 120 * time.Millisecond})
				defer spinnerTicker.Clear()
				defer spinnerTicker.Stop()
				ctx = logutil.WithProgressTicker(ctx, spinnerTicker)
			}
			setCommandProgress := func(message string) {
				if spinnerTicker == nil {
					return
				}
				spinnerTicker.SetMessage(message)
			}
			clearCommandProgress := func() {
				if spinnerTicker == nil {
					return
				}
				spinnerTicker.Clear()
			}

			session := esmresolver.NewTypeFetchSession(0)

			totalDirectTargets := 0
			totalDirectCached := 0
			totalDirectFetched := 0
			totalDirectReused := 0
			totalDirectFailed := 0
			totalTransitiveCached := 0
			totalTransitiveFetched := 0
			totalCompilerTypeTargets := 0
			totalCompilerTypeCached := 0
			totalCompilerTypeFetched := 0
			totalCompilerTypeFailed := 0
			totalCompilerTypeTransitiveCached := 0
			totalCompilerTypeTransitiveFetched := 0
			var allResults []esmresolver.TypeFetchResult
			compilerTypeRootLinks := make([]esmresolver.CompilerTypeRootLink, 0, len(compilerTypeTargets))

			for i, target := range compilerTypeTargets {
				if err := ctx.Err(); err != nil {
					return err
				}
				totalCompilerTypeTargets++
				setCommandProgress(fmt.Sprintf("[tsconfig] fetching compiler type (%d/%d): %s -> %s@%s", i+1, len(compilerTypeTargets), target.TypeName, target.PackageName, target.Version))
				result, transitive, err := esmresolver.FetchTypeDefinition(client, upstream, typesDir, target.PackageName, target.Version)
				clearCommandProgress()
				if err != nil {
					totalCompilerTypeFailed++
					cmd.Printf("[tsconfig] warning: failed to fetch compiler type %q via %s@%s: %v\n", target.TypeName, target.PackageName, target.Version, err)
					continue
				}
				if result != nil {
					if result.FromCache {
						totalCompilerTypeCached++
					} else {
						totalCompilerTypeFetched++
					}
					compilerTypeRootLinks = append(compilerTypeRootLinks, esmresolver.CompilerTypeRootLink{TypeName: target.TypeName, CachedPath: result.CachedPath})
					allResults = append(allResults, *result)
				}
				for _, item := range transitive {
					if item.FromCache {
						totalCompilerTypeTransitiveCached++
					} else {
						totalCompilerTypeTransitiveFetched++
					}
				}
				allResults = append(allResults, transitive...)
			}

			for i, appName := range appNames {
				if err := ctx.Err(); err != nil {
					return err
				}
				moduleDir := filepath.Join(modulesPath, appName)
				setCommandProgress(fmt.Sprintf("[%s] fetching dependency types (%d/%d)", appName, i+1, len(appNames)))
				results, stats, err := session.FetchTypesForModuleWithStats(ctx, client, upstream, typesDir, moduleDir)
				if err != nil {
					clearCommandProgress()
					if ctxErr := ctx.Err(); ctxErr != nil {
						return ctxErr
					}
					cmd.Printf("[%s] error: %v\n", appName, err)
					// When the user explicitly targets a single app (not --all),
					// any failure should be fatal so the caller gets a non-zero
					// exit code and can react accordingly.
					if len(appNames) == 1 {
						return err
					}
					continue
				}
				totalDirectTargets += stats.DirectTargets
				totalDirectCached += stats.DirectCached
				totalDirectFetched += stats.DirectFetched
				totalDirectReused += stats.DirectReused
				totalDirectFailed += stats.DirectFailed
				totalTransitiveCached += stats.TransitiveCached
				totalTransitiveFetched += stats.TransitiveFetched
				allResults = append(allResults, results...)
				clearCommandProgress()
				cmd.Printf("[%s] completed: direct targets=%d (cached=%d, fetched=%d, reused=%d, failed=%d), transitive (cached=%d, fetched=%d)\n",
					appName,
					stats.DirectTargets,
					stats.DirectCached,
					stats.DirectFetched,
					stats.DirectReused,
					stats.DirectFailed,
					stats.TransitiveCached,
					stats.TransitiveFetched,
				)
			}
			clearCommandProgress()

			cmd.Printf("\nType fetch complete: direct targets=%d (cached=%d, fetched=%d, reused=%d, failed=%d), transitive (cached=%d, fetched=%d).\n",
				totalDirectTargets,
				totalDirectCached,
				totalDirectFetched,
				totalDirectReused,
				totalDirectFailed,
				totalTransitiveCached,
				totalTransitiveFetched,
			)
			if totalCompilerTypeTargets > 0 {
				cmd.Printf("Compiler types complete: targets=%d (cached=%d, fetched=%d, failed=%d), transitive (cached=%d, fetched=%d).\n",
					totalCompilerTypeTargets,
					totalCompilerTypeCached,
					totalCompilerTypeFetched,
					totalCompilerTypeFailed,
					totalCompilerTypeTransitiveCached,
					totalCompilerTypeTransitiveFetched,
				)
			}
			cmd.Printf("Types directory: %s\n", typesDir)

			// Update tsconfig paths for IDE support.
			if err := esmresolver.UpdateTsconfigPaths(tsconfigPath, allResults); err != nil {
				cmd.Printf("Warning: failed to update tsconfig paths: %v\n", err)
			} else if len(allResults) > 0 {
				cmd.Println("Updated tsconfig paths.")
			} else {
				cmd.Printf("Ensured tsconfig exists: %s\n", tsconfigPath)
			}
			if err := esmresolver.EnsureTsconfigCompilerTypeRoots(tsconfigPath, typesDir, compilerTypeRootLinks); err != nil {
				cmd.Printf("Warning: failed to update tsconfig typeRoots bridges: %v\n", err)
			}

			if offline {
				cmd.Println("(offline mode — only cached types were used)")
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&all, "all", false, "fetch types for all installed modules")
	cmd.Flags().BoolVar(&withDepends, "with-depends", true, "include choysum.depends closure for single-app fetch and validate depends completeness for multi-app fetch")
	cmd.Flags().StringVar(&missingDepPolicy, "missing-dep-policy", "", "policy for missing choysum.depends modules: error|warn (default: error for single-app, warn otherwise)")
	cmd.Flags().StringVar(&upstream, "upstream", "", "override ESM upstream URL")
	cmd.Flags().BoolVar(&offline, "offline", false, "use only cached types, do not fetch")

	return cmd
}

type typeFetchModulePackage struct {
	Choysum struct {
		Depends []string `json:"depends"`
	} `json:"choysum"`
}

type typeFetchCompilerTypeTargetsConfig struct {
	CompilerOptions struct {
		Types []string `json:"types"`
	} `json:"compilerOptions"`
}

type typeFetchCompilerTypeTarget struct {
	TypeName    string
	PackageName string
	Version     string
}

const (
	typeFetchMissingDepPolicyError = "error"
	typeFetchMissingDepPolicyWarn  = "warn"
)

func resolveTypeFetchCompilerTypeTargets(tsconfigPath string, modulesPath string) ([]typeFetchCompilerTypeTarget, error) {
	data, err := os.ReadFile(tsconfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, xfmt.Errorf("type-fetch: read modules tsconfig: %w", err)
	}
	if strings.TrimSpace(string(data)) == "" {
		return nil, nil
	}

	var cfg typeFetchCompilerTypeTargetsConfig
	if err := json.Unmarshal(esmresolver.StripJSONComments(data), &cfg); err != nil {
		return nil, xfmt.Errorf("type-fetch: parse modules tsconfig: %w", err)
	}

	seen := make(map[string]struct{})
	targets := make([]typeFetchCompilerTypeTarget, 0, len(cfg.CompilerOptions.Types))
	for _, rawType := range cfg.CompilerOptions.Types {
		packageName, explicitVersion, ok := resolveTypeFetchCompilerTypePackage(rawType)
		if !ok {
			continue
		}
		version := resolveTypeFetchCompilerTypeVersion(modulesPath, packageName, explicitVersion)
		key := packageName + "@" + version
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		targets = append(targets, typeFetchCompilerTypeTarget{
			TypeName:    strings.TrimSpace(rawType),
			PackageName: packageName,
			Version:     version,
		})
	}

	return targets, nil
}

func resolveTypeFetchCompilerTypePackage(rawType string) (string, string, bool) {
	typeName := strings.TrimSpace(rawType)
	if typeName == "" {
		return "", "", false
	}

	name, version := splitTypeFetchNameAndVersion(typeName)
	if name == "" {
		return "", "", false
	}

	// Reject path traversal segments in package names and versions extracted
	// from user-controlled tsconfig before they reach filesystem operations.
	if containsPathTraversal(name) || containsPathTraversal(version) {
		return "", "", false
	}

	if strings.HasPrefix(name, "@types/") {
		return name, version, true
	}

	if strings.HasPrefix(name, "@") {
		parts := strings.Split(strings.TrimPrefix(name, "@"), "/")
		if len(parts) < 2 {
			return "", "", false
		}
		return "@types/" + parts[0] + "__" + parts[1], version, true
	}

	base := name
	if idx := strings.Index(base, "/"); idx >= 0 {
		base = base[:idx]
	}
	base = strings.TrimSpace(base)
	if base == "" {
		return "", "", false
	}
	return "@types/" + base, version, true
}

func containsPathTraversal(s string) bool {
	// Split on both / and \ so backslash-based traversal (relevant on
	// Windows) is caught alongside forward-slash variants.
	for _, part := range strings.FieldsFunc(s, func(r rune) bool { return r == '/' || r == '\\' }) {
		part = strings.TrimSpace(part)
		if part == ".." || part == "." {
			return true
		}
	}
	return false
}

func splitTypeFetchNameAndVersion(typeName string) (string, string) {
	typeName = strings.TrimSpace(typeName)
	if typeName == "" {
		return "", ""
	}

	if strings.HasPrefix(typeName, "@") {
		slash := strings.Index(typeName, "/")
		if slash > 0 {
			if at := strings.LastIndex(typeName, "@"); at > slash {
				return strings.TrimSpace(typeName[:at]), strings.TrimSpace(typeName[at+1:])
			}
		}
		return typeName, ""
	}

	if at := strings.LastIndex(typeName, "@"); at > 0 {
		return strings.TrimSpace(typeName[:at]), strings.TrimSpace(typeName[at+1:])
	}

	return typeName, ""
}

func resolveTypeFetchCompilerTypeVersion(modulesPath string, packageName string, explicitVersion string) string {
	version := strings.TrimSpace(explicitVersion)
	if version != "" {
		return version
	}

	modulesPath = strings.TrimSpace(modulesPath)
	searchRoots := []string{
		filepath.Join(modulesPath, "node_modules"),
		filepath.Join(filepath.Dir(modulesPath), "node_modules"),
	}
	for _, root := range searchRoots {
		pkgVersion, ok := readTypeFetchPackageVersion(root, packageName)
		if ok {
			return pkgVersion
		}
	}

	return "latest"
}

func readTypeFetchPackageVersion(nodeModulesRoot string, packageName string) (string, bool) {
	packageJSONPath := filepath.Join(strings.TrimSpace(nodeModulesRoot), filepath.FromSlash(packageName), "package.json")
	data, err := os.ReadFile(packageJSONPath)
	if err != nil {
		return "", false
	}

	var pkg struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return "", false
	}
	version := strings.TrimSpace(pkg.Version)
	if version == "" {
		return "", false
	}
	return version, true
}

func resolveTypeFetchMissingDepPolicy(raw string, singleModuleMode bool) (string, error) {
	policy := strings.ToLower(strings.TrimSpace(raw))
	if policy == "" {
		if singleModuleMode {
			return typeFetchMissingDepPolicyError, nil
		}
		return typeFetchMissingDepPolicyWarn, nil
	}
	switch policy {
	case typeFetchMissingDepPolicyError, typeFetchMissingDepPolicyWarn:
		return policy, nil
	default:
		return "", xfmt.Errorf("type-fetch: invalid --missing-dep-policy %q (want error|warn)", raw)
	}
}

func readTypeFetchModulePackage(path string) (typeFetchModulePackage, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return typeFetchModulePackage{}, err
	}
	if strings.TrimSpace(string(data)) == "" {
		return typeFetchModulePackage{}, nil
	}
	var pkg typeFetchModulePackage
	if err := json.Unmarshal(data, &pkg); err != nil {
		return typeFetchModulePackage{}, xfmt.Errorf("parse package.json: %w", err)
	}
	return pkg, nil
}

func typeFetchModulePackagePath(modulesPath string, moduleName string) (string, error) {
	modulesRoot := strings.TrimSpace(modulesPath)
	if modulesRoot == "" {
		return "", xfmt.Errorf("modules path is required")
	}
	trimmedModule := strings.TrimSpace(moduleName)
	if trimmedModule == "" {
		return "", xfmt.Errorf("module name is required")
	}

	rootAbs, err := filepath.Abs(modulesRoot)
	if err != nil {
		return "", xfmt.Errorf("resolve modules path: %w", err)
	}

	pkgPath := filepath.Clean(filepath.Join(rootAbs, trimmedModule, "package.json"))
	rel, err := filepath.Rel(rootAbs, pkgPath)
	if err != nil {
		return "", xfmt.Errorf("resolve module path %q: %w", moduleName, err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", xfmt.Errorf("invalid module path %q", moduleName)
	}

	return pkgPath, nil
}

func resolveTypeFetchDependsClosure(modulesPath string, rootModule string) ([]string, []string, error) {
	rootModule = strings.TrimSpace(rootModule)
	if rootModule == "" {
		return nil, nil, xfmt.Errorf("missing module name")
	}

	queue := []string{rootModule}
	seen := make(map[string]struct{})
	closure := make([]string, 0)
	missingSet := make(map[string]struct{})

	for len(queue) > 0 {
		moduleName := strings.TrimSpace(queue[0])
		queue = queue[1:]
		if moduleName == "" {
			continue
		}
		if _, ok := seen[moduleName]; ok {
			continue
		}
		seen[moduleName] = struct{}{}

		pkgPath, err := typeFetchModulePackagePath(modulesPath, moduleName)
		if err != nil {
			return nil, nil, xfmt.Errorf("%s: %w", moduleName, err)
		}
		pkg, err := readTypeFetchModulePackage(pkgPath)
		if err != nil {
			if os.IsNotExist(err) && moduleName != rootModule {
				missingSet[moduleName] = struct{}{}
				continue
			}
			return nil, nil, xfmt.Errorf("%s: %w", moduleName, err)
		}

		closure = append(closure, moduleName)
		for _, dep := range pkg.Choysum.Depends {
			depModule := strings.TrimSpace(dep)
			if depModule == "" {
				continue
			}
			if _, ok := seen[depModule]; ok {
				continue
			}

			depPkgPath, err := typeFetchModulePackagePath(modulesPath, depModule)
			if err != nil {
				return nil, nil, xfmt.Errorf("resolve depends module %q for %q: %w", depModule, moduleName, err)
			}
			if _, err := os.Stat(depPkgPath); err != nil {
				if os.IsNotExist(err) {
					missingSet[depModule] = struct{}{}
					continue
				}
				return nil, nil, xfmt.Errorf("stat depends module %q for %q: %w", depModule, moduleName, err)
			}
			queue = append(queue, depModule)
		}
	}

	missingModules := make([]string, 0, len(missingSet))
	for name := range missingSet {
		missingModules = append(missingModules, name)
	}
	sort.Strings(missingModules)

	return closure, missingModules, nil
}

func validateTypeFetchDependsCompleteness(modulesPath string, moduleNames []string) ([]string, error) {
	availableModules := make(map[string]struct{}, len(moduleNames))
	for _, moduleName := range moduleNames {
		trimmed := strings.TrimSpace(moduleName)
		if trimmed == "" {
			continue
		}
		availableModules[trimmed] = struct{}{}
	}

	missingSet := make(map[string]struct{})
	for _, moduleName := range moduleNames {
		trimmed := strings.TrimSpace(moduleName)
		if trimmed == "" {
			continue
		}

		pkgPath, err := typeFetchModulePackagePath(modulesPath, trimmed)
		if err != nil {
			return nil, xfmt.Errorf("%s: %w", trimmed, err)
		}
		pkg, err := readTypeFetchModulePackage(pkgPath)
		if err != nil {
			return nil, xfmt.Errorf("%s: %w", trimmed, err)
		}

		for _, dep := range pkg.Choysum.Depends {
			depModule := strings.TrimSpace(dep)
			if depModule == "" {
				continue
			}
			if _, ok := availableModules[depModule]; ok {
				continue
			}

			depPkgPath, err := typeFetchModulePackagePath(modulesPath, depModule)
			if err != nil {
				return nil, xfmt.Errorf("resolve depends module %q for %q: %w", depModule, trimmed, err)
			}
			if _, err := os.Stat(depPkgPath); err != nil {
				if !os.IsNotExist(err) {
					return nil, xfmt.Errorf("stat depends module %q for %q: %w", depModule, trimmed, err)
				}
			}
			missingSet[depModule] = struct{}{}
		}
	}

	missingModules := make([]string, 0, len(missingSet))
	for name := range missingSet {
		missingModules = append(missingModules, name)
	}
	sort.Strings(missingModules)

	return missingModules, nil
}

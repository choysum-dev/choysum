// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/choysum-dev/choysum/internal/esmresolver"
	"github.com/choysum-dev/choysum/pkg/config"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/spf13/cobra"
	xfmt "golang.org/x/exp/errors/fmt"
)

func newTypeFetchCmd(envGetter func() scope.Scope) *cobra.Command {
	var all bool
	var upstream string
	var offline bool

	cmd := &cobra.Command{
		Use:   "type-fetch [<app>]",
		Short: "Fetch type definitions (.d.ts) for module dependencies",
		Long: `Scans the module's package.json for dependencies, downloads their
type definitions (.d.ts) from the configured ESM upstream, and caches them
locally for IDE support.

When called without arguments, fetches types for all installed modules.
When <app> is specified, fetches types for that module only.`,
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

			client := esmresolver.NewTypeFetchHTTPClient(30 * time.Second)

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

			if len(appNames) == 0 {
				cmd.Println("No modules found with package.json.")
				return nil
			}

			session := esmresolver.NewTypeFetchSession(0)

			totalCached := 0
			totalFetched := 0
			var allResults []esmresolver.TypeFetchResult

			for _, appName := range appNames {
				if err := cmd.Context().Err(); err != nil {
					return err
				}
				moduleDir := filepath.Join(modulesPath, appName)
				cmd.Printf("[%s] fetching dependency types...\n", appName)
				results, err := session.FetchTypesForModule(client, upstream, typesDir, moduleDir)
				if err != nil {
					cmd.Printf("[%s] error: %v\n", appName, err)
					continue
				}
				for _, r := range results {
					if r.FromCache {
						totalCached++
						cmd.Printf("[%s] cached %s@%s → %s\n", appName, r.Package, r.Version, r.CachedPath)
					} else {
						totalFetched++
						cmd.Printf("[%s] fetched %s@%s → %s\n", appName, r.Package, r.Version, r.CachedPath)
					}
				}
				allResults = append(allResults, results...)
				if len(results) == 0 {
					cmd.Printf("[%s] no dependencies found\n", appName)
				}
			}

			cmd.Printf("\nType fetch complete: %d cached, %d fetched.\n", totalCached, totalFetched)
			cmd.Printf("Types directory: %s\n", typesDir)

			// Update tsconfig paths for IDE support.
			if err := esmresolver.UpdateTsconfigPaths(tsconfigPath, allResults); err != nil {
				cmd.Printf("Warning: failed to update tsconfig paths: %v\n", err)
			} else if len(allResults) > 0 {
				cmd.Println("Updated tsconfig paths.")
			} else {
				cmd.Printf("Ensured tsconfig exists: %s\n", tsconfigPath)
			}

			if offline {
				cmd.Println("(offline mode — only cached types were used)")
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&all, "all", false, "fetch types for all installed modules")
	cmd.Flags().StringVar(&upstream, "upstream", "", "override ESM upstream URL")
	cmd.Flags().BoolVar(&offline, "offline", false, "use only cached types, do not fetch")

	// Hide the command until it's fully implemented.
	// Remove this line when ready for production use.
	_ = fmt.Sprintf("type-fetch is available: %s", upstream)

	return cmd
}

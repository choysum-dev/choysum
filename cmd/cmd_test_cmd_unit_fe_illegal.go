// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	cliruntime "github.com/choysum-dev/choysum/internal/cli/runtime"
	"github.com/choysum-dev/choysum/internal/testing/frontend"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/spf13/cobra"
	xfmt "golang.org/x/exp/errors/fmt"
)

func newTestUnitFEIllegalCmd(_ func() scope.Scope, runtimeOptionsGetter func() cliruntime.Options) *cobra.Command {
	var all bool
	var githubAnnotations bool
	var failOnIllegal bool

	cmd := &cobra.Command{
		Use:   "unit-fe-illegal [app]",
		Short: "Inventory FE unit tests still on Node DOM/VTU (warn by default)",
		Long: strings.TrimSpace(`
Scan modules/<app>/web unit tests for legacy Node/Vitest DOM patterns:
happy-dom/jsdom environment pragmas or imports, @vue/test-utils, and *.vue imports.

Hits are an inventory of tests still on the Vitest/VTU stack (to migrate onto the
QuickJS + vuesfc + choysumMount host). They are not a mandate to delete mount tests.
Importing from 'vitest' is allowed until FE hard-cut. Default mode exits 0 and
prints warnings (optionally as GitHub Actions annotations).
`),
		Args: func(cmd *cobra.Command, args []string) error {
			if all {
				if len(args) != 0 {
					return xfmt.Errorf("test unit-fe-illegal: --all cannot be used with an app argument")
				}
				return nil
			}
			if len(args) != 1 {
				return xfmt.Errorf("test unit-fe-illegal: requires exactly 1 app argument (or use --all)")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			runtimeOptions, err := cliruntime.RequireOptionsForCommand("test unit-fe-illegal", runtimeOptionsGetter)
			if err != nil {
				return err
			}
			modulesPath := strings.TrimSpace(runtimeOptions.ModulesPath)
			// ModulesPath is normally <repo>/modules; if it already is the repo root
			// (contains a modules/ child), keep it. Otherwise take the parent.
			repoRoot := modulesPath
			if st, err := os.Stat(filepath.Join(modulesPath, "modules")); err != nil || !st.IsDir() {
				repoRoot = filepath.Dir(modulesPath)
			}
			if st, statErr := os.Stat(filepath.Join(repoRoot, "modules")); statErr != nil || !st.IsDir() {
				// Layout is unusual; fall back to cwd when it looks like a workspace root.
				if cwd, cwdErr := os.Getwd(); cwdErr == nil {
					if st2, err2 := os.Stat(filepath.Join(cwd, "modules")); err2 == nil && st2.IsDir() {
						repoRoot = cwd
					}
				}
			}

			apps := args
			if !all {
				for _, app := range apps {
					appDir := filepath.Join(repoRoot, "modules", app)
					if st, err := os.Stat(appDir); err != nil || !st.IsDir() {
						return xfmt.Errorf("test unit-fe-illegal: module %q does not exist", app)
					}
				}
			}
			if all {
				modulesRoot := filepath.Join(repoRoot, "modules")
				entries, readErr := os.ReadDir(modulesRoot)
				if readErr != nil {
					return xfmt.Errorf("test unit-fe-illegal: read modules: %w", readErr)
				}
				apps = nil
				for _, ent := range entries {
					if !ent.IsDir() || strings.HasPrefix(ent.Name(), ".") {
						continue
					}
					webDir := filepath.Join(modulesRoot, ent.Name(), "web")
					if st, err := os.Stat(webDir); err == nil && st.IsDir() {
						apps = append(apps, ent.Name())
					}
				}
			}

			var allHits []frontend.IllegalMark
			for _, app := range apps {
				hits, scanErr := frontend.CheckIllegalFrontendMarks(repoRoot, app, frontend.ScanModeWarn)
				if scanErr != nil {
					return xfmt.Errorf("test unit-fe-illegal: %w", scanErr)
				}
				allHits = append(allHits, hits...)
			}

			out := cmd.ErrOrStderr()
			if githubAnnotations {
				if ann := frontend.FormatIllegalMarksGitHubAnnotations(allHits, repoRoot); ann != "" {
					_, _ = out.Write([]byte(ann))
				} else {
					fmt.Fprintf(out, "choysum test unit-fe-illegal: no illegal FE marks in %d app(s)\n", len(apps))
				}
			} else if msg := frontend.FormatIllegalMarksWarn(allHits, repoRoot); msg != "" {
				_, _ = out.Write([]byte(msg))
			} else {
				fmt.Fprintf(out, "choysum test unit-fe-illegal: no illegal FE marks in %d app(s)\n", len(apps))
			}

			if failOnIllegal && len(allHits) > 0 {
				return xfmt.Errorf("test unit-fe-illegal: %d illegal mark(s)", len(allHits))
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&all, "all", false, "scan all apps that have FE unit tests")
	cmd.Flags().BoolVar(&githubAnnotations, "github-annotations", false, "emit GitHub Actions ::warning annotations")
	cmd.Flags().BoolVar(&failOnIllegal, "fail", false, "exit non-zero when illegal marks are found (hard-cut mode)")
	return cmd
}

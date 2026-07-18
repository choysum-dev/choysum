// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/choysum-dev/choysum/internal/i18n/extract"
	i18nstatus "github.com/choysum-dev/choysum/internal/i18n/status"
	i18nsync "github.com/choysum-dev/choysum/internal/i18n/sync"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/spf13/cobra"
	xfmt "golang.org/x/exp/errors/fmt"
)

func newI18nCmd(envGetter func() scope.Scope) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "i18n",
		Short: "Terminology extract, sync, and status tools",
		Long: `Development tooling for terminology i18n (gettext PO catalogs).

extract scans module source for explicit _t literals and writes modules/<m>/i18n/<m>.pot.
sync merges pot into language .po files (msgmerge semantics).
status reports missing/fuzzy/orphan/pot-dirty findings for CI gates.
Vue <template> extraction uses a temporary regex (S1-MVP); S1+ will use a full template AST.`,
		Annotations: map[string]string{
			lightweightScopeAnnotation: "true",
		},
	}
	cmd.AddCommand(newI18nExtractCmd(envGetter), newI18nSyncCmd(envGetter), newI18nStatusCmd(envGetter))
	return cmd
}

func newI18nExtractCmd(envGetter func() scope.Scope) *cobra.Command {
	var all bool

	cmd := &cobra.Command{
		Use:   "extract [module...]",
		Short: "Extract explicit _t literals into module .pot catalogs",
		Annotations: map[string]string{
			lightweightScopeAnnotation: "true",
		},
		Args: func(cmd *cobra.Command, args []string) error {
			if all && len(args) > 0 {
				return xfmt.Errorf("i18n extract: --all cannot be used with module arguments")
			}
			if !all && len(args) == 0 {
				return xfmt.Errorf("i18n extract: provide module name(s) or --all")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			env, err := requireCommandScope(envGetter)
			if err != nil {
				return err
			}
			runtimeOpts, ok := scope.PathsRuntimeOptionsFromScope(env)
			if !ok {
				return xfmt.Errorf("i18n extract: missing runtime options")
			}
			modulesPath := strings.TrimSpace(runtimeOpts.ModulesPath)
			if modulesPath == "" {
				return xfmt.Errorf("i18n extract: modules path is empty")
			}

			modules, err := resolveI18nModules(modulesPath, all, args)
			if err != nil {
				return err
			}

			pathAlias, err := extract.LoadPathAliasFromModulesTsconfig(modulesPath)
			if err != nil {
				return xfmt.Errorf("i18n extract: load path alias: %w", err)
			}

			var warnCount int
			for _, moduleName := range modules {
				moduleRoot := filepath.Join(modulesPath, moduleName)
				result, err := extract.ExtractModule(moduleRoot, moduleName, pathAlias, true)
				if err != nil {
					return xfmt.Errorf("i18n extract %s: %w", moduleName, err)
				}
				for _, issue := range result.Issues {
					warnCount++
					fmt.Fprintf(cmd.ErrOrStderr(), "warn[%s] %s:%d:%d: %s\n", issue.Code, issue.SourcePath, issue.Line, issue.Col, issue.Message)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "wrote %s (%d entries)\n", result.PotPath, len(result.Entries))
			}
			if warnCount > 0 {
				fmt.Fprintf(cmd.ErrOrStderr(), "i18n extract completed with %d warning(s)\n", warnCount)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&all, "all", false, "extract all modules under modules path")
	return cmd
}

func newI18nSyncCmd(envGetter func() scope.Scope) *cobra.Command {
	var all bool
	var lang string

	cmd := &cobra.Command{
		Use:   "sync [module...]",
		Short: "Sync module .po files from .pot (msgmerge semantics)",
		Long: `Update language .po catalogs from the module .pot.

Preserves existing msgstr for matching (msgctxt, msgid), adds new empty entries,
and marks removed pot entries obsolete (#~) without deleting translation history.`,
		Annotations: map[string]string{
			lightweightScopeAnnotation: "true",
		},
		Args: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(lang) == "" {
				return xfmt.Errorf("i18n sync: --lang is required")
			}
			if all && len(args) > 0 {
				return xfmt.Errorf("i18n sync: --all cannot be used with module arguments")
			}
			if !all && len(args) == 0 {
				return xfmt.Errorf("i18n sync: provide module name(s) or --all")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			env, err := requireCommandScope(envGetter)
			if err != nil {
				return err
			}
			runtimeOpts, ok := scope.PathsRuntimeOptionsFromScope(env)
			if !ok {
				return xfmt.Errorf("i18n sync: missing runtime options")
			}
			modulesPath := strings.TrimSpace(runtimeOpts.ModulesPath)
			if modulesPath == "" {
				return xfmt.Errorf("i18n sync: modules path is empty")
			}

			modules, err := resolveI18nModules(modulesPath, all, args)
			if err != nil {
				return err
			}

			for _, moduleName := range modules {
				moduleRoot := filepath.Join(modulesPath, moduleName)
				result, err := i18nsync.SyncModulePo(moduleRoot, moduleName, lang)
				if err != nil {
					return xfmt.Errorf("i18n sync %s: %w", moduleName, err)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "synced %s (kept=%d added=%d obsolete=%d)\n",
					result.PoPath, result.Kept, result.Added, result.Obsoleted)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&all, "all", false, "sync all modules under modules path")
	cmd.Flags().StringVar(&lang, "lang", "", "language code for the .po file (e.g. zh_CN)")
	return cmd
}

func newI18nStatusCmd(envGetter func() scope.Scope) *cobra.Command {
	var all bool
	var lang string
	var skipPot bool
	var asJSON bool
	var strictOrphan bool

	cmd := &cobra.Command{
		Use:   "status [module...]",
		Short: "Report missing/fuzzy/orphan/pot-dirty terminology findings",
		Long: `Scan module PO/POT catalogs and exit non-zero when CI-relevant issues exist.

Findings:
  missing    — non-obsolete entry with empty msgstr
  fuzzy      — entry flagged fuzzy
  orphan     — obsolete (#~) retained in PO, or PO entry absent from pot (D12a; report-only unless --strict-orphan)
  pot-dirty  — committed .pot drifts from extract of current sources
  no-po      — pot exists but language .po is missing

Filesystem-only (no DB). Safe for lightweight CI.
Default fail-on: missing, fuzzy, pot-dirty, no-po. Orphans are reported but do not fail unless --strict-orphan.`,
		Annotations: map[string]string{
			lightweightScopeAnnotation: "true",
		},
		Args: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(lang) == "" {
				return xfmt.Errorf("i18n status: --lang is required")
			}
			if all && len(args) > 0 {
				return xfmt.Errorf("i18n status: --all cannot be used with module arguments")
			}
			if !all && len(args) == 0 {
				return xfmt.Errorf("i18n status: provide module name(s) or --all")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			env, err := requireCommandScope(envGetter)
			if err != nil {
				return err
			}
			runtimeOpts, ok := scope.PathsRuntimeOptionsFromScope(env)
			if !ok {
				return xfmt.Errorf("i18n status: missing runtime options")
			}
			modulesPath := strings.TrimSpace(runtimeOpts.ModulesPath)
			if modulesPath == "" {
				return xfmt.Errorf("i18n status: modules path is empty")
			}

			modules, err := resolveI18nModules(modulesPath, all, args)
			if err != nil {
				return err
			}

			pathAlias, err := extract.LoadPathAliasFromModulesTsconfig(modulesPath)
			if err != nil {
				return xfmt.Errorf("i18n status: load path alias: %w", err)
			}

			report, err := i18nstatus.StatusReport(i18nstatus.Options{
				ModulesPath:  modulesPath,
				Modules:      modules,
				Lang:         lang,
				PathAlias:    pathAlias,
				SkipPotCheck: skipPot,
			})
			if err != nil {
				return err
			}

			if asJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				if err := enc.Encode(report); err != nil {
					return err
				}
			} else {
				fmt.Fprint(cmd.OutOrStdout(), i18nstatus.FormatText(report))
			}

			if i18nstatus.ExitCode(report, i18nstatus.ExitOptions{StrictOrphan: strictOrphan}) != 0 {
				return xfmt.Errorf("i18n status: blocking issue(s) found")
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&all, "all", false, "check all modules under modules path")
	cmd.Flags().StringVar(&lang, "lang", "", "language code for the .po file (e.g. zh_CN)")
	cmd.Flags().BoolVar(&skipPot, "skip-pot-check", false, "skip extract-vs-committed .pot drift detection")
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit machine-readable JSON")
	cmd.Flags().BoolVar(&strictOrphan, "strict-orphan", false, "treat obsolete/orphan findings as failures")
	return cmd
}

func resolveI18nModules(modulesPath string, all bool, args []string) ([]string, error) {
	if !all {
		out := make([]string, 0, len(args))
		for _, name := range args {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			moduleRoot := filepath.Join(modulesPath, name)
			info, err := os.Stat(moduleRoot)
			if err != nil || !info.IsDir() {
				return nil, xfmt.Errorf("i18n extract: module %q not found under %s", name, modulesPath)
			}
			out = append(out, name)
		}
		if len(out) == 0 {
			return nil, xfmt.Errorf("i18n extract: no modules specified")
		}
		return out, nil
	}

	entries, err := os.ReadDir(modulesPath)
	if err != nil {
		return nil, xfmt.Errorf("i18n extract: read modules path: %w", err)
	}
	var modules []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		// Skip non-module dirs commonly present under modules/.
		switch name {
		case "node_modules", "dist":
			continue
		}
		modules = append(modules, name)
	}
	return modules, nil
}

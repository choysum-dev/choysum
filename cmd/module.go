// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cmd

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	metadata "github.com/choysum-dev/choysum/internal/module/metadata"

	internalorigin "github.com/choysum-dev/choysum/internal/module/origin"
	sourceregistry "github.com/choysum-dev/choysum/internal/module/origin/registry"
	"github.com/choysum-dev/choysum/pkg/meta"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/spf13/cobra"
	xfmt "golang.org/x/exp/errors/fmt"
)

func newModuleCmd(envGetter func() scope.Scope, runtimeOptionsGetter func() cliRuntimeOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "module",
		Short: "Inspect and manage module sources",
	}

	cmd.AddCommand(
		newModuleSearchCmd(envGetter, runtimeOptionsGetter),
		newModuleInfoCmd(envGetter, runtimeOptionsGetter),
		newModuleListCmd(envGetter, runtimeOptionsGetter),
		newModuleFetchCmd(envGetter),
		newModulePurgeCmd(envGetter),
	)

	return cmd
}

func newModuleSearchCmd(envGetter func() scope.Scope, runtimeOptionsGetter func() cliRuntimeOptions) *cobra.Command {
	var remote bool
	var registryAlias string
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search modules from local workspace or remote registry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			env, err := requireCommandScope(envGetter)
			if err != nil {
				return err
			}
			runtimeOptions, err := requireCliRuntimeOptions(runtimeOptionsGetter)
			if err != nil {
				return xfmt.Errorf("module: invalid runtime options: %w", err)
			}
			if remote {
				return runModuleSearchRemote(cmd, env, runtimeOptions, registryAlias, args[0])
			}
			return runModuleSearchLocal(cmd, env, runtimeOptions, args[0])
		},
	}
	cmd.Flags().BoolVar(&remote, "remote", false, "query modules from remote registry catalog")
	cmd.Flags().StringVar(&registryAlias, "registry", sourceregistry.DefaultRegistryAlias, "registry alias for --remote queries")
	return cmd
}

func runModuleSearchLocal(cmd *cobra.Command, runtimeScope scope.Scope, runtimeOptions cliRuntimeOptions, rawQuery string) error {
	query := strings.ToLower(strings.TrimSpace(rawQuery))
	if query == "" {
		return xfmt.Errorf("query is required")
	}

	workspaceRoot := internalorigin.WorkspaceRoot(runtimeScope)
	store := internalorigin.NewLockStore(internalorigin.WithLockStoreDefaultChoysumPath(runtimeOptions.defaultChoysumPath))
	lockFile, err := store.Read(workspaceRoot)
	if err != nil {
		return xfmt.Errorf("read modules lock: %w", err)
	}

	results := map[string]struct{}{}
	for moduleName := range lockFile.Modules {
		if strings.Contains(strings.ToLower(moduleName), query) {
			results[moduleName] = struct{}{}
		}
	}

	if entries, err := os.ReadDir(runtimeOptions.modulesPath); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			name := entry.Name()
			if strings.Contains(strings.ToLower(name), query) {
				results[name] = struct{}{}
			}
		}
	}

	if len(results) == 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "No modules matched query %q\n", rawQuery)
		return nil
	}

	names := make([]string, 0, len(results))
	for name := range results {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		fmt.Fprintln(cmd.OutOrStdout(), name)
	}
	return nil
}

func runModuleSearchRemote(cmd *cobra.Command, runtimeScope scope.Scope, runtimeOptions cliRuntimeOptions, registryAlias, rawQuery string) error {
	query := strings.TrimSpace(rawQuery)
	if query == "" {
		return xfmt.Errorf("query is required")
	}

	items, err := listRemoteCatalogModules(cmd, runtimeScope, runtimeOptions, registryAlias, query)
	if err != nil {
		return err
	}
	if len(items) == 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "No remote modules matched query %q in registry %q\n", rawQuery, strings.TrimSpace(registryAlias))
		return nil
	}

	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "MODULE\tLATEST\tDESCRIPTION")
	for _, item := range items {
		fmt.Fprintf(w, "%s\t%s\t%s\n", item.Name, item.LatestVersion, item.Description)
	}
	_ = w.Flush()
	return nil
}

func newModuleInfoCmd(envGetter func() scope.Scope, runtimeOptionsGetter func() cliRuntimeOptions) *cobra.Command {
	var remote bool
	var registryAlias string
	cmd := &cobra.Command{
		Use:   "info <input>",
		Short: "Inspect source metadata for a module input",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			env, err := requireCommandScope(envGetter)
			if err != nil {
				return err
			}
			if remote {
				runtimeOptions, err := requireCliRuntimeOptions(runtimeOptionsGetter)
				if err != nil {
					return xfmt.Errorf("module: invalid runtime options: %w", err)
				}
				return runModuleInfoRemote(cmd, env, runtimeOptions, registryAlias, args[0])
			}
			return runModuleInfoLocal(cmd, envGetter, args[0])
		},
	}
	cmd.Flags().BoolVar(&remote, "remote", false, "query module info from remote registry catalog")
	cmd.Flags().StringVar(&registryAlias, "registry", sourceregistry.DefaultRegistryAlias, "registry alias for --remote lookup when input is local module name")
	return cmd
}

func runModuleInfoLocal(cmd *cobra.Command, envGetter func() scope.Scope, input string) error {
	coordinator, ctx, err := newCoordinatorForCommand(envGetter, cmd)
	if err != nil {
		return err
	}

	module, err := coordinator.Peek(ctx, strings.TrimSpace(input))
	if err != nil {
		return err
	}
	payloadMap := map[string]any{}
	payloadRaw, err := json.Marshal(module)
	if err != nil {
		return xfmt.Errorf("marshal module info: %w", err)
	}
	if err := json.Unmarshal(payloadRaw, &payloadMap); err != nil {
		return xfmt.Errorf("decode module info payload: %w", err)
	}

	env := envGetter()
	parsed, parseErr := internalorigin.ParseInput(strings.TrimSpace(input))
	if parseErr == nil && parsed.Kind == internalorigin.InputKindLocal && env != nil {
		if views, hasIndex, queryErr := queryModuleIndexViews(env, strings.TrimSpace(parsed.ModuleName)); queryErr != nil {
			return queryErr
		} else if hasIndex && len(views) > 0 {
			payloadMap["source"] = moduleIndexViewToPayload(views[0])
		}
	}

	payload, err := json.MarshalIndent(payloadMap, "", "  ")
	if err != nil {
		return xfmt.Errorf("marshal module info: %w", err)
	}
	fmt.Fprintln(cmd.OutOrStdout(), string(payload))
	return nil
}

func runModuleInfoRemote(cmd *cobra.Command, runtimeScope scope.Scope, runtimeOptions cliRuntimeOptions, registryAlias, input string) error {
	raw := strings.TrimSpace(input)
	if raw == "" {
		return xfmt.Errorf("module input is required")
	}

	moduleName := raw
	if parsed, err := internalorigin.ParseInput(raw); err == nil {
		if parsed.Kind == internalorigin.InputKindRegistry {
			registryAlias = parsed.RegistryAlias
			moduleName = parsed.ModuleName
			version := strings.TrimSpace(parsed.Version)
			if version != "" && !strings.EqualFold(version, "latest") {
				coordinator, ctx, err := newCoordinatorForCommand(func() scope.Scope { return runtimeScope }, cmd)
				if err != nil {
					return err
				}
				module, err := coordinator.Peek(ctx, parsed.CanonicalRef())
				if err != nil {
					return err
				}
				payload, err := json.MarshalIndent(module, "", "  ")
				if err != nil {
					return xfmt.Errorf("marshal module info: %w", err)
				}
				fmt.Fprintln(cmd.OutOrStdout(), string(payload))
				return nil
			}
		} else {
			moduleName = strings.TrimSpace(parsed.ModuleName)
		}
	} else if strings.Contains(raw, "/") {
		return err
	}

	item, err := loadRemoteModuleInfo(cmd, runtimeScope, runtimeOptions, registryAlias, moduleName)
	if err != nil {
		return err
	}
	payload, err := json.MarshalIndent(item, "", "  ")
	if err != nil {
		return xfmt.Errorf("marshal remote module info: %w", err)
	}
	fmt.Fprintln(cmd.OutOrStdout(), string(payload))
	return nil
}

func newModuleListCmd(envGetter func() scope.Scope, runtimeOptionsGetter func() cliRuntimeOptions) *cobra.Command {
	var remote bool
	var registryAlias string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List modules in source lock bindings or remote registry",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			env, err := requireCommandScope(envGetter)
			if err != nil {
				return err
			}
			runtimeOptions, err := requireCliRuntimeOptions(runtimeOptionsGetter)
			if err != nil {
				return xfmt.Errorf("module: invalid runtime options: %w", err)
			}
			if remote {
				return runModuleListRemote(cmd, env, runtimeOptions, registryAlias)
			}
			return runModuleListLocal(cmd, env, runtimeOptions)
		},
	}
	cmd.Flags().BoolVar(&remote, "remote", false, "list modules from remote registry catalog")
	cmd.Flags().StringVar(&registryAlias, "registry", sourceregistry.DefaultRegistryAlias, "registry alias for --remote list")
	return cmd
}

func runModuleListLocal(cmd *cobra.Command, runtimeScope scope.Scope, runtimeOptions cliRuntimeOptions) error {
	if views, hasIndex, err := queryModuleIndexViews(runtimeScope, ""); err != nil {
		return err
	} else if hasIndex && len(views) > 0 {
		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
		fmt.Fprintln(w, "MODULE\tSOURCE\tREF\tVERSION\tPATH\tINSTALLED\tAVAILABLE")
		for _, view := range views {
			fmt.Fprintf(
				w,
				"%s\t%s\t%s\t%s\t%s\t%s\t%t\n",
				view.ModuleName,
				view.OriginType,
				view.OriginRef,
				nullStringValue(view.Version),
				nullStringValue(view.LocalPath),
				nullStringValue(view.InstallStatus),
				view.Available,
			)
		}
		_ = w.Flush()
		return nil
	}

	workspaceRoot := internalorigin.WorkspaceRoot(runtimeScope)
	store := internalorigin.NewLockStore(internalorigin.WithLockStoreDefaultChoysumPath(runtimeOptions.defaultChoysumPath))
	lockFile, err := store.Read(workspaceRoot)
	if err != nil {
		return xfmt.Errorf("read modules lock: %w", err)
	}
	if len(lockFile.Modules) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No source bindings found.")
		return nil
	}

	names := make([]string, 0, len(lockFile.Modules))
	for name := range lockFile.Modules {
		names = append(names, name)
	}
	sort.Strings(names)

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "MODULE\tSOURCE\tREF\tVERSION\tPATH")
	for _, name := range names {
		binding := lockFile.Modules[name]
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", name, binding.OriginType, binding.OriginRef, binding.ResolvedVersion, binding.LocalPath)
	}
	_ = w.Flush()
	return nil
}

func runModuleListRemote(cmd *cobra.Command, runtimeScope scope.Scope, runtimeOptions cliRuntimeOptions, registryAlias string) error {
	items, err := listRemoteCatalogModules(cmd, runtimeScope, runtimeOptions, registryAlias, "")
	if err != nil {
		return err
	}
	if len(items) == 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "No remote modules found in registry %q\n", strings.TrimSpace(registryAlias))
		return nil
	}

	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "MODULE\tLATEST\tDESCRIPTION")
	for _, item := range items {
		fmt.Fprintf(w, "%s\t%s\t%s\n", item.Name, item.LatestVersion, item.Description)
	}
	_ = w.Flush()
	return nil
}

func listRemoteCatalogModules(cmd *cobra.Command, runtimeScope scope.Scope, runtimeOptions cliRuntimeOptions, registryAlias, query string) ([]sourceregistry.CatalogModule, error) {
	entry, err := resolveRegistryEntry(runtimeOptions, registryAlias)
	if err != nil {
		return nil, err
	}
	catalog := sourceregistry.NewCatalog(runtimeScope)
	items, err := catalog.List(contextFromCommand(cmd), entry.IndexURL, query)
	if err != nil {
		return nil, xfmt.Errorf("query remote registry %q failed: %w", strings.TrimSpace(registryAlias), err)
	}
	return items, nil
}

func loadRemoteModuleInfo(cmd *cobra.Command, runtimeScope scope.Scope, runtimeOptions cliRuntimeOptions, registryAlias, moduleName string) (*sourceregistry.CatalogModule, error) {
	entry, err := resolveRegistryEntry(runtimeOptions, registryAlias)
	if err != nil {
		return nil, err
	}
	catalog := sourceregistry.NewCatalog(runtimeScope)
	item, err := catalog.Info(contextFromCommand(cmd), entry.IndexURL, moduleName)
	if err != nil {
		return nil, xfmt.Errorf("query remote module info failed (registry=%s module=%s): %w", strings.TrimSpace(registryAlias), strings.TrimSpace(moduleName), err)
	}
	return item, nil
}

func resolveRegistryEntry(runtimeOptions cliRuntimeOptions, registryAlias string) (sourceregistry.Entry, error) {
	alias := strings.TrimSpace(registryAlias)
	if alias == "" {
		alias = sourceregistry.DefaultRegistryAlias
	}
	entry, err := registryStoreFromRuntimeOptions(runtimeOptions).Resolve(alias)
	if err != nil {
		return sourceregistry.Entry{}, xfmt.Errorf("resolve registry alias %q failed: %w", alias, err)
	}
	return entry, nil
}

func contextFromCommand(cmd *cobra.Command) context.Context {
	if cmd == nil || cmd.Context() == nil {
		return context.Background()
	}
	return cmd.Context()
}

type moduleIndexView struct {
	ModuleName    string
	OriginType    string
	OriginRef     string
	Available     bool
	Version       sql.NullString
	LocalPath     sql.NullString
	InstallStatus sql.NullString
}

func queryModuleIndexViews(runtimeScope scope.Scope, moduleName string) ([]moduleIndexView, bool, error) {
	views := make([]moduleIndexView, 0)
	hasIndex := false

	err := runtimeScope.Transactor().Required(runtimeScope.Context(), func(txScope scope.Scope, _ scope.Transaction) error {
		if txScope == nil || txScope.Session() == nil || txScope.Session().DB == nil {
			return nil
		}
		db := txScope.Session().DB
		if !db.Migrator().HasTable(&metadata.IrModuleIndex{}) {
			return nil
		}
		hasIndex = true

		q := db.Table((metadata.IrModuleIndex{}).TableName() + " AS idx")
		if db.Migrator().HasTable(&meta.IrModule{}) {
			q = q.
				Select("idx.module_name, idx.origin_type, idx.origin_ref, idx.available, idx.version, idx.local_path, mod.status AS install_status").
				Joins("LEFT JOIN " + (&meta.IrModule{}).TableName() + " AS mod ON mod.name = idx.module_name")
		} else {
			q = q.Select("idx.module_name, idx.origin_type, idx.origin_ref, idx.available, idx.version, idx.local_path, '' AS install_status")
		}
		if strings.TrimSpace(moduleName) != "" {
			q = q.Where("idx.module_name = ?", strings.TrimSpace(moduleName))
		}
		if err := q.Order("idx.module_name ASC").Scan(&views).Error; err != nil {
			return xfmt.Errorf("query module index failed: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, false, err
	}
	return views, hasIndex, nil
}

func moduleIndexViewToPayload(view moduleIndexView) map[string]any {
	return map[string]any{
		"moduleName":    strings.TrimSpace(view.ModuleName),
		"originType":    strings.TrimSpace(view.OriginType),
		"originRef":     strings.TrimSpace(view.OriginRef),
		"available":     view.Available,
		"version":       nullStringValue(view.Version),
		"localPath":     nullStringValue(view.LocalPath),
		"installStatus": nullStringValue(view.InstallStatus),
	}
}

func nullStringValue(v sql.NullString) string {
	if !v.Valid {
		return ""
	}
	return strings.TrimSpace(v.String)
}

func newModuleFetchCmd(envGetter func() scope.Scope) *cobra.Command {
	return &cobra.Command{
		Use:   "fetch <input>",
		Short: "Fetch a module from source and bind it in modules.lock.json",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			coordinator, ctx, err := newCoordinatorForCommand(envGetter, cmd)
			if err != nil {
				return err
			}

			module, err := coordinator.Fetch(ctx, strings.TrimSpace(args[0]))
			if err != nil {
				return err
			}
			if module == nil {
				return xfmt.Errorf("source returned nil module")
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Fetched module %s@%s to %s\n", module.Name, module.Version, module.Path)
			return nil
		},
	}
}

func newModulePurgeCmd(envGetter func() scope.Scope) *cobra.Command {
	return &cobra.Command{
		Use:   "purge <module>",
		Short: "Purge a module source binding and local cache",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			env, err := requireCommandScope(envGetter)
			if err != nil {
				return err
			}

			coordinator, ctx, err := newCoordinatorForCommand(envGetter, cmd)
			if err != nil {
				return err
			}

			moduleName := strings.TrimSpace(args[0])
			if moduleName == "" {
				return xfmt.Errorf("module name is required")
			}
			if err := ensurePurgeModuleNotInstalled(env, moduleName); err != nil {
				return err
			}
			if err := coordinator.Purge(ctx, moduleName); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Purged module %s\n", moduleName)
			return nil
		},
	}
}

func ensurePurgeModuleNotInstalled(runtimeScope scope.Scope, moduleName string) error {
	moduleName = strings.TrimSpace(moduleName)
	if moduleName == "" {
		return xfmt.Errorf("module name is required")
	}

	var installed bool
	err := runtimeScope.Transactor().Required(runtimeScope.Context(), func(txScope scope.Scope, _ scope.Transaction) error {
		if txScope == nil || txScope.Session() == nil || txScope.Session().DB == nil {
			return nil
		}
		if !txScope.Session().DB.Migrator().HasTable(&meta.IrModule{}) {
			return nil
		}

		var count int64
		queryErr := txScope.Session().
			Model(&meta.IrModule{}).
			Where("name = ? AND status IN ?", moduleName, []meta.Status{meta.Installed, meta.ToUpgrade}).
			Count(&count).Error
		if queryErr != nil {
			return xfmt.Errorf("query module install state failed: %w", queryErr)
		}
		installed = count > 0
		return nil
	})
	if err != nil {
		return err
	}
	if installed {
		return xfmt.Errorf("module %s is installed; run 'choysum uninstall %s' before purge", moduleName, moduleName)
	}
	return nil
}

func newCoordinatorForCommand(scopeGetter func() scope.Scope, cmd *cobra.Command) (*internalorigin.Coordinator, context.Context, error) {
	runtimeScope, err := requireCommandScope(scopeGetter)
	if err != nil {
		return nil, nil, err
	}

	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	return internalorigin.NewCoordinator(runtimeScope), ctx, nil
}

func requireCommandScope(scopeGetter func() scope.Scope) (scope.Scope, error) {
	runtimeScope := scopeGetter()
	if runtimeScope == nil {
		return nil, xfmt.Errorf("scope is not initialized")
	}
	return runtimeScope, nil
}

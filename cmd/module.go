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
	"github.com/choysum-dev/choysum/pkg/config"
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
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search modules from local workspace or remote module catalog index",
		Args:  cobra.ExactArgs(1),
		Annotations: map[string]string{
			lightweightScopeAnnotation: "true",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			env, err := requireCommandScope(envGetter)
			if err != nil {
				return err
			}
			runtimeOptions, err := requireCliRuntimeOptionsForCommand("module", runtimeOptionsGetter)
			if err != nil {
				return err
			}
			if remote {
				return runModuleSearchRemote(cmd, env, runtimeOptions, args[0])
			}
			return runModuleSearchLocal(cmd, env, runtimeOptions, args[0])
		},
	}
	cmd.Flags().BoolVar(&remote, "remote", false, "query modules from remote module catalog index")
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

func runModuleSearchRemote(cmd *cobra.Command, runtimeScope scope.Scope, runtimeOptions cliRuntimeOptions, rawQuery string) error {
	query := strings.TrimSpace(rawQuery)
	if query == "" {
		return xfmt.Errorf("query is required")
	}

	items, err := listRemoteCatalogModules(cmd, runtimeScope, runtimeOptions, query)
	if err != nil {
		return err
	}
	if len(items) == 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "No remote modules matched query %q\n", rawQuery)
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
	var showAll bool
	var cliCompatVersion string
	cmd := &cobra.Command{
		Use:   "info <module|module@version>",
		Short: "Inspect source metadata for a module input",
		Args:  cobra.ExactArgs(1),
		Annotations: map[string]string{
			lightweightScopeAnnotation: "true",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			env, err := requireCommandScope(envGetter)
			if err != nil {
				return err
			}
			if remote {
				runtimeOptions, err := requireCliRuntimeOptionsForCommand("module", runtimeOptionsGetter)
				if err != nil {
					return err
				}
				return runModuleInfoRemote(cmd, env, runtimeOptions, args[0], cliCompatVersion, showAll)
			}
			return runModuleInfoLocal(cmd, envGetter, args[0])
		},
	}
	cmd.Flags().BoolVar(&remote, "remote", false, "query module info from remote module catalog index")
	cmd.Flags().BoolVar(&showAll, "all", false, "show all remote versions without default compatibility filtering")
	cmd.Flags().StringVar(&cliCompatVersion, "cli-compat-version", "", "override CLI compatibility version for module compatibility checks")
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

func runModuleInfoRemote(cmd *cobra.Command, runtimeScope scope.Scope, runtimeOptions cliRuntimeOptions, input string, cliCompatVersion string, showAll bool) error {
	raw := strings.TrimSpace(input)
	if raw == "" {
		return xfmt.Errorf("module input is required")
	}

	moduleName := raw
	if parsed, err := internalorigin.ParseInput(raw); err == nil {
		if parsed.Kind == internalorigin.InputKindRegistry {
			moduleName = parsed.ModuleName
			version := strings.TrimSpace(parsed.Version)
			if version != "" && !strings.EqualFold(version, "latest") {
				resolvedCompat, compatErr := resolveCLICompatVersionForCommand(cmd, cliCompatVersion)
				if compatErr != nil {
					return compatErr
				}
				if !showAll {
					if strings.TrimSpace(resolvedCompat.Version) == "" {
						return xfmt.Errorf("ERR_CLI_COMPAT_VERSION_UNRESOLVED: Cannot resolve a CLI compatibility version in development mode. Provide '--cli-compat-version' or set 'CHOYSUM_CLI_COMPAT_VERSION'.")
					}
					catalogItem, catalogErr := loadRemoteModuleInfo(cmd, runtimeScope, runtimeOptions, moduleName)
					if catalogErr != nil {
						return catalogErr
					}
					compatibleVersions, compatibilityErr := compatibleCatalogVersions(catalogItem, resolvedCompat.Version)
					if compatibilityErr != nil {
						return compatibilityErr
					}
					if !containsCatalogVersion(compatibleVersions, version) {
						return xfmt.Errorf("ERR_MODULE_NO_COMPATIBLE_VERSION: No compatible version found for module '%s' with CLI version '%s'.", strings.TrimSpace(moduleName), strings.TrimSpace(resolvedCompat.Version))
					}
				} else if strings.TrimSpace(resolvedCompat.Version) == "" {
					printCLIWarning(cliCompatFilterSkippedWarning())
				}

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
		}
		moduleName = strings.TrimSpace(parsed.ModuleName)
	} else if strings.Contains(raw, "/") {
		return err
	}

	resolvedCompat, err := resolveCLICompatVersionForCommand(cmd, cliCompatVersion)
	if err != nil {
		return err
	}

	item, err := loadRemoteModuleInfo(cmd, runtimeScope, runtimeOptions, moduleName)
	if err != nil {
		return err
	}

	itemForOutput := item
	if showAll {
		if strings.TrimSpace(resolvedCompat.Version) == "" {
			printCLIWarning(cliCompatFilterSkippedWarning())
		}
	} else {
		if strings.TrimSpace(resolvedCompat.Version) == "" {
			return xfmt.Errorf("ERR_CLI_COMPAT_VERSION_UNRESOLVED: Cannot resolve a CLI compatibility version in development mode. Provide '--cli-compat-version' or set 'CHOYSUM_CLI_COMPAT_VERSION'.")
		}
		filtered, err := filterCatalogModuleByCompatibility(item, resolvedCompat.Version)
		if err != nil {
			return err
		}
		itemForOutput = filtered
	}

	payload, err := json.MarshalIndent(itemForOutput, "", "  ")
	if err != nil {
		return xfmt.Errorf("marshal remote module info: %w", err)
	}
	fmt.Fprintln(cmd.OutOrStdout(), string(payload))
	return nil
}

func newModuleListCmd(envGetter func() scope.Scope, runtimeOptionsGetter func() cliRuntimeOptions) *cobra.Command {
	var remote bool
	var showAll bool
	var cliCompatVersion string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List modules in source lock bindings or remote module catalog index",
		Args:  cobra.NoArgs,
		Annotations: map[string]string{
			lightweightScopeAnnotation: "true",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			env, err := requireCommandScope(envGetter)
			if err != nil {
				return err
			}
			runtimeOptions, err := requireCliRuntimeOptionsForCommand("module", runtimeOptionsGetter)
			if err != nil {
				return err
			}
			if remote {
				return runModuleListRemote(cmd, env, runtimeOptions, cliCompatVersion, showAll)
			}
			return runModuleListLocal(cmd, env, runtimeOptions)
		},
	}
	cmd.Flags().BoolVar(&remote, "remote", false, "list modules from remote module catalog index")
	cmd.Flags().BoolVar(&showAll, "all", false, "show all remote versions without default compatibility filtering")
	cmd.Flags().StringVar(&cliCompatVersion, "cli-compat-version", "", "override CLI compatibility version for module compatibility checks")
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

func runModuleListRemote(cmd *cobra.Command, runtimeScope scope.Scope, runtimeOptions cliRuntimeOptions, cliCompatVersion string, showAll bool) error {
	resolvedCompat, err := resolveCLICompatVersionForCommand(cmd, cliCompatVersion)
	if err != nil {
		return err
	}

	items, err := listRemoteCatalogModules(cmd, runtimeScope, runtimeOptions, "")
	if err != nil {
		return err
	}
	if !showAll {
		if strings.TrimSpace(resolvedCompat.Version) == "" {
			return xfmt.Errorf("ERR_CLI_COMPAT_VERSION_UNRESOLVED: Cannot resolve a CLI compatibility version in development mode. Provide '--cli-compat-version' or set 'CHOYSUM_CLI_COMPAT_VERSION'.")
		}
		filteredItems := make([]sourceregistry.CatalogModule, 0, len(items))
		for i := range items {
			filtered, err := filterCatalogModuleByCompatibility(&items[i], resolvedCompat.Version)
			if err != nil {
				if strings.Contains(err.Error(), "ERR_MODULE_NO_COMPATIBLE_VERSION") {
					continue
				}
				return err
			}
			filteredItems = append(filteredItems, *filtered)
		}
		items = filteredItems
	} else if strings.TrimSpace(resolvedCompat.Version) == "" {
		printCLIWarning(cliCompatFilterSkippedWarning())
	}

	if len(items) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No remote modules found.")
		return nil
	}

	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "MODULE\tLATEST\tCLI_RANGE\tDESCRIPTION")
	for _, item := range items {
		latestCLIRange, _ := item.LatestCLIRange()
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", item.Name, item.LatestVersion, latestCLIRange, item.Description)
	}
	_ = w.Flush()
	return nil
}

func listRemoteCatalogModules(cmd *cobra.Command, runtimeScope scope.Scope, runtimeOptions cliRuntimeOptions, query string) ([]sourceregistry.CatalogModule, error) {
	indexURL, err := resolveModuleCatalogIndexURL(runtimeOptions)
	if err != nil {
		return nil, err
	}
	catalog := sourceregistry.NewCatalog(runtimeScope)
	items, err := catalog.List(contextFromCommand(cmd), indexURL, query)
	if err != nil {
		return nil, xfmt.Errorf("query remote module catalog index %q failed: %w", indexURL, err)
	}
	return items, nil
}

func loadRemoteModuleInfo(cmd *cobra.Command, runtimeScope scope.Scope, runtimeOptions cliRuntimeOptions, moduleName string) (*sourceregistry.CatalogModule, error) {
	indexURL, err := resolveModuleCatalogIndexURL(runtimeOptions)
	if err != nil {
		return nil, err
	}
	catalog := sourceregistry.NewCatalog(runtimeScope)
	item, err := catalog.Info(contextFromCommand(cmd), indexURL, moduleName)
	if err != nil {
		return nil, xfmt.Errorf("query remote module info failed (module=%s): %w", strings.TrimSpace(moduleName), err)
	}
	return item, nil
}

func resolveModuleCatalogIndexURL(runtimeOptions cliRuntimeOptions) (string, error) {
	indexURL := strings.TrimSpace(runtimeOptions.moduleCatalogIndexURL)
	if indexURL == "" {
		indexURL = config.DefaultModuleCatalogIndexURL
	}
	if err := config.ValidateModuleCatalogIndexURL(indexURL); err != nil {
		return "", err
	}
	return indexURL, nil
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
		Use:   "fetch <module|module@version>",
		Short: "Fetch a module from source and bind it in modules.lock.json",
		Args:  cobra.ExactArgs(1),
		Annotations: map[string]string{
			lightweightScopeAnnotation: "true",
		},
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

// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cmd

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"text/tabwriter"

	sourceregistry "github.com/choysum-dev/choysum/internal/module/origin/registry"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/spf13/cobra"
	xfmt "golang.org/x/exp/errors/fmt"
)

func newRegistryCmd(envGetter func() scope.Scope, runtimeOptionsGetter func() cliRuntimeOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "registry",
		Short: "Manage module source registries",
	}

	cmd.AddCommand(
		newRegistryAddCmd(envGetter, runtimeOptionsGetter),
		newRegistryListCmd(envGetter, runtimeOptionsGetter),
		newRegistryRemoveCmd(envGetter, runtimeOptionsGetter),
		newRegistryLoginCmd(envGetter, runtimeOptionsGetter),
	)

	return cmd
}

func newRegistryAddCmd(envGetter func() scope.Scope, runtimeOptionsGetter func() cliRuntimeOptions) *cobra.Command {
	var authRef string
	cmd := &cobra.Command{
		Use:   "add <alias> <index-url>",
		Short: "Add or update a registry alias",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			env := envGetter()
			if env == nil {
				return xfmt.Errorf("scope is not initialized")
			}
			runtimeOptions, err := requireCliRuntimeOptions(runtimeOptionsGetter)
			if err != nil {
				return xfmt.Errorf("registry: invalid runtime options: %w", err)
			}
			alias := strings.TrimSpace(args[0])
			indexURL := strings.TrimSpace(args[1])
			if err := validateRegistryAlias(alias); err != nil {
				return err
			}
			if err := validateRegistryIndexURL(indexURL); err != nil {
				return err
			}

			store := registryStoreFromRuntimeOptions(runtimeOptions)
			cfg, err := store.Load()
			if err != nil {
				return xfmt.Errorf("load registries config: %w", err)
			}
			if cfg.Registries == nil {
				cfg.Registries = map[string]sourceregistry.Entry{}
			}
			cfg.Registries[alias] = sourceregistry.Entry{IndexURL: indexURL, AuthRef: strings.TrimSpace(authRef)}
			if err := store.Save(cfg); err != nil {
				return xfmt.Errorf("save registries config: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Registry %q -> %s\n", alias, indexURL)
			return nil
		},
	}
	cmd.Flags().StringVar(&authRef, "auth-ref", "", "authentication reference for the registry")
	return cmd
}

func newRegistryListCmd(envGetter func() scope.Scope, runtimeOptionsGetter func() cliRuntimeOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List configured registry aliases",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			env := envGetter()
			if env == nil {
				return xfmt.Errorf("scope is not initialized")
			}
			runtimeOptions, err := requireCliRuntimeOptions(runtimeOptionsGetter)
			if err != nil {
				return xfmt.Errorf("registry: invalid runtime options: %w", err)
			}
			store := registryStoreFromRuntimeOptions(runtimeOptions)
			cfg, err := store.Load()
			if err != nil {
				return xfmt.Errorf("load registries config: %w", err)
			}

			aliases := make([]string, 0, len(cfg.Registries))
			for alias := range cfg.Registries {
				aliases = append(aliases, alias)
			}
			sort.Strings(aliases)
			if len(aliases) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No registries configured.")
				return nil
			}

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "ALIAS\tINDEX_URL\tAUTH_REF")
			for _, alias := range aliases {
				entry := cfg.Registries[alias]
				fmt.Fprintf(w, "%s\t%s\t%s\n", alias, entry.IndexURL, entry.AuthRef)
			}
			_ = w.Flush()
			return nil
		},
	}
}

func newRegistryRemoveCmd(envGetter func() scope.Scope, runtimeOptionsGetter func() cliRuntimeOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "remove <alias>",
		Short: "Remove a registry alias",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			env := envGetter()
			if env == nil {
				return xfmt.Errorf("scope is not initialized")
			}
			runtimeOptions, err := requireCliRuntimeOptions(runtimeOptionsGetter)
			if err != nil {
				return xfmt.Errorf("registry: invalid runtime options: %w", err)
			}
			alias := strings.TrimSpace(args[0])
			if err := validateRegistryAlias(alias); err != nil {
				return err
			}
			if alias == sourceregistry.DefaultRegistryAlias {
				return xfmt.Errorf("cannot remove default registry alias %q", sourceregistry.DefaultRegistryAlias)
			}

			store := registryStoreFromRuntimeOptions(runtimeOptions)
			cfg, err := store.Load()
			if err != nil {
				return xfmt.Errorf("load registries config: %w", err)
			}
			if _, ok := cfg.Registries[alias]; !ok {
				return xfmt.Errorf("registry alias %q not found", alias)
			}
			delete(cfg.Registries, alias)
			if err := store.Save(cfg); err != nil {
				return xfmt.Errorf("save registries config: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Registry %q removed\n", alias)
			return nil
		},
	}
}

func newRegistryLoginCmd(envGetter func() scope.Scope, runtimeOptionsGetter func() cliRuntimeOptions) *cobra.Command {
	var authRef string
	cmd := &cobra.Command{
		Use:   "login <alias>",
		Short: "Set auth reference for a registry alias",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			env := envGetter()
			if env == nil {
				return xfmt.Errorf("scope is not initialized")
			}
			runtimeOptions, err := requireCliRuntimeOptions(runtimeOptionsGetter)
			if err != nil {
				return xfmt.Errorf("registry: invalid runtime options: %w", err)
			}
			alias := strings.TrimSpace(args[0])
			if err := validateRegistryAlias(alias); err != nil {
				return err
			}
			authRef = strings.TrimSpace(authRef)
			if authRef == "" {
				return xfmt.Errorf("--auth-ref is required")
			}

			store := registryStoreFromRuntimeOptions(runtimeOptions)
			cfg, err := store.Load()
			if err != nil {
				return xfmt.Errorf("load registries config: %w", err)
			}
			entry, ok := cfg.Registries[alias]
			if !ok {
				return xfmt.Errorf("registry alias %q not found", alias)
			}
			entry.AuthRef = authRef
			cfg.Registries[alias] = entry
			if err := store.Save(cfg); err != nil {
				return xfmt.Errorf("save registries config: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Registry %q auth updated\n", alias)
			return nil
		},
	}
	cmd.Flags().StringVar(&authRef, "auth-ref", "", "authentication reference for the registry")
	return cmd
}

func validateRegistryAlias(alias string) error {
	if alias == "" {
		return xfmt.Errorf("registry alias is required")
	}
	if strings.Contains(alias, "/") || strings.Contains(alias, "\\\\") {
		return xfmt.Errorf("invalid registry alias %q", alias)
	}
	return nil
}

func validateRegistryIndexURL(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil {
		return xfmt.Errorf("invalid registry index url %q: %w", raw, err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return xfmt.Errorf("invalid registry index url %q: only http/https are supported", raw)
	}
	if parsed.Host == "" {
		return xfmt.Errorf("invalid registry index url %q: host is required", raw)
	}
	if !strings.HasSuffix(strings.ToLower(strings.TrimSpace(parsed.Path)), "/index.json") {
		return xfmt.Errorf("invalid registry index url %q: must point to an index.json resource", raw)
	}
	return nil
}

func registryStoreFromRuntimeOptions(runtimeOptions cliRuntimeOptions) *sourceregistry.Store {
	return sourceregistry.NewStore(sourceregistry.WithDefaultChoysumPath(runtimeOptions.defaultChoysumPath))
}

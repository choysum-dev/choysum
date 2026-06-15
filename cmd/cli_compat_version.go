// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package cmd

import (
	"context"
	"os"
	"strings"

	msemver "github.com/Masterminds/semver/v3"
	internalorigin "github.com/choysum-dev/choysum/internal/module/origin"
	sourceregistry "github.com/choysum-dev/choysum/internal/module/origin/registry"
	"github.com/choysum-dev/choysum/pkg/scope"
	"github.com/spf13/cobra"
	xfmt "golang.org/x/exp/errors/fmt"
	modsemver "golang.org/x/mod/semver"
)

const cliCompatVersionEnv = "CHOYSUM_CLI_COMPAT_VERSION"

type resolvedCLICompatVersion struct {
	Version string
	Source  string
}

func normalizeCLICompatVersion(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}
	candidate := raw
	if !strings.HasPrefix(candidate, "v") {
		candidate = "v" + candidate
	}
	if !modsemver.IsValid(candidate) {
		return "", false
	}
	return candidate, true
}

func parseCLICompatVersion(raw string) (string, error) {
	normalized, ok := normalizeCLICompatVersion(raw)
	if !ok {
		return "", xfmt.Errorf("ERR_CLI_COMPAT_VERSION_INVALID: Invalid CLI compatibility version %q. Expected SemVer like '1.7.0' or 'v1.7.0'.", strings.TrimSpace(raw))
	}
	return normalized, nil
}

func resolveCLICompatVersion(flagValue, runtimeVersion string) (resolvedCLICompatVersion, error) {
	if strings.TrimSpace(flagValue) != "" {
		normalized, err := parseCLICompatVersion(flagValue)
		if err != nil {
			return resolvedCLICompatVersion{}, err
		}
		return resolvedCLICompatVersion{Version: normalized, Source: "flag"}, nil
	}

	if envValue := strings.TrimSpace(os.Getenv(cliCompatVersionEnv)); envValue != "" {
		normalized, err := parseCLICompatVersion(envValue)
		if err != nil {
			return resolvedCLICompatVersion{}, err
		}
		return resolvedCLICompatVersion{Version: normalized, Source: "env"}, nil
	}

	if normalized, ok := normalizeCLICompatVersion(runtimeVersion); ok {
		return resolvedCLICompatVersion{Version: normalized, Source: "runtime"}, nil
	}

	return resolvedCLICompatVersion{}, nil
}

func resolveCLICompatVersionForCommand(cmd *cobra.Command, flagValue string) (resolvedCLICompatVersion, error) {
	runtimeVersion := ""
	if cmd != nil && cmd.Root() != nil {
		runtimeVersion = strings.TrimSpace(cmd.Root().Version)
	}
	return resolveCLICompatVersion(flagValue, runtimeVersion)
}

func cliVersionForConstraint(cliVersion string) (*msemver.Version, error) {
	normalized, ok := normalizeCLICompatVersion(cliVersion)
	if !ok {
		return nil, xfmt.Errorf("ERR_CLI_COMPAT_VERSION_INVALID: Invalid CLI compatibility version %q. Expected SemVer like '1.7.0' or 'v1.7.0'.", strings.TrimSpace(cliVersion))
	}
	version, err := msemver.NewVersion(strings.TrimPrefix(normalized, "v"))
	if err != nil {
		return nil, xfmt.Errorf("ERR_CLI_COMPAT_VERSION_INVALID: Invalid CLI compatibility version %q. Expected SemVer like '1.7.0' or 'v1.7.0'.", strings.TrimSpace(cliVersion))
	}
	return version, nil
}

func catalogCandidateVersions(item *sourceregistry.CatalogModule) []string {
	if item == nil {
		return nil
	}
	if len(item.Versions) > 0 {
		result := make([]string, 0, len(item.Versions))
		for _, version := range item.Versions {
			version = strings.TrimSpace(version)
			if version == "" {
				continue
			}
			result = append(result, version)
		}
		return result
	}
	if latest := strings.TrimSpace(item.LatestVersion); latest != "" {
		return []string{latest}
	}
	return nil
}

func normalizeCatalogModuleVersion(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}
	candidate := raw
	if !strings.HasPrefix(candidate, "v") {
		candidate = "v" + candidate
	}
	if !modsemver.IsValid(candidate) {
		return "", false
	}
	return candidate, true
}

func containsCatalogVersion(versions []string, target string) bool {
	target = strings.TrimSpace(target)
	if target == "" {
		return false
	}
	targetNormalized, targetIsSemVer := normalizeCatalogModuleVersion(target)
	for _, version := range versions {
		version = strings.TrimSpace(version)
		if version == "" {
			continue
		}
		if version == target {
			return true
		}
		if !targetIsSemVer {
			continue
		}
		if normalized, ok := normalizeCatalogModuleVersion(version); ok && normalized == targetNormalized {
			return true
		}
	}
	return false
}

func compatibleCatalogVersions(item *sourceregistry.CatalogModule, cliVersion string) ([]string, error) {
	if item == nil {
		return nil, xfmt.Errorf("remote module is nil")
	}
	moduleName := strings.TrimSpace(item.Name)
	if moduleName == "" {
		moduleName = "<unknown>"
	}

	version, err := cliVersionForConstraint(cliVersion)
	if err != nil {
		return nil, err
	}

	compatible := make([]string, 0)
	for _, moduleVersion := range catalogCandidateVersions(item) {
		cliRange, ok := item.CLIRangeForVersion(moduleVersion)
		if !ok || strings.TrimSpace(cliRange) == "" {
			return nil, xfmt.Errorf("ERR_MODULE_CLI_RANGE_MISSING: Module '%s' is missing required field 'choysum.cli'.", moduleName)
		}
		constraint, err := msemver.NewConstraint(strings.TrimSpace(cliRange))
		if err != nil {
			return nil, xfmt.Errorf("ERR_MODULE_CLI_RANGE_INVALID: Module '%s' has invalid choysum.cli range '%s'.", moduleName, strings.TrimSpace(cliRange))
		}
		if constraint.Check(version) {
			compatible = append(compatible, moduleVersion)
		}
	}
	if len(compatible) == 0 {
		return nil, xfmt.Errorf("ERR_MODULE_NO_COMPATIBLE_VERSION: No compatible version found for module '%s' with CLI version '%s'.", moduleName, strings.TrimSpace(cliVersion))
	}
	return compatible, nil
}

func selectLatestCompatibleCatalogVersion(item *sourceregistry.CatalogModule, cliVersion string) (string, error) {
	versions, err := compatibleCatalogVersions(item, cliVersion)
	if err != nil {
		return "", err
	}
	return versions[len(versions)-1], nil
}

func filterCatalogModuleByCompatibility(item *sourceregistry.CatalogModule, cliVersion string) (*sourceregistry.CatalogModule, error) {
	if item == nil {
		return nil, xfmt.Errorf("remote module is nil")
	}
	versions, err := compatibleCatalogVersions(item, cliVersion)
	if err != nil {
		return nil, err
	}
	filtered := *item
	filtered.Versions = append([]string{}, versions...)
	filtered.LatestVersion = versions[len(versions)-1]
	if len(filtered.VersionCLIRanges) > 0 {
		ranges := make(map[string]string, len(versions))
		for _, version := range versions {
			if cliRange, ok := item.CLIRangeForVersion(version); ok && strings.TrimSpace(cliRange) != "" {
				ranges[version] = strings.TrimSpace(cliRange)
			}
		}
		if len(ranges) == 0 {
			filtered.VersionCLIRanges = nil
		} else {
			filtered.VersionCLIRanges = ranges
		}
	}
	if filtered.Source != nil {
		if strings.TrimSpace(filtered.LatestVersion) == strings.TrimSpace(item.LatestVersion) {
			source := *filtered.Source
			source.Version = filtered.LatestVersion
			filtered.Source = &source
		} else {
			filtered.Source = nil
		}
	}
	return &filtered, nil
}

func resolveCompatibleRegistryLatestVersion(ctx context.Context, runtimeScope scope.Scope, runtimeOptions cliRuntimeOptions, moduleName, cliVersion string) (string, error) {
	indexURL, err := resolveModuleCatalogIndexURL(runtimeOptions)
	if err != nil {
		return "", err
	}
	catalog := sourceregistry.NewCatalog(runtimeScope)
	item, err := catalog.Info(ctx, indexURL, strings.TrimSpace(moduleName))
	if err != nil {
		return "", xfmt.Errorf("query remote module info failed (module=%s): %w", strings.TrimSpace(moduleName), err)
	}
	version, err := selectLatestCompatibleCatalogVersion(item, cliVersion)
	if err != nil {
		return "", err
	}
	return version, nil
}

func resolveRegistryBackedUpgradeInput(ctx context.Context, runtimeScope scope.Scope, runtimeOptions cliRuntimeOptions, moduleName, cliVersion string) (string, bool, error) {
	moduleName = strings.TrimSpace(moduleName)
	if moduleName == "" {
		return "", false, xfmt.Errorf("module name is empty")
	}
	if err := runtimeOptions.Validate(); err != nil {
		return "", false, err
	}

	registryBacked, err := hasRegistryOriginBinding(runtimeScope, runtimeOptions, moduleName)
	if err != nil {
		return "", false, err
	}
	if !registryBacked {
		return moduleName, false, nil
	}

	compatibleVersion, err := resolveCompatibleRegistryLatestVersion(ctx, runtimeScope, runtimeOptions, moduleName, cliVersion)
	if err != nil {
		return "", true, err
	}
	return moduleName + "@" + compatibleVersion, true, nil
}

func hasRegistryOriginBinding(runtimeScope scope.Scope, runtimeOptions cliRuntimeOptions, moduleName string) (bool, error) {
	moduleName = strings.TrimSpace(moduleName)
	if moduleName == "" {
		return false, xfmt.Errorf("module name is empty")
	}
	if err := runtimeOptions.Validate(); err != nil {
		return false, err
	}

	workspaceRoot := internalorigin.WorkspaceRoot(runtimeScope)
	lockStore := internalorigin.NewLockStore(internalorigin.WithLockStoreDefaultChoysumPath(runtimeOptions.defaultChoysumPath))
	binding, ok, err := lockStore.LookupBinding(workspaceRoot, moduleName)
	if err != nil {
		return false, xfmt.Errorf("lookup module origin binding failed: %w", err)
	}
	return ok && strings.TrimSpace(binding.OriginType) == internalorigin.OriginTypeRegistry, nil
}

func cliCompatFilterSkippedWarning() string {
	return "WARN_CLI_COMPAT_FILTER_SKIPPED: Compatibility filtering is skipped in '--all' mode because no CLI compatibility version is available."
}

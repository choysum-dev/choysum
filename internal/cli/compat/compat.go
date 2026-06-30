// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package compat

import (
	"context"
	"strings"

	msemver "github.com/Masterminds/semver/v3"
	internalorigin "github.com/choysum-dev/choysum/internal/module/origin"
	sourceregistry "github.com/choysum-dev/choysum/internal/module/origin/registry"
	"github.com/choysum-dev/choysum/pkg/scope"
	xfmt "golang.org/x/exp/errors/fmt"
	modsemver "golang.org/x/mod/semver"
)

const CLICompatVersionEnv = "CHOYSUM_CLI_COMPAT_VERSION"

type ResolvedCLICompatVersion struct {
	Version string
	Source  string
}

func NormalizeCLICompatVersion(raw string) (string, bool) {
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

func ParseCLICompatVersion(raw string) (string, error) {
	normalized, ok := NormalizeCLICompatVersion(raw)
	if !ok {
		return "", xfmt.Errorf("ERR_CLI_COMPAT_VERSION_INVALID: Invalid CLI compatibility version %q. Expected SemVer like '1.7.0' or 'v1.7.0'.", strings.TrimSpace(raw))
	}
	return normalized, nil
}

func ResolveCLICompatVersion(flagValue, runtimeVersion, envValue string) (ResolvedCLICompatVersion, error) {
	if strings.TrimSpace(flagValue) != "" {
		normalized, err := ParseCLICompatVersion(flagValue)
		if err != nil {
			return ResolvedCLICompatVersion{}, err
		}
		return ResolvedCLICompatVersion{Version: normalized, Source: "flag"}, nil
	}

	if strings.TrimSpace(envValue) != "" {
		normalized, err := ParseCLICompatVersion(envValue)
		if err != nil {
			return ResolvedCLICompatVersion{}, err
		}
		return ResolvedCLICompatVersion{Version: normalized, Source: "env"}, nil
	}

	if normalized, ok := NormalizeCLICompatVersion(runtimeVersion); ok {
		return ResolvedCLICompatVersion{Version: normalized, Source: "runtime"}, nil
	}

	return ResolvedCLICompatVersion{}, nil
}

func CLIVersionForConstraint(cliVersion string) (*msemver.Version, error) {
	normalized, ok := NormalizeCLICompatVersion(cliVersion)
	if !ok {
		return nil, xfmt.Errorf("ERR_CLI_COMPAT_VERSION_INVALID: Invalid CLI compatibility version %q. Expected SemVer like '1.7.0' or 'v1.7.0'.", strings.TrimSpace(cliVersion))
	}
	version, err := msemver.NewVersion(strings.TrimPrefix(normalized, "v"))
	if err != nil {
		return nil, xfmt.Errorf("ERR_CLI_COMPAT_VERSION_INVALID: Invalid CLI compatibility version %q. Expected SemVer like '1.7.0' or 'v1.7.0'.", strings.TrimSpace(cliVersion))
	}
	return version, nil
}

func CatalogCandidateVersions(item *sourceregistry.CatalogModule) []string {
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

func NormalizeCatalogModuleVersion(raw string) (string, bool) {
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

func ContainsCatalogVersion(versions []string, target string) bool {
	target = strings.TrimSpace(target)
	if target == "" {
		return false
	}
	targetNormalized, targetIsSemVer := NormalizeCatalogModuleVersion(target)
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
		if normalized, ok := NormalizeCatalogModuleVersion(version); ok && normalized == targetNormalized {
			return true
		}
	}
	return false
}

func CompatibleCatalogVersions(item *sourceregistry.CatalogModule, cliVersion string) ([]string, error) {
	if item == nil {
		return nil, xfmt.Errorf("remote module is nil")
	}
	moduleName := strings.TrimSpace(item.Name)
	if moduleName == "" {
		moduleName = "<unknown>"
	}

	version, err := CLIVersionForConstraint(cliVersion)
	if err != nil {
		return nil, err
	}

	compatible := make([]string, 0)
	for _, moduleVersion := range CatalogCandidateVersions(item) {
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

func SelectLatestCompatibleCatalogVersion(item *sourceregistry.CatalogModule, cliVersion string) (string, error) {
	versions, err := CompatibleCatalogVersions(item, cliVersion)
	if err != nil {
		return "", err
	}
	latest := latestCatalogVersion(versions)
	if latest == "" {
		return "", xfmt.Errorf("ERR_MODULE_NO_COMPATIBLE_VERSION: No compatible version found for module '%s' with CLI version '%s'.", strings.TrimSpace(item.Name), strings.TrimSpace(cliVersion))
	}
	return latest, nil
}

func FilterCatalogModuleByCompatibility(item *sourceregistry.CatalogModule, cliVersion string) (*sourceregistry.CatalogModule, error) {
	if item == nil {
		return nil, xfmt.Errorf("remote module is nil")
	}
	versions, err := CompatibleCatalogVersions(item, cliVersion)
	if err != nil {
		return nil, err
	}
	filtered := *item
	filtered.Versions = append([]string{}, versions...)
	filtered.LatestVersion = latestCatalogVersion(versions)
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

func latestCatalogVersion(versions []string) string {
	latest := ""
	latestNormalized := ""

	for _, version := range versions {
		version = strings.TrimSpace(version)
		if version == "" {
			continue
		}

		normalized, isSemVer := NormalizeCatalogModuleVersion(version)
		if !isSemVer {
			if latestNormalized == "" {
				latest = version
			}
			continue
		}

		if latestNormalized == "" || modsemver.Compare(normalized, latestNormalized) > 0 {
			latest = version
			latestNormalized = normalized
		}
	}

	return latest
}

func ResolveCompatibleRegistryLatestVersion(ctx context.Context, runtimeScope scope.Scope, indexURL, moduleName, cliVersion string) (string, error) {
	catalog := sourceregistry.NewCatalog(runtimeScope)
	item, err := catalog.Info(ctx, strings.TrimSpace(indexURL), strings.TrimSpace(moduleName))
	if err != nil {
		return "", xfmt.Errorf("query remote module info failed (module=%s): %w", strings.TrimSpace(moduleName), err)
	}
	version, err := SelectLatestCompatibleCatalogVersion(item, cliVersion)
	if err != nil {
		return "", err
	}
	return version, nil
}

func HasRegistryOriginBinding(runtimeScope scope.Scope, defaultChoysumPath, moduleName string) (bool, error) {
	moduleName = strings.TrimSpace(moduleName)
	if moduleName == "" {
		return false, xfmt.Errorf("module name is empty")
	}
	defaultChoysumPath = strings.TrimSpace(defaultChoysumPath)
	if defaultChoysumPath == "" {
		return false, xfmt.Errorf("cli runtime options: defaultChoysumPath is required")
	}

	workspaceRoot := internalorigin.WorkspaceRoot(runtimeScope)
	lockStore := internalorigin.NewLockStore(internalorigin.WithLockStoreDefaultChoysumPath(defaultChoysumPath))
	binding, ok, err := lockStore.LookupBinding(workspaceRoot, moduleName)
	if err != nil {
		return false, xfmt.Errorf("lookup module origin binding failed: %w", err)
	}
	return ok && strings.TrimSpace(binding.OriginType) == internalorigin.OriginTypeRegistry, nil
}

func CompatFilterSkippedWarning() string {
	return "WARN_CLI_COMPAT_FILTER_SKIPPED: Compatibility filtering is skipped in '--all' mode because no CLI compatibility version is available."
}

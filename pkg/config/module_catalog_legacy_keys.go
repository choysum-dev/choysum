// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package config

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/viper"
	xfmt "golang.org/x/exp/errors/fmt"
)

func rejectLegacyModuleCatalogConfigKeys(v *viper.Viper) error {
	if v == nil {
		return nil
	}

	invalid := make([]string, 0, 6)
	appendLegacy := func(key string) {
		key = strings.TrimSpace(key)
		if key == "" {
			return
		}
		invalid = append(invalid, key+" (use module_catalog_index_url)")
	}

	if v.InConfig("registry_index_url") {
		appendLegacy("registry_index_url")
	}

	if v.InConfig("registries") {
		appendLegacy("registries")

		registries := v.GetStringMap("registries")
		for alias, raw := range registries {
			for _, key := range legacyRegistryEntryKeys(raw) {
				appendLegacy("registries." + strings.TrimSpace(alias) + "." + key)
			}
		}
	}

	if len(invalid) == 0 {
		return nil
	}

	sort.Strings(invalid)
	return xfmt.Errorf("legacy module catalog config keys are no longer supported: %s", strings.Join(invalid, ", "))
}

func legacyRegistryEntryKeys(raw any) []string {
	switch typed := raw.(type) {
	case map[string]any:
		return collectLegacyRegistryEntryKeys(typed)
	case map[any]any:
		normalized := make(map[string]any, len(typed))
		for key, value := range typed {
			normalized[strings.TrimSpace(fmt.Sprintf("%v", key))] = value
		}
		return collectLegacyRegistryEntryKeys(normalized)
	default:
		return nil
	}
}

func collectLegacyRegistryEntryKeys(entry map[string]any) []string {
	legacy := make([]string, 0, 2)
	hasURL := false
	hasIndexURL := false
	hasIndexSnakeURL := false
	for key := range entry {
		switch strings.ToLower(strings.TrimSpace(key)) {
		case "url":
			hasURL = true
		case "indexurl":
			hasIndexURL = true
		case "index_url":
			hasIndexSnakeURL = true
		}
	}
	if hasURL {
		legacy = append(legacy, "url")
	}
	if hasIndexURL {
		legacy = append(legacy, "indexURL")
	}
	if hasIndexSnakeURL {
		legacy = append(legacy, "index_url")
	}
	sort.Strings(legacy)
	return legacy
}

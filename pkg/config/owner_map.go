// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package config

import (
	"reflect"
	"strings"
	"sync"

	xfmt "golang.org/x/exp/errors/fmt"
)

// ConfigRootOwner freezes root-field ownership by domain and options owner package.
type ConfigRootOwner struct {
	Domain      string
	PackagePath string
	OptionsType string
}

var configRootOwnerMap = map[string]ConfigRootOwner{
	"modules_path": {
		Domain:      "module",
		PackagePath: "internal/module/config",
		OptionsType: "ModulesPathConfig",
	},
	"dist_path": {
		Domain:      "server",
		PackagePath: "internal/server/serverconfig",
		OptionsType: "DistPathConfig",
	},
	"npm_path": {
		Domain:      "cli",
		PackagePath: "cmd",
		OptionsType: "cliRuntimeOptions",
	},
	"npm_registry_url": {
		Domain:      "module",
		PackagePath: "internal/module/origin/registry",
		OptionsType: "runtimeOptions",
	},
	"module_catalog_index_url": {
		Domain:      "module",
		PackagePath: "internal/module/origin/registry",
		OptionsType: "runtimeOptions",
	},
	"default_choysum_path": {
		Domain:      "cli",
		PackagePath: "cmd",
		OptionsType: "cliRuntimeOptions",
	},
	"tmp_path": {
		Domain:      "cli",
		PackagePath: "cmd",
		OptionsType: "cliRuntimeOptions",
	},
	"log": {
		Domain:      "logging",
		PackagePath: "pkg/logger",
		OptionsType: "LogConfig",
	},
	"db": {
		Domain:      "meta",
		PackagePath: "pkg/meta",
		OptionsType: "DbConfig",
	},
	"compile": {
		Domain:      "build",
		PackagePath: "internal/module/artifact/config/compile",
		OptionsType: "CompileConfig",
	},
	"server": {
		Domain:      "server",
		PackagePath: "internal/server/serverconfig",
		OptionsType: "ServerConfig",
	},
	"auth": {
		Domain:      "auth",
		PackagePath: "internal/config/authoptions",
		OptionsType: "AuthConfig",
	},
	"document": {
		Domain:      "document",
		PackagePath: "internal/document/documentconfig",
		OptionsType: "DocumentConfig",
	},
	"task": {
		Domain:      "task",
		PackagePath: "internal/task/taskconfig",
		OptionsType: "TaskConfig",
	},
	"frontendEnv": {
		Domain:      "build",
		PackagePath: "internal/module/artifact/config/env",
		OptionsType: "RuntimeEnvironmentConfig",
	},
	"backendEnv": {
		Domain:      "build",
		PackagePath: "internal/module/artifact/config/env",
		OptionsType: "RuntimeEnvironmentConfig",
	},
}

var (
	validateConfigRootOwnerMapOnce sync.Once
	validateConfigRootOwnerMapErr  error
)

func ensureConfigRootOwnerMap() error {
	validateConfigRootOwnerMapOnce.Do(func() {
		validateConfigRootOwnerMapErr = validateRootOwnerMap(reflect.TypeOf(Config{}), configRootOwnerMap)
	})
	return validateConfigRootOwnerMapErr
}

func validateRootOwnerMap(configType reflect.Type, owners map[string]ConfigRootOwner) error {
	if configType.Kind() == reflect.Pointer {
		configType = configType.Elem()
	}
	if configType.Kind() != reflect.Struct {
		return xfmt.Errorf("config owner map validation requires a struct type, got %s", configType.Kind())
	}
	if len(owners) == 0 {
		return xfmt.Errorf("config root owner map is empty")
	}

	fieldByKey := make(map[string]string)
	for i := 0; i < configType.NumField(); i++ {
		field := configType.Field(i)
		if !field.IsExported() {
			continue
		}

		tag := strings.TrimSpace(field.Tag.Get("mapstructure"))
		if tag == "-" || strings.HasPrefix(tag, "-,") {
			continue
		}

		key := rootMapstructureKey(tag)
		if key == "" {
			return xfmt.Errorf("config root field %s has invalid mapstructure tag %q", field.Name, tag)
		}
		if prevField, exists := fieldByKey[key]; exists {
			return xfmt.Errorf("config root mapstructure key %q is duplicated by fields %s and %s", key, prevField, field.Name)
		}
		fieldByKey[key] = field.Name

		owner, ok := owners[key]
		if !ok {
			return xfmt.Errorf("config root key %q (field %s) is missing from owner map", key, field.Name)
		}
		if err := validateOwnerSpec(key, owner); err != nil {
			return xfmt.Errorf("config root key %q (field %s) has invalid owner metadata: %w", key, field.Name, err)
		}
	}

	for key, owner := range owners {
		normalizedKey := strings.TrimSpace(key)
		if normalizedKey == "" {
			return xfmt.Errorf("config root owner map contains empty key")
		}
		if err := validateOwnerSpec(normalizedKey, owner); err != nil {
			return xfmt.Errorf("config root owner map key %q has invalid owner metadata: %w", normalizedKey, err)
		}
		if _, ok := fieldByKey[normalizedKey]; !ok {
			return xfmt.Errorf("config root owner map key %q does not match any Config field", normalizedKey)
		}
	}

	return nil
}

func validateOwnerSpec(key string, owner ConfigRootOwner) error {
	if strings.TrimSpace(owner.Domain) == "" {
		return xfmt.Errorf("empty domain")
	}
	if strings.TrimSpace(owner.PackagePath) == "" {
		return xfmt.Errorf("empty package")
	}
	if strings.TrimSpace(owner.OptionsType) == "" {
		return xfmt.Errorf("empty options type")
	}
	if strings.Contains(key, " ") {
		return xfmt.Errorf("invalid key %q", key)
	}
	return nil
}

func rootMapstructureKey(tag string) string {
	trimmed := strings.TrimSpace(tag)
	if trimmed == "" || trimmed == "-" || strings.HasPrefix(trimmed, "-,") {
		return ""
	}
	parts := strings.Split(trimmed, ",")
	key := strings.TrimSpace(parts[0])
	if key == "" || key == "-" {
		return ""
	}
	return key
}

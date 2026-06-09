// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package config

// ModulesPathConfig owns the root-level modules_path mapping.
type ModulesPathConfig struct {
	ModulesPath string `mapstructure:"modules_path"`
}

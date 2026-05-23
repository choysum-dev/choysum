// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package config

// AddonsPathConfig owns the root-level addons_path mapping.
type AddonsPathConfig struct {
	AddonsPath string `mapstructure:"addons_path"`
}

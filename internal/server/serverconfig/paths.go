// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package serverconfig

// DistPathConfig owns the root-level dist_path mapping.
type DistPathConfig struct {
	DistPath string `mapstructure:"dist_path"`
}

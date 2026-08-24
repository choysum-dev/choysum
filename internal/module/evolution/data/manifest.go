// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package dataloader

import "github.com/choysum-dev/choysum/pkg/meta"

// ManifestDataFiles decodes manifest choysum.data paths for module.
func ManifestDataFiles(mod *meta.Module) ([]string, error) {
	if mod == nil {
		return nil, nil
	}
	return decodeStringArray(mod.DataStr)
}

// ManifestDemoFiles decodes manifest choysum.demo paths for module.
func ManifestDemoFiles(mod *meta.Module) ([]string, error) {
	if mod == nil {
		return nil, nil
	}
	return decodeStringArray(mod.DemoStr)
}

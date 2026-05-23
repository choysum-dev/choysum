// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

// Package jsengine defines the public JavaScript engine contracts used across
// Choysum. Runtime plugin composition is centered on RuntimePluginAssembly,
// which keeps base-slot replacements separate from appended plugins so callers
// can express assembly intent explicitly at package boundaries. Lower-level
// slice helpers remain available for advanced callers that already manage
// replacement and extra plugin slices directly.
package jsengine

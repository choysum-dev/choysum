// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

// Package quickjsengine exposes the public QuickJS engine implementation and
// its runtime extension seam. Preferred runtime assembly goes through
// jsengine.RuntimePluginAssembly together with ApplyRuntimePluginAssembly so
// published replacement slots stay separate from appended plugins. QuickJS
// platform service bridges live in sibling package quickjsbridge.
// NormalizeRuntimePluginAssembly remains available for advanced callers that
// need to inspect or persist a policy-filtered assembly before applying it.
package quickjsengine

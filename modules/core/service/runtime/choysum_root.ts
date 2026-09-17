// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Resolve the injected `$choysum` runtime carrier.
 * Prefer the ambient binding; fall back to `globalThis` for unit harnesses.
 */
export function getChoysumRuntime(): typeof $choysum | undefined {
  return typeof $choysum !== 'undefined' ? $choysum : globalThis.$choysum;
}

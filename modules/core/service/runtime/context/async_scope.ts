// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Helpers for withContext / withUser promise detection.
 *
 * QuickJS has no AsyncLocalStorage. Nested withContext/withUser after an outer
 * await is required by repository/update and bilingual test paths, so we do not
 * reject overlapping async scopes. Concurrent sibling scopes (e.g. Promise.all of
 * two withUser) remain unsupported and can corrupt the shared carrier stacks.
 */

export function isPromiseLikeResult(value: unknown): value is PromiseLike<unknown> {
  return !!value && typeof (value as { then?: unknown }).then === 'function';
}

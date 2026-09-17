// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Set mutually exclusive fields to null on a row (onchange / prepare paths).
 *
 * Prefer this over assigning null through a type assertion when clearing FK / selection scopes.
 */
export function clearExclusive<T>(row: T, keys: Array<keyof T>): void {
  for (const k of keys) {
    (row as Record<keyof T, unknown>)[k] = null;
  }
}

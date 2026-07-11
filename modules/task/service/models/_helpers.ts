// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Shared pure helpers for task service models.
 */

/**
 * Clamp a requested page size to a supported range with a fallback.
 *
 * Returns `fallback` when the input is missing, non-number, or ≤ 0.
 * Otherwise returns the input capped at `max`.
 */
export function clampLimit(limit?: number, fallback: number = 50, max: number = 500): number {
  const val = typeof limit === 'number' ? limit : fallback;
  if (val <= 0) return fallback;
  return Math.min(val, max);
}

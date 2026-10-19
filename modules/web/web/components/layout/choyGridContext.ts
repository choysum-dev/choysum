// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { InjectionKey, Ref } from 'vue';

/** Parent ChoyGrid column count for ChoyCol span clamping. */
export const ChoyGridColsKey: InjectionKey<Ref<number>> = Symbol.for('choysum.choyGridCols');

/** Upper bound for generated grid tracks (guards against pathological input). */
export const MAX_CHOY_GRID_COLS = 24;

/** Reads a template-attribute number, treating blank / whitespace as unset. */
function toAttrNumber(value: unknown): number {
  return value == null || String(value).trim() === '' ? Number.NaN : Number(value);
}

/** Normalizes a parent grid track count to a positive integer (defaults to 12). */
export function normalizeChoyGridCols(cols?: number): number {
  const colsRaw = toAttrNumber(cols);
  return Number.isFinite(colsRaw)
    ? Math.min(MAX_CHOY_GRID_COLS, Math.max(1, Math.floor(colsRaw)))
    : 12;
}

/**
 * Clamps a ChoyCol span against the parent track count.
 * Invalid / missing cols default to 12; span defaults to full width.
 */
export function resolveChoyColSpan(span?: number | null, cols?: number): number {
  const safeCols = normalizeChoyGridCols(cols);
  const spanRaw = toAttrNumber(span);
  const spanBase = Number.isFinite(spanRaw) ? Math.floor(spanRaw) : safeCols;
  return Math.min(Math.max(spanBase, 1), safeCols);
}

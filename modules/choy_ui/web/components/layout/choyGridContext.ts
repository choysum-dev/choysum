// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { InjectionKey, Ref } from 'vue';

/** Parent ChoyGrid column count for ChoyCol span clamping. */
export const ChoyGridColsKey: InjectionKey<Ref<number>> = Symbol.for('choysum.choyGridCols');

/**
 * Clamps a ChoyCol span against the parent track count.
 * Invalid / missing cols default to 12; span defaults to full width.
 */
export function resolveChoyColSpan(span?: number, cols?: number): number {
  const colsRaw = Number(cols ?? 12);
  const safeCols = Number.isFinite(colsRaw) ? Math.max(1, Math.floor(colsRaw)) : 12;
  const spanRaw = span === undefined ? safeCols : Number(span);
  const spanBase = Number.isFinite(spanRaw) ? Math.floor(spanRaw) : safeCols;
  return Math.min(Math.max(spanBase, 1), safeCols);
}

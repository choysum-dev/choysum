// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  CHOY_KANBAN_DEFAULT_CARD_SIZE,
  CHOY_KANBAN_VIRTUAL_THRESHOLD,
  type ChoyKanbanVirtualizeMode,
} from '@/web/web/components/view/kanbanVirtualHelpers';

/** Re-export list kanban threshold so O2M board stays aligned. */
export const O2M_KANBAN_VIRTUAL_THRESHOLD = CHOY_KANBAN_VIRTUAL_THRESHOLD;

export const O2M_KANBAN_DEFAULT_CARD_SIZE = CHOY_KANBAN_DEFAULT_CARD_SIZE;

export type O2MKanbanVirtualizeMode = ChoyKanbanVirtualizeMode;

/** Column count for CSS `auto-fill` / `minmax(minCardWidth, 1fr)` boards. */
export function o2mKanbanColumnCount(opts: {
  boardWidth: number;
  minCardWidth: number;
  gap: number;
}): number {
  const width = Math.max(0, Number(opts.boardWidth) || 0);
  const minW = Math.max(1, Math.floor(Number(opts.minCardWidth) || 1));
  const gap = Math.max(0, Number(opts.gap) || 0);
  if (width <= 0) return 1;
  const cols = Math.floor((width + gap) / (minW + gap));
  return Math.max(1, cols);
}

/** Logical row count for a wrap grid of `cardCount` cards. */
export function o2mKanbanRowCount(cardCount: number, cols: number): number {
  const n = Math.max(0, Math.floor(Number(cardCount) || 0));
  const c = Math.max(1, Math.floor(Number(cols) || 1));
  if (n <= 0) return 0;
  return Math.ceil(n / c);
}

/** Card indices in a wrap-grid row (`rowIndex` is 0-based). */
export function o2mKanbanRowCardIndices(
  rowIndex: number,
  cols: number,
  cardCount: number,
): number[] {
  const c = Math.max(1, Math.floor(Number(cols) || 1));
  const n = Math.max(0, Math.floor(Number(cardCount) || 0));
  const start = Math.max(0, Math.floor(Number(rowIndex) || 0)) * c;
  if (start >= n) return [];
  const end = Math.min(n, start + c);
  const out: number[] = [];
  for (let i = start; i < end; i += 1) out.push(i);
  return out;
}

/**
 * Whether the O2M wrap board should mount TanStack Virtual.
 * `auto` requires at least O2M_KANBAN_VIRTUAL_THRESHOLD cards (same as list lanes).
 */
export function o2mKanbanShouldVirtualize(
  virtualize: O2MKanbanVirtualizeMode | undefined,
  cardCount: number,
): boolean {
  const n = Math.max(0, Math.floor(Number(cardCount) || 0));
  if (virtualize === false) return false;
  if (virtualize === true) return n > 0;
  return n >= O2M_KANBAN_VIRTUAL_THRESHOLD;
}

export function resolveO2MKanbanEstimateCardSize(raw: unknown): number {
  const n = Number(raw);
  if (!Number.isFinite(n) || n <= 0) return O2M_KANBAN_DEFAULT_CARD_SIZE;
  return Math.floor(n);
}

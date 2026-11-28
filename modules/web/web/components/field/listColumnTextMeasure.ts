// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Lightweight text width helpers for list / to-many column sizing.
 * Prefer canvas measureText when available; fall back to a conservative char budget.
 */

import {
  LIST_INDEX_COLUMN_MIN_WIDTH,
  listColumnPrefersStretch,
  snapListColumnWidthUp,
  snapListIndexColumnWidthUp,
} from './listColumnSizeDefaults';

/** Matches list header/cell `px-2` both sides. */
export const LIST_COLUMN_CELL_PAD_PX = 16;

/** Conservative fallback when canvas is unavailable (~CJK-safe). */
export const LIST_COLUMN_CHAR_PX = 12;

/** Tabular digit fallback for index chrome when canvas is unavailable. */
export const LIST_INDEX_DIGIT_CHAR_PX = 8;

/** First-screen row sample for soft min raise (~one viewport of dense rows). */
export const LIST_SOFT_SAMPLE_ROW_LIMIT = 20;

/** Default list body/header font used for offscreen measure. */
export const LIST_COLUMN_MEASURE_FONT =
  '13px ui-sans-serif, system-ui, -apple-system, "Segoe UI", Roboto, "Helvetica Neue", Arial, "Noto Sans", "PingFang SC", "Microsoft YaHei", sans-serif';

export type ListTextMeasureFn = (text: string) => number;

let cachedMeasure: ListTextMeasureFn | null | undefined;

/** Char-budget width (no DOM / canvas). */
export function estimateListTextWidthByChars(text: string): number {
  const s = String(text || '');
  if (!s) return 0;
  return Math.ceil(s.length * LIST_COLUMN_CHAR_PX);
}

/**
 * Build a canvas measureText fn, or null when the environment has no canvas.
 * Injected in unit tests; browsers get a lazy singleton via getDefaultListTextMeasure.
 */
export function createListCanvasTextMeasure(
  font: string = LIST_COLUMN_MEASURE_FONT,
): ListTextMeasureFn | null {
  try {
    const doc = typeof document !== 'undefined' ? document : null;
    if (!doc || typeof doc.createElement !== 'function') return null;
    const canvas = doc.createElement('canvas');
    const ctx = canvas.getContext?.('2d');
    if (!ctx || typeof ctx.measureText !== 'function') return null;
    ctx.font = font;
    return (text: string) => {
      const s = String(text || '');
      if (!s) return 0;
      const w = ctx.measureText(s).width;
      return Number.isFinite(w) && w > 0 ? Math.ceil(w) : estimateListTextWidthByChars(s);
    };
  } catch {
    return null;
  }
}

/** Lazy browser canvas measure (null in headless / QuickJS without canvas). */
export function getDefaultListTextMeasure(): ListTextMeasureFn | null {
  if (cachedMeasure !== undefined) return cachedMeasure;
  cachedMeasure = createListCanvasTextMeasure();
  return cachedMeasure;
}

/** Test helper: clear the lazy canvas singleton. */
export function resetDefaultListTextMeasureForTests(): void {
  cachedMeasure = undefined;
}

/** Measure text width with optional override; falls back to char budget. */
export function listTextWidthPx(
  text: unknown,
  measure?: ListTextMeasureFn | null,
): number {
  const s = String(text ?? '').trim();
  if (!s) return 0;
  const fn = measure === undefined ? getDefaultListTextMeasure() : measure;
  if (typeof fn === 'function') {
    const w = fn(s);
    if (Number.isFinite(w) && w > 0) return Math.ceil(w);
  }
  return estimateListTextWidthByChars(s);
}

/** Hard-header raise cap relative to type min (F3). */
const HEADER_RAISE_CAP_PX = 80;

/**
 * Hard-column header budget with font measure when available.
 * Soft columns should not use this path for type sizing.
 */
export function estimateListHeaderMinWidth(
  label: unknown,
  typeMin: number,
  opts?: { measure?: ListTextMeasureFn | null },
): number {
  const base = Number.isFinite(typeMin) && typeMin > 0 ? typeMin : 80;
  const text = String(label || '').trim();
  if (!text) return base;
  const content = listTextWidthPx(text, opts?.measure);
  const budget = content + LIST_COLUMN_CELL_PAD_PX;
  const raised = Math.max(base, Math.min(base + HEADER_RAISE_CAP_PX, budget));
  return snapListColumnWidthUp(raised);
}

/**
 * Index chrome digit budget: floor at LIST_INDEX_COLUMN_MIN_WIDTH, snap on the
 * index ladder, cap at LIST_INDEX_COLUMN_MAX_WIDTH. Empty text → floor only.
 * Digit-only labels use a tighter char fallback than generic CJK budget so
 * 1–3 digit pages stay on the 44px floor without canvas.
 */
export function estimateListIndexColumnWidth(
  displayText: unknown,
  opts?: { measure?: ListTextMeasureFn | null },
): number {
  const base = LIST_INDEX_COLUMN_MIN_WIDTH;
  const text = String(displayText ?? '').trim();
  if (!text) return base;
  let content: number;
  if (/^\d+$/.test(text)) {
    const fn = opts?.measure === undefined ? getDefaultListTextMeasure() : opts.measure;
    const fallback = Math.ceil(text.length * LIST_INDEX_DIGIT_CHAR_PX);
    if (typeof fn === 'function') {
      const w = fn(text);
      content = Number.isFinite(w) && w > 0 ? Math.ceil(w) : fallback;
    } else {
      content = fallback;
    }
  } else {
    content = listTextWidthPx(text, opts?.measure);
  }
  const budget = content + LIST_COLUMN_CELL_PAD_PX;
  return snapListIndexColumnWidthUp(Math.max(base, budget));
}

/** Best-effort display string for sample cells (scalars + M2O-like objects). */
export function formatListSampleCellText(value: unknown): string {
  if (value == null) return '';
  if (typeof value === 'string') return value.trim();
  if (typeof value === 'number' || typeof value === 'boolean' || typeof value === 'bigint') {
    return String(value);
  }
  if (typeof value === 'object') {
    const o = value as Record<string, unknown>;
    for (const key of ['DisplayName', 'Name', 'name', 'label', 'Label', 'title', 'Title']) {
      const v = o[key];
      if (v != null && String(v).trim()) return String(v).trim();
    }
  }
  return '';
}

/**
 * Read a sample cell value: prefer a literal row key (e.g. `"Partner.Name"`),
 * then walk dotted paths like ChoyTableColumn `getByPath`.
 */
export function readListSampleCellValue(
  row: Record<string, unknown> | null | undefined,
  dataKey: string,
): unknown {
  if (!row || !dataKey) return undefined;
  if (Object.prototype.hasOwnProperty.call(row, dataKey)) return row[dataKey];
  if (!dataKey.includes('.')) return row[dataKey];
  const segs = dataKey.split('.').filter(Boolean);
  let cur: unknown = row;
  for (const s of segs) {
    if (cur == null || typeof cur !== 'object') return undefined;
    cur = (cur as Record<string, unknown>)[s];
  }
  return cur;
}

export type SoftSampleColumnInput = {
  key: string;
  dataKey?: string;
  title?: unknown;
  fieldType?: unknown;
  /** Author-fixed pixel width — skip sample raise. */
  explicitWidth?: number;
  /** Current type / author min (track floor). */
  minWidth: number;
  /** Soft growth cap; raise never exceeds this. */
  maxWidth?: number;
  /** Author flex/auto/percent/capped escape — skip sample raise. */
  skipSample?: boolean;
};

/**
 * Raise soft / semi-hard mins from first-screen cell (+ header) text.
 * Returns only keys that need a higher min than the current floor.
 */
export function sampleSoftColumnMinWidths(
  columns: readonly SoftSampleColumnInput[],
  rows: readonly Record<string, unknown>[],
  opts?: {
    measure?: ListTextMeasureFn | null;
    rowLimit?: number;
  },
): Map<string, number> {
  const out = new Map<string, number>();
  if (!columns.length) return out;
  const limit =
    typeof opts?.rowLimit === 'number' && opts.rowLimit > 0
      ? Math.floor(opts.rowLimit)
      : LIST_SOFT_SAMPLE_ROW_LIMIT;
  const sample = rows.length > limit ? rows.slice(0, limit) : rows;

  for (const col of columns) {
    if (col.skipSample) continue;
    if (typeof col.explicitWidth === 'number' && col.explicitWidth > 0) continue;
    if (!listColumnPrefersStretch(col.fieldType)) continue;
    const floor = Number.isFinite(col.minWidth) && col.minWidth > 0 ? col.minWidth : 140;
    const cap =
      typeof col.maxWidth === 'number' && col.maxWidth > 0
        ? col.maxWidth
        : Number.POSITIVE_INFINITY;

    const dataKey = String(col.dataKey || col.key || '').trim();
    let bestText = String(col.title ?? '').trim();
    let bestW = listTextWidthPx(bestText, opts?.measure);
    if (dataKey) {
      for (const row of sample) {
        const text = formatListSampleCellText(readListSampleCellValue(row, dataKey));
        if (!text) continue;
        const w = listTextWidthPx(text, opts?.measure);
        if (w > bestW) {
          bestW = w;
          bestText = text;
        }
      }
    }
    if (!bestText && !(bestW > 0)) continue;

    const content = bestW + LIST_COLUMN_CELL_PAD_PX;
    const raised = snapListColumnWidthUp(Math.min(cap, Math.max(floor, content)));
    if (raised > floor) {
      out.set(col.key, Math.min(cap, raised));
    }
  }
  return out;
}

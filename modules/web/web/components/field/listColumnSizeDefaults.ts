// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Default list / to-many column size by field type (20px scale).
 * Soft columns (text-like) may grow up to a type max; hard columns stay fixed.
 * Code-like leaf names are semi-hard (narrower min/max). Explicit vColumnProps win.
 */

const CODE_LIKE_LEAF = /^(Code|Sku|Ref)$/i;
const CODE_LIKE_SUFFIX = /(_code|_sku)$/i;

/** Last segment of a dotted prop / dataKey. */
export function listColumnLeafName(propOrKey: unknown): string {
  const s = String(propOrKey || '').trim();
  if (!s) return '';
  const parts = s.split('.').filter(Boolean);
  return parts.length ? String(parts[parts.length - 1]) : s;
}

/** Code / Sku / Ref style leaves — semi-hard width (not full soft stretch). */
export function isListCodeLikeLeaf(propOrKey: unknown): boolean {
  const leaf = listColumnLeafName(propOrKey);
  if (!leaf) return false;
  return CODE_LIKE_LEAF.test(leaf) || CODE_LIKE_SUFFIX.test(leaf);
}

/** Soft columns share leftover width up to a per-column max (Odoo-like). */
export function listColumnPrefersStretch(fieldType: unknown): boolean {
  const t = String(fieldType || '').trim();
  if (!t) return false;
  const lower = t.toLowerCase();
  return (
    lower === 'char' ||
    lower === 'varchar' ||
    lower === 'text' ||
    lower === 'html' ||
    t === 'ManyToOne' ||
    lower === 'many2one' ||
    lower === 'm2o'
  );
}

/** Hard columns stay fixed px (bool / number / selection / date). */
export function listColumnIsHard(fieldType: unknown): boolean {
  const t = String(fieldType || '').trim();
  if (!t) return false;
  const lower = t.toLowerCase();
  return (
    lower === 'boolean' ||
    lower === 'binary' ||
    lower === 'image' ||
    lower === 'int' ||
    lower === 'integer' ||
    lower === 'float' ||
    lower === 'double' ||
    lower === 'decimal' ||
    lower === 'monetary' ||
    lower === 'number' ||
    lower === 'selection' ||
    lower === 'date' ||
    lower === 'datetime' ||
    lower === 'timestamp'
  );
}

export const LIST_CODE_LIKE_MIN_WIDTH = 100;
export const LIST_CODE_LIKE_MAX_WIDTH = 160;

/**
 * Returns a type default minWidth, or undefined when the type has no opinion
 * (caller keeps FALLBACK-only sizing). Code-like leaves on stretch types use 100.
 */
export function defaultListColumnMinWidth(
  fieldType: unknown,
  propOrKey?: unknown,
): number | undefined {
  const t = String(fieldType || '').trim();
  if (!t) return undefined;
  const lower = t.toLowerCase();

  if (
    lower === 'boolean' ||
    lower === 'binary' ||
    lower === 'image' ||
    lower === 'int' ||
    lower === 'integer' ||
    lower === 'float' ||
    lower === 'double' ||
    lower === 'decimal' ||
    lower === 'monetary' ||
    lower === 'number'
  ) {
    return 80;
  }
  if (lower === 'selection') {
    return 100;
  }
  if (lower === 'date' || lower === 'datetime' || lower === 'timestamp') {
    return 160;
  }
  if (
    lower === 'char' ||
    lower === 'varchar' ||
    lower === 'text' ||
    lower === 'html' ||
    t === 'ManyToOne' ||
    lower === 'many2one' ||
    lower === 'm2o'
  ) {
    if (isListCodeLikeLeaf(propOrKey)) return LIST_CODE_LIKE_MIN_WIDTH;
    return 140;
  }
  return undefined;
}

/**
 * Soft / semi-hard growth cap. Hard types return undefined (fixed px track).
 * Code-like leaves cap at 160. Floor is always ≥ minWidth.
 */
export function defaultListColumnMaxWidth(
  fieldType: unknown,
  minWidth: number,
  propOrKey?: unknown,
): number | undefined {
  if (!listColumnPrefersStretch(fieldType)) return undefined;
  if (!(Number.isFinite(minWidth) && minWidth > 0)) return undefined;
  if (isListCodeLikeLeaf(propOrKey)) {
    return Math.max(LIST_CODE_LIKE_MAX_WIDTH, minWidth);
  }
  const lower = String(fieldType || '')
    .trim()
    .toLowerCase();
  let max: number;
  if (lower === 'text' || lower === 'html') {
    max = Math.min(Math.max(Math.round(minWidth * 3), 360), 560);
  } else {
    // char / varchar / m2o
    max = Math.min(Math.max(Math.round(minWidth * 2.5), 280), 360);
  }
  return Math.max(max, minWidth);
}

const LIST_WIDTH_SCALE = [80, 100, 120, 140, 160, 180, 200, 220, 240, 260, 280] as const;

/** Snap up to the nearest 20px product scale step. */
export function snapListColumnWidthUp(px: number): number {
  if (!(Number.isFinite(px) && px > 0)) return LIST_WIDTH_SCALE[0];
  for (const step of LIST_WIDTH_SCALE) {
    if (px <= step) return step;
  }
  return Math.ceil(px / 20) * 20;
}

// Re-export after snapListColumnWidthUp so the circular import with
// listColumnTextMeasure resolves to an initialized snap binding.
export {
  estimateListHeaderMinWidth,
  type ListTextMeasureFn,
} from './listColumnTextMeasure';

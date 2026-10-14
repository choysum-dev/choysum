// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Pure helpers for 1-based page indexing used by ChoyPagination.
 */

/** Clamps page into [1, totalPages], treating non-finite as 1. */
export function clampChoyPage(page: number, totalPages: number): number {
  const max = Math.max(1, Math.floor(Number.isFinite(totalPages) ? totalPages : 1));
  const raw = Number.isFinite(page) ? Math.floor(page) : 1;
  if (raw < 1) {
    return 1;
  }
  if (raw > max) {
    return max;
  }
  return raw;
}

/** Zero-based row offset for a 1-based page and page size. */
export function choyPageOffset(page: number, pageSize: number): number {
  const size =
    Number.isFinite(pageSize) && pageSize > 0 ? Math.floor(pageSize) : 0;
  const p = Number.isFinite(page) && page > 0 ? Math.floor(page) : 1;
  return Math.max(0, (p - 1) * size);
}

/** Total page count for a result set (at least 1 when pageSize is valid). */
export function choyTotalPages(total: number, pageSize: number): number {
  const size =
    Number.isFinite(pageSize) && pageSize > 0 ? Math.floor(pageSize) : 0;
  if (size <= 0) {
    return 1;
  }
  const n = Number.isFinite(total) && total > 0 ? Math.floor(total) : 0;
  return Math.max(1, Math.ceil(n / size));
}

/** Display form for the editable pager window (ASCII hyphen, Odoo-compatible). */
export function formatChoyPagerRangeValue(start: number, end: number): string {
  if (!Number.isFinite(start) || start <= 0) return '0';
  if (!Number.isFinite(end) || end <= 0 || end === start) return String(start);
  return `${start}-${end}`;
}

/**
 * Parse Odoo-style pager input: `min` or `min-max` / `min,max` / `min;max`.
 * Unicode dashes in the draft are normalized to ASCII `-` first.
 */
export function parseChoyPagerRange(
  value: string,
): { minimum: number; maximum?: number } | null {
  let s = String(value ?? '').trim();
  s = s.split('\u2013').join('-').split('\u2014').join('-');
  if (!s) return null;
  if (/^\d+$/.test(s)) {
    const n = Number(s);
    if (!Number.isFinite(n) || n < 1) return null;
    return { minimum: n };
  }
  const parts = s.split(/[-;,]/);
  if (parts.length !== 2) return null;
  const a = Number(String(parts[0] ?? '').trim());
  const b = Number(String(parts[1] ?? '').trim());
  if (!Number.isFinite(a) || !Number.isFinite(b) || a < 1 || b < 1) return null;
  return a <= b ? { minimum: a, maximum: b } : { minimum: b, maximum: a };
}

/**
 * Apply an edited pager range: jump by start index; optional end updates pageSize
 * to the window width (Odoo pager semantics without a rows-per-page select).
 */
export function applyChoyPagerRangeEdit(opts: {
  draft: string;
  total: number;
  pageSize: number;
}): { page: number; pageSize: number } | null {
  const parsed = parseChoyPagerRange(opts.draft);
  if (!parsed) return null;
  const total =
    Number.isFinite(opts.total) && opts.total > 0 ? Math.floor(opts.total) : 0;
  if (total <= 0) return null;

  let minimum = Math.min(Math.max(1, Math.floor(parsed.minimum)), total);
  let size =
    Number.isFinite(opts.pageSize) && opts.pageSize > 0
      ? Math.floor(opts.pageSize)
      : 20;

  if (parsed.maximum != null) {
    const maximum = Math.min(Math.max(minimum, Math.floor(parsed.maximum)), total);
    // A single-index draft like "5-5" jumps to that row; keep the current page size.
    if (maximum > minimum) {
      size = Math.max(1, maximum - minimum + 1);
    }
  }

  const offset = minimum - 1;
  const page = clampChoyPage(Math.floor(offset / size) + 1, choyTotalPages(total, size));
  return { page, pageSize: size };
}

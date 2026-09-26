// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Pure helpers for ChoySearchView keyword + filter query shaping.
 */

export type ChoySearchFilter = {
  field: string;
  op: string;
  value: string;
};

export type ChoySearchQuery = {
  keyword: string;
  filters: ChoySearchFilter[];
};

/** Trims keyword text; null/undefined become ''. */
export function normalizeChoySearchKeyword(keyword: string | null | undefined): string {
  return String(keyword ?? '').trim();
}

/** Builds a search query from a keyword and optional filter list. */
export function buildChoySearchQuery(
  keyword: string | null | undefined,
  filters?: ReadonlyArray<ChoySearchFilter> | null,
): ChoySearchQuery {
  return {
    keyword: normalizeChoySearchKeyword(keyword),
    filters: (filters ?? [])
      .map((f) => ({
        field: String(f.field ?? '').trim(),
        op: String(f.op ?? '').trim(),
        value: String(f.value ?? ''),
      }))
      .filter((f) => f.field !== ''),
  };
}

/**
 * Case-insensitive substring filter over the listed string fields.
 * Blank keyword returns all rows. Only primitive string/number/boolean cells
 * are compared; hosts should pre-format other values.
 */
export function filterRowsByKeyword<T extends Record<string, unknown>>(
  rows: ReadonlyArray<T>,
  keyword: string | null | undefined,
  fields: ReadonlyArray<string>,
): T[] {
  const needle = normalizeChoySearchKeyword(keyword).toLowerCase();
  if (!needle) {
    return [...rows];
  }
  const keys = fields.map((f) => String(f).trim()).filter(Boolean);
  if (!keys.length) {
    return [...rows];
  }
  return rows.filter((row) =>
    keys.some((key) => {
      const cell = row[key];
      if (
        typeof cell !== 'string' &&
        typeof cell !== 'number' &&
        typeof cell !== 'boolean'
      ) {
        return false;
      }
      return String(cell).toLowerCase().includes(needle);
    }),
  );
}

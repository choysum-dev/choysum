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

type FilterTreeNode = {
  field?: unknown;
  operator?: unknown;
  value?: unknown;
  children?: ReadonlyArray<FilterTreeNode> | null;
};

/** Flattens OSearchView ConditionGroup trees into Choy chrome filter rows. */
export function flattenChoySearchFilters(
  groups: ReadonlyArray<FilterTreeNode> | null | undefined,
): ChoySearchFilter[] {
  const out: ChoySearchFilter[] = [];
  const walk = (nodes: ReadonlyArray<FilterTreeNode> | null | undefined): void => {
    if (!nodes) return;
    for (const node of nodes) {
      if (!node || typeof node !== 'object') continue;
      if (Array.isArray(node.children)) {
        walk(node.children);
        continue;
      }
      const field = String(node.field ?? '').trim();
      if (!field) continue;
      out.push({
        field,
        op: String(node.operator ?? '').trim(),
        value: String(node.value ?? ''),
      });
    }
  };
  walk(groups);
  return out;
}

/** Adapts OSearchView query-update payload to the ChoySearchQuery chrome shape. */
export function choySearchQueryFromPayload(payload: {
  keyword?: string | null;
  appliedFilters?: ReadonlyArray<FilterTreeNode> | null;
}): ChoySearchQuery {
  return buildChoySearchQuery(
    payload?.keyword,
    flattenChoySearchFilters(payload?.appliedFilters),
  );
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

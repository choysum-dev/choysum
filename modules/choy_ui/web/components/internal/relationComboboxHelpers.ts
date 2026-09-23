// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Pure helpers for RelationCombobox / NameSearch wiring.
 * The Vue SFC owns Combobox + virtual list presentation.
 */

export type RelationOption = {
  id: string;
  label: string;
  raw?: unknown;
};

export type RelationNameSearchFn = (
  query: string,
  opts: { limit: number },
) => Promise<RelationOption[]> | RelationOption[];

/** Trims and normalizes a typeahead keyword. */
export function normalizeRelationQuery(query: string | null | undefined): string {
  return String(query ?? '').trim();
}

/** Maps NameSearch rows that already look like { Id, DisplayName }. */
export function mapNameSearchRows(
  rows: ReadonlyArray<{ Id?: unknown; DisplayName?: unknown; label?: unknown; id?: unknown }>,
): RelationOption[] {
  const out: RelationOption[] = [];
  for (const row of rows) {
    const id = String(row.Id ?? row.id ?? '').trim();
    if (!id) {
      continue;
    }
    const label = String(row.DisplayName ?? row.label ?? id).trim() || id;
    out.push({ id, label, raw: row });
  }
  return out;
}

/**
 * Runs a NameSearch-style callback and normalizes the result.
 * Rejects blank queries into an empty list (callers may still open with '').
 */
export async function runRelationNameSearch(
  search: RelationNameSearchFn,
  query: string | null | undefined,
  pageSize = 20,
): Promise<RelationOption[]> {
  const limit = Number.isFinite(pageSize) ? Math.max(1, Math.floor(pageSize)) : 20;
  const keyword = normalizeRelationQuery(query);
  const rows = await search(keyword, { limit });
  return Array.isArray(rows) ? rows.slice(0, limit) : [];
}

/** Merges a selected option into the list so the trigger keeps a label. */
export function upsertRelationOption(
  options: readonly RelationOption[],
  selected: RelationOption | null | undefined,
): RelationOption[] {
  if (!selected?.id) {
    return [...options];
  }
  if (options.some((item) => item.id === selected.id)) {
    return options.map((item) => (item.id === selected.id ? selected : item));
  }
  return [selected, ...options];
}

/** Finds an option by id. */
export function findRelationOption(
  options: readonly RelationOption[],
  id: string | null | undefined,
): RelationOption | null {
  const key = String(id ?? '').trim();
  if (!key) {
    return null;
  }
  return options.find((item) => item.id === key) ?? null;
}

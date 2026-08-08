// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { NamedFilter } from '@/web/web/query/types';

export type SavedFilterRow = {
  Id?: string;
  Name?: string;
  Condition?: any;
  IsDefault?: boolean;
  UserId?: string | null;
};

/**
 * Convert a SavedFilter row into a NamedFilter for OSearch menus / first-frame defaults.
 */
export function savedFilterToNamedFilter(row: SavedFilterRow, selected = false): NamedFilter {
  return {
    name: String(row.Name || '').trim(),
    query: (row.Condition ?? {}) as any,
    selected,
  };
}

/**
 * Merge order (SF): private IsDefault > shared IsDefault > code defaultFilters.selected.
 * At most one selected server default is promoted; code selected presets are kept only
 * when no server default wins.
 */
export function mergeSavedFilterDefaults(opts: {
  privateDefault?: SavedFilterRow | null;
  sharedDefault?: SavedFilterRow | null;
  codeDefaults?: NamedFilter[] | NamedFilter | null;
}): NamedFilter[] {
  const codeList: NamedFilter[] = !opts.codeDefaults
    ? []
    : Array.isArray(opts.codeDefaults)
      ? opts.codeDefaults
      : [opts.codeDefaults];

  if (opts.privateDefault && opts.privateDefault.IsDefault) {
    const winner = savedFilterToNamedFilter(opts.privateDefault, true);
    const rest = codeList
      .filter(nf => nf && typeof nf.name === 'string' && nf.name.length > 0 && nf.name !== winner.name)
      .map(nf => ({ ...nf, selected: false }));
    return [winner, ...rest];
  }

  if (opts.sharedDefault && opts.sharedDefault.IsDefault) {
    const winner = savedFilterToNamedFilter(opts.sharedDefault, true);
    const rest = codeList
      .filter(nf => nf && typeof nf.name === 'string' && nf.name.length > 0 && nf.name !== winner.name)
      .map(nf => ({ ...nf, selected: false }));
    return [winner, ...rest];
  }

  return codeList.map(nf => ({ ...nf }));
}

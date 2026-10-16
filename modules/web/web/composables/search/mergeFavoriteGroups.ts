// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { GroupBySpec } from '@/web/web/query/types';
import { normalizeGroupby } from '@/web/web/query/utils/grouping/normalize';

export type FavoriteGroupItem = { field: string; granularity?: string; alias?: string };

function groupKey(g: FavoriteGroupItem): string {
  return g.granularity ? `${g.field}:${g.granularity}` : g.field;
}

function asItem(g: FavoriteGroupItem): FavoriteGroupItem {
  return g.granularity ? { field: g.field, granularity: g.granularity } : { field: g.field };
}

/** Merge favorite groups into current ones; same field keeps the favorite's granularity. */
export function mergeFavoriteGroups(
  current: Array<string | GroupBySpec | FavoriteGroupItem> | null | undefined,
  favorite: Array<string | GroupBySpec | FavoriteGroupItem> | null | undefined
): FavoriteGroupItem[] {
  const cur = normalizeGroupby(current as any) as FavoriteGroupItem[];
  const fav = normalizeGroupby(favorite as any) as FavoriteGroupItem[];
  const byField = new Map<string, FavoriteGroupItem>();
  for (const g of cur) byField.set(g.field, asItem(g));
  for (const g of fav) byField.set(g.field, asItem(g));
  const out: FavoriteGroupItem[] = [];
  const seen = new Set<string>();
  for (const g of cur) {
    const next = byField.get(g.field);
    if (!next || seen.has(g.field)) continue;
    out.push(next);
    seen.add(g.field);
  }
  for (const g of fav) {
    if (seen.has(g.field)) continue;
    out.push(asItem(g));
    seen.add(g.field);
  }
  return out;
}

/**
 * Drop exact field+granularity entries that a favorite contributed.
 * User-changed granularities (different key) are kept.
 */
export function removeFavoriteGroups(
  current: Array<string | GroupBySpec | FavoriteGroupItem> | null | undefined,
  favorite: Array<string | GroupBySpec | FavoriteGroupItem> | null | undefined
): FavoriteGroupItem[] {
  const favKeys = new Set((normalizeGroupby(favorite as any) as FavoriteGroupItem[]).map(groupKey));
  return (normalizeGroupby(current as any) as FavoriteGroupItem[])
    .filter(g => !favKeys.has(groupKey(g)))
    .map(asItem);
}

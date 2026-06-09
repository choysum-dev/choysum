// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// Grouping normalization and lightweight alias extraction helpers.
import type { GroupBySpec, TemporalGranularity } from '../../types';

const KNOWN_GRANS = new Set<TemporalGranularity>(['year', 'quarter', 'month', 'week', 'day']);

export function normalizeGroupby<T = any>(
  groupby?: GroupBySpec<T> | GroupBySpec<T>[]
): Array<{ field: string; granularity?: TemporalGranularity; alias?: string }> {
  if (!groupby) return [];
  const arr = Array.isArray(groupby) ? groupby : [groupby];
  const out: Array<{ field: string; granularity?: TemporalGranularity; alias?: string }> = [];
  for (const g of arr) {
    if (!g) continue;
    if (typeof g === 'string') {
      const [f, gran] = g.split(':');
      if (gran && KNOWN_GRANS.has(gran as TemporalGranularity)) out.push({ field: f, granularity: gran as TemporalGranularity });
      else out.push({ field: f });
    } else if (typeof g === 'object') {
      let field: string = g.field;
      let gran = g.granularity as TemporalGranularity | undefined;
      if (!gran && field.includes(':')) {
        const [base, maybeGran] = field.split(':');
        if (KNOWN_GRANS.has(maybeGran as TemporalGranularity)) {
          field = base;
          gran = maybeGran as TemporalGranularity;
        }
      }
      out.push({ field, granularity: gran, alias: g.alias });
    }
  }
  return out;
}

// Returns first-level alias names for grouped keys.
export function getGroupKeyAliases(groupby: Array<{ field: string; granularity?: TemporalGranularity; alias?: string }>): string[] {
  return groupby.map(g => g.alias || g.field).filter(Boolean);
}

// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// Pure helpers for parsing grouped-field strings and formatting them for display.
import type { TemporalGranularity } from '@/web/web/query/types';

const KNOWN_GRANS = new Set<TemporalGranularity>(['year', 'quarter', 'month', 'week', 'day']);

export function parseGbString(s: string): { field: string; granularity?: TemporalGranularity } {
  const idx = s.indexOf(':');
  if (idx <= 0) return { field: s };
  const field = s.slice(0, idx);
  const gran = s
    .slice(idx + 1)
    .trim()
    .toLowerCase() as TemporalGranularity;
  return KNOWN_GRANS.has(gran) ? { field, granularity: gran } : { field: s as any };
}

export function formatGroupItemForDisplay(x: any, granLabelMap: Record<string, string>): string {
  if (typeof x === 'string') {
    const p = parseGbString(x);
    const gran = p.granularity ? ` · ${granLabelMap[p.granularity] || p.granularity}` : '';
    return `${p.field}${gran}`;
  }
  const f = x?.field || '';
  const gran = x?.granularity ? ` · ${granLabelMap[x.granularity] || x.granularity}` : '';
  return `${f}${gran}`;
}

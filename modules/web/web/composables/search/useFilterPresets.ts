// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed } from 'vue';
import { toFilters, normalizeFilters } from '@/web/web/query/utils/filter/structures';
import type { ConditionGroup, NamedFilter } from '@/web/web/query/types';

export function useFilterPresets(params: {
  store: any;
  filtersRef: import('vue').Ref<ConditionGroup[]>;
  applyNamedFilter: (nf: NamedFilter) => void;
  defaultFiltersOverride?: NamedFilter[] | NamedFilter;
}) {
  const { store, filtersRef, applyNamedFilter, defaultFiltersOverride } = params;

  type FilterMenuItem = { name: string; filter: any };
  const defaultFilterItems = computed<FilterMenuItem[]>(() => {
    const src = defaultFiltersOverride
      ? Array.isArray(defaultFiltersOverride)
        ? defaultFiltersOverride
        : [defaultFiltersOverride]
      : (((store.state as any)?.queryState?.defaultFilters || []) as Array<NamedFilter>);
    const defs = (src || []) as Array<{ name?: string; query?: any }>;
    return defs
      .filter((df): df is { name: string; query?: any } => typeof df?.name === 'string' && df.name.length > 0)
      .map(nf => {
        if (nf.query) return { name: nf.name!, filter: nf.query } as FilterMenuItem;
        const f = (normalizeFilters(toFilters(nf as any)) || [])[0];
        return f ? { name: nf.name!, filter: f } : null;
      })
      .filter((x): x is FilterMenuItem => !!x);
  });

  const appliedFilterNameSet = computed<Set<string>>(() => new Set((filtersRef.value || []).map(f => f.name).filter(Boolean) as string[]));

  function removeFiltersByName(name: string) {
    const before = (filtersRef.value || []).length;
    filtersRef.value = (filtersRef.value || []).filter((f: any) => f.name !== name);
    return (filtersRef.value || []).length !== before;
  }

  function toggleDefaultFilter(it: FilterMenuItem, onChange: (changed: boolean) => void) {
    if (it.name && appliedFilterNameSet.value.has(it.name)) {
      const changed = removeFiltersByName(it.name);
      if (changed) onChange(true);
      return;
    }
    applyNamedFilter({ name: it.name, query: it.filter } as any);
    onChange(true);
  }

  return { defaultFilterItems, appliedFilterNameSet, toggleDefaultFilter } as const;
}

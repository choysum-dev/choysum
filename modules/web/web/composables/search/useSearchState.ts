// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// Local UI search state decoupled from store.state.queryState.
// Used for temporary keyword and filter editing inside OSearch-like components.
// Controllers persist the final keyword and filters back into the shared query state.
import { ref, computed } from 'vue';
import type { ConditionGroup, NamedFilter } from '@/web/web/query/types';
import { toFilters, normalizeFilters } from '@/web/web/query/utils/filter/structures';

export interface UseSearchStateOptions {
  keywordFields?: string[];
  initialKeyword?: string;
  initialFilters?: ConditionGroup[];
}

export function useSearchState(opts: UseSearchStateOptions = {}) {
  const keyword = ref(opts.initialKeyword || '');
  const filters = ref<ConditionGroup[]>(normalizeFilters(opts.initialFilters || []) as ConditionGroup[]);
  const keywordFieldsList = ref<string[] | undefined>(opts.keywordFields);

  const hasActive = computed(() => !!keyword.value.trim() || filters.value.length > 0);
  const keywordFields = computed(() => keywordFieldsList.value || []);

  function setKeyword(v: string) {
    keyword.value = v;
  }
  function clearAll() {
    keyword.value = '';
    filters.value = [];
  }
  function applyNamedFilter(nf: NamedFilter) {
    const f = (normalizeFilters(toFilters(nf)) || [])[0];
    if (f) filters.value.push(f);
  }

  return {
    keyword,
    filters,
    keywordFields,
    hasActive,
    setKeyword,
    clearAll,
    applyNamedFilter,
  };
}

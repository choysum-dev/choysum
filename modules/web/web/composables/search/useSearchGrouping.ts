// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, toValue, type MaybeRefOrGetter } from 'vue';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type { TemporalGranularity, GroupBySpec } from '@/core/service/api/query';
import { normalizeGroupby } from '@/web/web/query/utils/grouping/normalize';
import { parseGbString } from '@/web/web/query/utils/grouping/format';
import { useGroupingOptions } from './useGroupingOptions';

export type SearchGroupByItem = string | { field: string; granularity?: TemporalGranularity };

export type AppliedGroupMenuItem = {
  key: string;
  type: 'plain' | 'temporal';
  field: string;
  granularity?: TemporalGranularity;
  label: string;
};

/**
 * Menu/tree helpers for OSearch "Group by" — keeps grouping UI out of the main component body.
 */
export function useSearchGrouping(opts: {
  store: WebModelStore<any>;
  currentAppliedGroups: MaybeRefOrGetter<Array<GroupBySpec<any>> | undefined>;
  onGroupsChange: (next: SearchGroupByItem[]) => void;
}) {
  const {
    availableGroupFields,
    groupTreeData,
    DUMMY_ROOT_SUFFIX,
    temporalComboLabel,
    treeSelectValue,
    resetTreeSelect,
  } = useGroupingOptions(opts.store);

  const currentAppliedGroups = computed<SearchGroupByItem[]>(() => {
    const gbArr = toValue(opts.currentAppliedGroups) || [];
    const out: SearchGroupByItem[] = [];
    for (const g of gbArr) {
      if (typeof g === 'string') {
        out.push(g);
      } else if (g && typeof g === 'object') {
        const field = (g as any).field ?? (g as any).name ?? (g as any).prop;
        const granularity = (g as any).granularity ?? (g as any).gran;
        if (field) out.push(granularity ? { field, granularity } : field);
      }
    }
    return out;
  });

  function setGroupbyLocal(next: SearchGroupByItem[]) {
    const normalized = normalizeGroupby(next as any) as SearchGroupByItem[];
    opts.onGroupsChange(normalized || []);
  }

  function togglePlainGroupby(field: string) {
    const list = [...currentAppliedGroups.value];
    const i = list.findIndex(gb => (typeof gb === 'string' ? gb === field : gb.field === field && !gb.granularity));
    if (i >= 0) list.splice(i, 1);
    else list.push(field);
    setGroupbyLocal(list);
  }

  function toggleTemporalGroupby(field: string, gran: TemporalGranularity) {
    const list = [...currentAppliedGroups.value];
    const i = list.findIndex(gb => {
      if (typeof gb === 'string') {
        const p = parseGbString(gb);
        return !!p.granularity && p.field === field && p.granularity === gran;
      }
      return gb.field === field && (gb.granularity || '') === gran;
    });
    if (i >= 0) list.splice(i, 1);
    else list.push({ field, granularity: gran });
    setGroupbyLocal(list);
  }

  const appliedGroupItems = computed<AppliedGroupMenuItem[]>(() => {
    const items: AppliedGroupMenuItem[] = [];
    for (const gb of currentAppliedGroups.value) {
      if (typeof gb === 'string') {
        const p = parseGbString(gb);
        if (p.granularity) {
          items.push({
            key: `cur:temp:${p.field}:${p.granularity}`,
            type: 'temporal',
            field: p.field,
            granularity: p.granularity,
            label: temporalComboLabel(p.field, p.granularity),
          });
        } else {
          const meta = availableGroupFields.value.find(f => f.prop === p.field);
          items.push({
            key: `cur:plain:${p.field}`,
            type: 'plain',
            field: p.field,
            label: meta?.label || p.field,
          });
        }
      } else {
        const field = gb.field;
        const gran = (gb.granularity || '') as TemporalGranularity | '';
        if (gran) {
          items.push({
            key: `cur:temp:${field}:${gran}`,
            type: 'temporal',
            field,
            granularity: gran as TemporalGranularity,
            label: temporalComboLabel(field, gran as TemporalGranularity),
          });
        } else {
          const meta = availableGroupFields.value.find(f => f.prop === field);
          items.push({
            key: `cur:plain:${field}`,
            type: 'plain',
            field,
            label: meta?.label || field,
          });
        }
      }
    }
    return items;
  });

  const treeProps = { value: 'value', label: 'label', children: 'children', selectable: 'selectable' } as const;

  function onTreeSelectChange(v?: string) {
    if (!v) return;
    if (v.endsWith(`:${DUMMY_ROOT_SUFFIX}`)) {
      resetTreeSelect();
      return;
    }
    if (v.startsWith('f:')) {
      togglePlainGroupby(v.slice(2));
    } else if (v.startsWith('d:')) {
      const [, field, gran] = v.split(':');
      if (field && gran) toggleTemporalGroupby(field, gran as TemporalGranularity);
    }
    resetTreeSelect();
  }

  return {
    currentAppliedGroups,
    availableGroupFields,
    groupTreeData,
    appliedGroupItems,
    treeSelectValue,
    treeProps,
    DUMMY_ROOT_SUFFIX,
    temporalComboLabel,
    togglePlainGroupby,
    toggleTemporalGroupby,
    onTreeSelectChange,
    setGroupbyLocal,
  } as const;
}

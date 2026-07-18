// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, nextTick, ref } from 'vue';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import { createTranslate } from '@/web/web/i18n';

type FieldMeta = { type?: string; relation?: string };

export function useGroupingOptions(store: WebModelStore<any>) {
  const { _t } = createTranslate('web', { scope: 'web/composables/search/useGroupingOptions' });

  const granularityOptions = computed(() => [
    { value: 'year', label: _t('Year') },
    { value: 'quarter', label: _t('Quarter') },
    { value: 'month', label: _t('Month') },
    { value: 'week', label: _t('Week') },
    { value: 'day', label: _t('Day') },
  ] as const);

  const allFields = computed(() => {
    const md = store.fieldsMetadata ?? ({} as Record<string, FieldMeta>);
    return Object.entries(md).map(([prop, meta]: any) => ({ prop, label: prop, meta }));
  });

  function isTemporalField(prop?: string) {
    const t = String((store as any)?.fieldsMetadata?.[prop!]?.type || '').toLowerCase();
    return t === 'date' || t === 'datetime' || t === 'time';
  }

  const availableGroupFields = computed(() => {
    const md = (store as any)?.fieldsMetadata ?? ({} as Record<string, any>);
    return allFields.value
      .filter(({ prop, meta }: any) => {
        const t = String(meta?.type || '').toLowerCase();
        if (prop === 'DeletedAt') return false;
        if (t === 'onetomany' || t === 'manytomany' || t === 'jsonobject') return false;
        return true;
      })
      .sort((a, b) => {
        const idA = String(md[a.prop]?.id ?? '');
        const idB = String(md[b.prop]?.id ?? '');
        const cmp = idA.localeCompare(idB, 'en', { sensitivity: 'base' });
        if (cmp !== 0) return cmp;
        return a.label.localeCompare(b.label, 'en', { sensitivity: 'base' });
      });
  });

  const temporalGroupFields = computed(() => availableGroupFields.value.filter(f => isTemporalField(f.prop)));
  const nonTemporalGroupFields = computed(() => availableGroupFields.value.filter(f => !isTemporalField(f.prop)));

  type TreeNode = { id: string; value?: string; label: string; selectable?: boolean; children?: TreeNode[] };
  const DUMMY_ROOT_SUFFIX = '__root';
  const groupTreeData = computed<TreeNode[]>(() => {
    // Preserve the already-sorted availableGroupFields order while building the tree.
    const nodes: TreeNode[] = [];
    for (const f of availableGroupFields.value) {
      if (isTemporalField(f.prop)) {
        nodes.push({
          id: `d:${f.prop}`,
          value: `d:${f.prop}:${DUMMY_ROOT_SUFFIX}`,
          label: f.label,
          selectable: false,
          children: granularityOptions.value.map(g => ({ id: `d:${f.prop}:${g.value}`, value: `d:${f.prop}:${g.value}`, label: g.label })),
        });
      } else {
        nodes.push({ id: `f:${f.prop}`, value: `f:${f.prop}`, label: f.label });
      }
    }
    return nodes;
  });

  function temporalComboLabel(field: string, gran: string) {
    const f = availableGroupFields.value.find(x => x.prop === field);
    const map: Record<string, string> = {
      year: _t('Year'),
      quarter: _t('Quarter'),
      month: _t('Month'),
      week: _t('Week'),
      day: _t('Day'),
    };
    const label = map[gran] || gran;
    return `${f?.label || field} · ${label}`;
  }

  const treeSelectValue = ref<string | undefined>();
  function resetTreeSelect() {
    nextTick(() => (treeSelectValue.value = undefined));
  }

  return {
    granularityOptions,
    availableGroupFields,
    temporalGroupFields,
    nonTemporalGroupFields,
    groupTreeData,
    DUMMY_ROOT_SUFFIX,
    temporalComboLabel,
    treeSelectValue,
    resetTreeSelect,
  } as const;
}

// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// Note: no implicit store side-effects; pure local composable aggregating state+editor.
import { toValue, watch, type MaybeRefOrGetter } from 'vue';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type { ConditionGroup, NamedFilter } from '@/web/web/query/types';
import { filtersToQuery } from '@/web/web/query/utils/condition/builder';
import { deepCloneFilter, isGroup, normalizeFilters } from '@/web/web/query/utils/filter/structures';
import { createTranslate } from '@/web/web/i18n';
import { useSearchState, type UseSearchStateOptions } from './useSearchState';
import { useSearchEditor } from './useSearchEditor';
import { resolveSearchFieldLabel } from './useSearchFieldOptions';

const { _t } = createTranslate('web', { scope: 'web/composables/search/useSearch' });

export interface UseSearchOptions extends Omit<UseSearchStateOptions, 'initialFilters'> {
  attachStore?: WebModelStore<any>; // Optional source of field metadata used while building query conditions.
  onTrigger?: (ctx: { keyword: string; filters: ConditionGroup[] }) => void; // Optional unified trigger callback.
  allowDuplicateNamedFilter?: boolean; // Whether named presets can be added more than once.
  initialFilters?: MaybeRefOrGetter<ConditionGroup[] | undefined>;
  /** When true, re-apply initialFilters whenever they change and the current filter list is empty. */
  dynamicInitialFilters?: boolean;
}

// This composable has no implicit store side effects.
// Callers decide when to sync local state back into store.state.queryState.
export function useSearch(opts: UseSearchOptions = {}) {
  const state = useSearchState({
    keywordFields: opts.keywordFields,
    initialKeyword: opts.initialKeyword,
    initialFilters: toValue(opts.initialFilters) as ConditionGroup[] | undefined,
  });
  const editor = useSearchEditor({ filters: state.filters });

  if (opts.dynamicInitialFilters) {
    watch(
      () => toValue(opts.initialFilters),
      initial => {
        const list = initial as ConditionGroup[] | undefined;
        if (list && list.length && state.filters.value.length === 0) {
          // Deep-clone + normalize so local filters do not share nested nodes with the caller source.
          state.filters.value = normalizeFilters(list.map(deepCloneFilter));
        }
      },
      { immediate: true, deep: true }
    );
  }

  function fieldLabel(prop: string): string {
    return resolveSearchFieldLabel(opts.attachStore, prop);
  }

  function summarizeFilter(f: ConditionGroup, maxDepth = 2): string {
    if (!f) return _t('(empty)');
    const parts: string[] = [];
    const children = Array.isArray(f.children) ? f.children : [];
    for (const ch of children) {
      if (isGroup(ch)) {
        if (maxDepth > 0) parts.push(summarizeFilter(ch, maxDepth - 1));
      } else {
        const c = ch as { field?: string; operator?: string };
        if (c.field && c.operator) parts.push(`${fieldLabel(c.field)} ${c.operator}`);
      }
    }
    const joiner = f.logic === 'Or' ? ' OR ' : ' AND ';
    return parts.join(joiner) || _t('(empty)');
  }
  function summarizeFilterFields(f: ConditionGroup, maxDepth = 1): string {
    if (!f) return _t('Not configured');
    const fields: string[] = [];
    const children = Array.isArray(f.children) ? f.children : [];
    for (const ch of children) {
      if (isGroup(ch)) {
        if (maxDepth > 0) fields.push(summarizeFilterFields(ch, maxDepth - 1));
      } else {
        const c = ch as { field?: string };
        if (c.field) fields.push(fieldLabel(c.field));
      }
    }
    return fields.join(', ') || _t('Not configured');
  }
  function filterTooltip(f: ConditionGroup): string {
    return summarizeFilter(f, 3);
  }

  function buildQuery(node: ConditionGroup): any {
    try {
      const fieldsMeta = (opts.attachStore as any)?.fieldsMetadata;
      return filtersToQuery([node], undefined, undefined, fieldsMeta);
    } catch {
      return undefined;
    }
  }

  function applyNamedFilter(nf: NamedFilter) {
    if (!opts.allowDuplicateNamedFilter && nf.name) {
      const exists = state.filters.value.some(f => f.name === nf.name);
      if (exists) return;
    }
    state.applyNamedFilter(nf);
    opts.onTrigger?.({ keyword: state.keyword.value, filters: state.filters.value });
  }

  function popLastFilter(perform?: boolean): boolean {
    const arr = state.filters.value;
    if (!arr.length) return false;
    if (perform) {
      arr.pop();
      opts.onTrigger?.({ keyword: state.keyword.value, filters: state.filters.value });
      return true;
    }
    return true;
  }

  function trigger() {
    opts.onTrigger?.({ keyword: state.keyword.value, filters: state.filters.value });
  }

  function clearAll() {
    state.clearAll();
    trigger();
  }

  return {
    state: {
      keyword: state.keyword,
      filters: state.filters,
      keywordFields: state.keywordFields,
      hasActive: state.hasActive,
    },
    editor: {
      isEditorOpen: editor.isEditorOpen,
      activeFilterId: editor.activeFilterId,
      draftFilter: editor.draftFilter,
      openNew: editor.openNewFilter,
      openEdit: editor.openEditFilter,
      close: editor.closeEditor,
      setLogic: editor.setDraftLogic,
      addGroup: editor.addDraftGroup,
      removeGroup: editor.removeDraftGroup,
      addCondition: editor.addDraftCondition,
      updateCondition: editor.updateDraftCondition,
      removeCondition: editor.removeDraftCondition,
      saveDraft: editor.saveDraft,
      deleteFilter: editor.deleteFilter,
    },
    actions: {
      applyNamedFilter,
      clearAll,
      popLastFilter,
      trigger,
    },
    helpers: {
      summarizeFilter,
      summarizeFilterFields,
      filterTooltip,
      buildQuery,
      fieldLabel,
    },
  } as const;
}

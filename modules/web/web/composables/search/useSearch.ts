// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// Note: no implicit store side-effects; pure local composable aggregating state+editor.
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type { ConditionGroup, NamedFilter } from '@/web/web/query/types';
import { filtersToQuery } from '@/web/web/query/utils/condition/builder';
import { useSearchState, type UseSearchStateOptions } from './useSearchState';
import { useSearchEditor } from './useSearchEditor';

export interface UseSearchOptions extends UseSearchStateOptions {
  attachStore?: WebModelStore<any>; // Optional source of field metadata used while building query conditions.
  onTrigger?: (ctx: { keyword: string; filters: ConditionGroup[] }) => void; // Optional unified trigger callback.
  allowDuplicateNamedFilter?: boolean; // Whether named presets can be added more than once.
  dynamicInitialFilters?: boolean; // Reapply initialFilters when they change and the current filters are empty.
}

// This composable has no implicit store side effects.
// Callers decide when to sync local state back into store.state.queryState.
export function useSearch(opts: UseSearchOptions = {}) {
  const state = useSearchState({
    keywordFields: opts.keywordFields,
    initialKeyword: opts.initialKeyword,
    initialFilters: opts.initialFilters as ConditionGroup[] | undefined,
  });
  const editor = useSearchEditor({ filters: state.filters });

  // Optional dynamic initial-filter injection.
  if (opts.dynamicInitialFilters) {
    const initial = opts.initialFilters as ConditionGroup[] | undefined;
    if (initial && initial.length && state.filters.value.length === 0) {
      state.filters.value = initial.slice();
    }
  }

  // ===== summarization helpers (reintroduced for backward compatibility) =====
  function summarizeFilter(f: ConditionGroup, maxDepth = 2): string {
    if (!f) return '(空)';
    const parts: string[] = [];
    const children = Array.isArray((f as any).children) ? (f as any).children : [];
    for (const ch of children) {
      if ((ch as any).children) {
        if (maxDepth > 0) parts.push(summarizeFilter(ch as ConditionGroup, maxDepth - 1));
      } else {
        const c = ch as any as { field?: string; operator?: string };
        if (c.field && c.operator) parts.push(`${c.field} ${c.operator}`);
      }
    }
    return parts.join(' AND ') || '(空)';
  }
  function summarizeFilterFields(f: ConditionGroup, maxDepth = 1): string {
    if (!f) return '未配置';
    const fields: string[] = [];
    const children = Array.isArray((f as any).children) ? (f as any).children : [];
    for (const ch of children) {
      if ((ch as any).children) {
        if (maxDepth > 0) fields.push(summarizeFilterFields(ch as ConditionGroup, maxDepth - 1));
      } else {
        const c = ch as any as { field?: string };
        if (c.field) fields.push(c.field);
      }
    }
    return fields.join(', ') || '未配置';
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
    const f = (state as any).applyNamedFilter(nf);
    // applyNamedFilter already mutates the filter list, so only trigger callbacks here.
    opts.onTrigger?.({ keyword: state.keyword.value, filters: state.filters.value });
    return f;
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

  // Backward-compatible property names.
  const api = {
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
      clearAll: () => {
        state.clearAll();
        trigger();
      },
      popLastFilter,
      trigger,
    },
    helpers: {
      summarizeFilter,
      summarizeFilterFields,
      filterTooltip,
      buildQuery,
    },
    // Flattened aliases kept for compatibility.
    keyword: state.keyword,
    filters: state.filters,
    keywordFields: state.keywordFields,
    hasActive: state.hasActive,
    applyNamedFilter: applyNamedFilter,
    clearAll: () => {
      state.clearAll();
      trigger();
    },
    // Legacy editor aliases.
    isEditorOpen: editor.isEditorOpen,
    activeFilterId: editor.activeFilterId,
    draftFilter: editor.draftFilter,
    openNewFilter: editor.openNewFilter,
    openEditFilter: editor.openEditFilter,
    closeEditor: editor.closeEditor,
    setDraftLogic: editor.setDraftLogic,
    addDraftGroup: editor.addDraftGroup,
    removeDraftGroup: editor.removeDraftGroup,
    addDraftCondition: editor.addDraftCondition,
    updateDraftCondition: editor.updateDraftCondition,
    removeDraftCondition: editor.removeDraftCondition,
    saveDraft: editor.saveDraft,
    deleteFilter: editor.deleteFilter,
    // Deprecated aliases kept temporarily for compatibility.
    saveDraftAndSearch: editor.saveDraft,
    deleteFilterAndSearch: editor.deleteFilter,
    // Behavioral aliases.
    triggerSearch: trigger,
    handleBackspaceDeleteLastFilter: popLastFilter,
    _nodeToExpr: buildQuery,
    summarizeFilter,
    summarizeFilterFields,
    filterTooltip,
  } as const;
  return api;
}

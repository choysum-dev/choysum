// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ref } from 'vue';
import type { ConditionGroup, Condition } from '@/web/web/query/types';

export interface UseSearchEditorOptions {
  createGroupId?: () => string;
  createCondId?: () => string;
  filters: import('vue').Ref<ConditionGroup[]>;
}

function defaultGroupId() {
  return `grp_${Date.now()}_${Math.random()}`;
}
function defaultCondId() {
  return `c_${Date.now()}_${Math.random()}`;
}

export function useSearchEditor(opts: UseSearchEditorOptions) {
  const { filters } = opts;
  const createGroupId = opts.createGroupId || defaultGroupId;
  const createCondId = opts.createCondId || defaultCondId;

  const isEditorOpen = ref(false);
  const activeFilterId = ref<string | null>(null);
  const draftFilter = ref<{ baseId?: string; root: ConditionGroup } | null>(null);

  function openNewFilter() {
    const root: ConditionGroup = { id: `draft_root_${Date.now()}`, logic: 'And', children: [] } as any;
    draftFilter.value = { baseId: undefined, root };
    isEditorOpen.value = true;
    activeFilterId.value = null;
  }
  function openEditFilter(id: string) {
    const target = filters.value.find(f => f.id === id);
    if (!target) return;
    if (!Array.isArray((target as any).children)) return;
    const cloned: ConditionGroup = JSON.parse(JSON.stringify(target));
    draftFilter.value = { baseId: id, root: cloned };
    activeFilterId.value = id;
    isEditorOpen.value = true;
  }
  function closeEditor(reset?: boolean) {
    if (reset) draftFilter.value = null;
    isEditorOpen.value = false;
    activeFilterId.value = null;
  }

  function setDraftLogic(logic: 'And' | 'Or', groupId?: string) {
    if (!draftFilter.value) return;
    const g = findGroupInDraft(groupId) || draftFilter.value.root;
    g.logic = logic;
  }
  function addDraftGroup(parentId?: string) {
    if (!draftFilter.value) return;
    const parent = findGroupInDraft(parentId) || draftFilter.value.root;
    const newGroup: ConditionGroup = { id: createGroupId(), logic: 'And', children: [] } as any;
    parent.children.push(newGroup as any);
  }
  function removeDraftGroup(groupId: string) {
    if (!draftFilter.value) return;
    if (draftFilter.value.root.id === groupId) return;
    removeNodeById(draftFilter.value.root, groupId);
  }
  function addDraftCondition(parentId?: string) {
    if (!draftFilter.value) return;
    const parent = findGroupInDraft(parentId) || draftFilter.value.root;
    const cond: Condition = { id: createCondId(), field: '', operator: '=', value: '' };
    parent.children.push(cond);
  }
  function updateDraftCondition(tempId: string, patch: Partial<Condition>) {
    if (!draftFilter.value) return;
    const node = findConditionInDraft(tempId);
    if (!node) return;
    Object.assign(node, patch);
  }
  function removeDraftCondition(tempId: string) {
    if (!draftFilter.value) return;
    removeNodeById(draftFilter.value.root, tempId);
  }

  function saveDraft(): boolean {
    if (!draftFilter.value) return false;
    const root: ConditionGroup = draftFilter.value.root;
    if (!Array.isArray(root.children) || root.children.length === 0) return false;
    if (draftFilter.value.baseId) {
      const i = filters.value.findIndex(f => f.id === draftFilter.value!.baseId);
      if (i >= 0) {
        root.id = draftFilter.value.baseId;
        filters.value.splice(i, 1, root);
      }
    } else {
      filters.value.push(root);
    }
    return true;
  }
  function deleteFilter(id: string) {
    filters.value = filters.value.filter(f => f.id !== id);
  }

  function findGroupInDraft(groupId?: string): ConditionGroup | null {
    if (!draftFilter.value) return null;
    const root: ConditionGroup = draftFilter.value.root;
    if (!groupId) return root;
    const stack: ConditionGroup[] = [root];
    while (stack.length) {
      const g = stack.pop()!;
      if (g.id === groupId) return g;
      for (const ch of g.children) if ((ch as any).children) stack.push(ch as ConditionGroup);
    }
    return null;
  }
  function findConditionInDraft(id: string): Condition | null {
    if (!draftFilter.value) return null;
    const root: ConditionGroup = draftFilter.value.root;
    const stack: ConditionGroup[] = [root];
    while (stack.length) {
      const g = stack.pop()!;
      for (const ch of g.children) {
        if ((ch as any).children) stack.push(ch as ConditionGroup);
        else if ((ch as Condition).id === id) return ch as Condition;
      }
    }
    return null;
  }
  function removeNodeById(group: ConditionGroup, id: string) {
    group.children = group.children.filter(ch => {
      if ((ch as any).children) {
        if ((ch as ConditionGroup).id === id) return false;
        removeNodeById(ch as ConditionGroup, id);
        return true;
      }
      return (ch as Condition).id !== id;
    });
  }

  return {
    // state
    isEditorOpen,
    activeFilterId,
    draftFilter,
    // ops
    openNewFilter,
    openEditFilter,
    closeEditor,
    setDraftLogic,
    addDraftGroup,
    removeDraftGroup,
    addDraftCondition,
    updateDraftCondition,
    removeDraftCondition,
    saveDraft,
    deleteFilter,
  };
}

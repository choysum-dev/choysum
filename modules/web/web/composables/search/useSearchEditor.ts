// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ref } from 'vue';
import type { ConditionGroup, Condition } from '@/web/web/query/types';
import {
  countFilterLeaves,
  createFilter,
  deepCloneFilter,
  genId,
  isGroup,
  normalizeFilters,
} from '@/web/web/query/utils/filter/structures';

export interface UseSearchEditorOptions {
  createGroupId?: () => string;
  createCondId?: () => string;
  filters: import('vue').Ref<ConditionGroup[]>;
}

export function useSearchEditor(opts: UseSearchEditorOptions) {
  const { filters } = opts;
  const createGroupId = opts.createGroupId || genId;
  const createCondId = opts.createCondId || genId;

  const isEditorOpen = ref(false);
  const activeFilterId = ref<string | null>(null);
  const draftFilter = ref<{ baseId?: string; root: ConditionGroup } | null>(null);

  function openNewFilter() {
    const root = createFilter('And', []);
    root.id = createGroupId();
    root.children.push({ id: createCondId(), field: '', operator: '=', value: '' });
    draftFilter.value = { baseId: undefined, root };
    isEditorOpen.value = true;
    activeFilterId.value = null;
  }
  function openEditFilter(id: string) {
    const target = filters.value.find(f => f.id === id);
    if (!target) return;
    if (!Array.isArray(target.children)) return;
    // Preserve ids so Vue keys stay stable while editing.
    const cloned = deepCloneFilter(target);
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
    const group = createFilter('And', []);
    group.id = createGroupId();
    parent.children.push(group);
  }
  function removeDraftGroup(groupId: string) {
    if (!draftFilter.value) return;
    if (draftFilter.value.root.id === groupId) return;
    removeNodeById(draftFilter.value.root, groupId);
  }
  function addDraftCondition(parentId?: string) {
    if (!draftFilter.value) return;
    const parent = findGroupInDraft(parentId) || draftFilter.value.root;
    parent.children.push({ id: createCondId(), field: '', operator: '=', value: '' });
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
    const leaves = countFilterLeaves(draftFilter.value.root);
    // Refuse while any leaf is incomplete so Confirm cannot silently drop unfinished rows.
    if (leaves.incomplete > 0 || leaves.complete === 0) return false;
    const normalized = normalizeFilters([draftFilter.value.root]);
    const root = normalized[0];
    // Drop empty groups; refuse to persist a filter that would not affect Search.
    if (!root || !Array.isArray(root.children) || root.children.length === 0) return false;
    if (draftFilter.value.baseId) {
      const i = filters.value.findIndex(f => f.id === draftFilter.value!.baseId);
      if (i >= 0) {
        root.id = draftFilter.value.baseId;
        // Preserve display name from the draft when editing a named preset tag.
        if (draftFilter.value.root.name) root.name = draftFilter.value.root.name;
        filters.value.splice(i, 1, root);
      } else {
        return false;
      }
    } else {
      if (draftFilter.value.root.name) root.name = draftFilter.value.root.name;
      filters.value.push(root);
    }
    return true;
  }
  function deleteFilter(id: string) {
    filters.value = filters.value.filter(f => f.id !== id);
  }

  function groupKey(g: ConditionGroup & { tempId?: string }) {
    return g.tempId || g.id;
  }
  function conditionKey(c: Condition & { tempId?: string }) {
    return c.tempId || c.id;
  }
  function findGroupInDraft(groupId?: string): ConditionGroup | null {
    if (!draftFilter.value) return null;
    const root: ConditionGroup = draftFilter.value.root;
    if (!groupId) return root;
    const stack: ConditionGroup[] = [root];
    while (stack.length) {
      const g = stack.pop()! as ConditionGroup & { tempId?: string };
      if (groupKey(g) === groupId) return g;
      for (const ch of g.children) if (isGroup(ch)) stack.push(ch);
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
        if (isGroup(ch)) stack.push(ch);
        else if (conditionKey(ch as Condition & { tempId?: string }) === id) return ch as Condition;
      }
    }
    return null;
  }
  function removeNodeById(group: ConditionGroup, id: string) {
    group.children = group.children.filter(ch => {
      if (isGroup(ch)) {
        if (groupKey(ch as ConditionGroup & { tempId?: string }) === id) return false;
        removeNodeById(ch, id);
        return true;
      }
      return conditionKey(ch as Condition & { tempId?: string }) !== id;
    });
  }

  return {
    isEditorOpen,
    activeFilterId,
    draftFilter,
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

// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi } from 'vitest';
import { ref, nextTick } from 'vue';
import { useSearch } from './useSearch';
import { useSearchEditor } from './useSearchEditor';
import { useFilterPresets } from './useFilterPresets';
import { deepCloneFilter, createFilter, createCondition } from '@/web/web/query/utils/filter/structures';
import type { ConditionGroup, NamedFilter } from '@/web/web/query/types';

vi.mock('@/web/web/i18n', async () => {
  const actual = await vi.importActual<typeof import('@/web/web/i18n')>('@/web/web/i18n');
  return {
    ...actual,
    createTranslate: () => ({
      _t: (msg: string) => msg,
    }),
  };
});

describe('useSearch summarizeFilter', () => {
  it('joins sibling conditions with OR when group logic is Or', () => {
    const { helpers } = useSearch({});
    const group: ConditionGroup = {
      id: 'g1',
      logic: 'Or',
      children: [
        { id: 'c1', field: 'Name', operator: '=', value: 'a' },
        { id: 'c2', field: 'Code', operator: 'like', value: 'b' },
      ],
    };
    expect(helpers.summarizeFilter(group)).toBe('Name = OR Code like');
  });

  it('reapplies dynamicInitialFilters when empty', async () => {
    const initial = ref<ConditionGroup[]>([
      {
        id: 'g0',
        logic: 'And',
        children: [{ id: 'c0', field: 'Name', operator: '=', value: 'x' }],
      },
    ]);
    const api = useSearch({
      dynamicInitialFilters: true,
      initialFilters: initial,
    });
    expect(api.filters.value).toHaveLength(1);

    api.filters.value = [];
    initial.value = [
      {
        id: 'g1',
        logic: 'And',
        children: [{ id: 'c1', field: 'Code', operator: '=', value: 'y' }],
      },
    ];
    await nextTick();
    expect(api.filters.value).toHaveLength(1);
    expect((api.filters.value[0].children[0] as any).field).toBe('Code');
  });
});

describe('useSearchEditor clone/ids', () => {
  it('preserves ids when opening an edit draft', () => {
    const filters = ref<ConditionGroup[]>([
      {
        id: 'root-1',
        logic: 'And',
        children: [{ id: 'cond-1', field: 'Name', operator: '=', value: 'a' }],
      },
    ]);
    const editor = useSearchEditor({ filters });
    editor.openEditFilter('root-1');
    expect(editor.draftFilter.value?.baseId).toBe('root-1');
    expect(editor.draftFilter.value?.root.id).toBe('root-1');
    expect((editor.draftFilter.value?.root.children[0] as any).id).toBe('cond-1');
    // Mutating the draft must not mutate the live filter until save.
    (editor.draftFilter.value!.root.children[0] as any).value = 'changed';
    expect((filters.value[0].children[0] as any).value).toBe('a');
  });

  it('rejects save when all conditions are incomplete', () => {
    const filters = ref<ConditionGroup[]>([]);
    const editor = useSearchEditor({ filters });
    editor.openNewFilter();
    editor.addDraftCondition();
    expect(editor.saveDraft()).toBe(false);
    expect(filters.value).toHaveLength(0);
  });

  it('saves only complete conditions after normalize', () => {
    const filters = ref<ConditionGroup[]>([]);
    const editor = useSearchEditor({ filters });
    editor.openNewFilter();
    editor.addDraftCondition();
    editor.addDraftCondition();
    const [incomplete, complete] = editor.draftFilter.value!.root.children as any[];
    incomplete.field = '';
    complete.field = 'Name';
    complete.operator = '=';
    complete.value = 'ok';
    expect(editor.saveDraft()).toBe(true);
    expect(filters.value).toHaveLength(1);
    expect(filters.value[0].children).toHaveLength(1);
    expect((filters.value[0].children[0] as any).field).toBe('Name');
  });

  it('clears draft when closeEditor(true)', () => {
    const filters = ref<ConditionGroup[]>([]);
    const editor = useSearchEditor({ filters });
    editor.openNewFilter();
    expect(editor.draftFilter.value).not.toBeNull();
    editor.closeEditor(true);
    expect(editor.draftFilter.value).toBeNull();
    expect(editor.isEditorOpen.value).toBe(false);
  });
});

describe('deepCloneFilter', () => {
  it('clones nested groups without sharing references', () => {
    const original = createFilter('Or', [createCondition('Name', '=', 'a')]);
    const cloned = deepCloneFilter(original);
    expect(cloned).not.toBe(original);
    expect(cloned.children[0]).not.toBe(original.children[0]);
    expect(cloned.id).toBe(original.id);
    (cloned.children[0] as any).value = 'b';
    expect((original.children[0] as any).value).toBe('a');
  });
});

describe('useFilterPresets reactivity', () => {
  it('updates menu items when override getter returns new presets', async () => {
    const filtersRef = ref<ConditionGroup[]>([]);
    const override = ref<NamedFilter[] | undefined>([{ name: 'Active', query: ['Active', '=', true] }]);
    const { defaultFilterItems } = useFilterPresets({
      store: { state: { queryState: {} } },
      filtersRef,
      applyNamedFilter: () => {},
      defaultFiltersOverride: () => override.value,
    });
    expect(defaultFilterItems.value.map(i => i.name)).toEqual(['Active']);
    override.value = [{ name: 'Archived', query: ['Active', '=', false] }];
    await nextTick();
    expect(defaultFilterItems.value.map(i => i.name)).toEqual(['Archived']);
  });
});

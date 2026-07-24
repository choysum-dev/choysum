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
    const { state } = useSearch({
      dynamicInitialFilters: true,
      initialFilters: initial,
    });
    expect(state.filters.value).toHaveLength(1);

    state.filters.value = [];
    const nextSource: ConditionGroup = {
      id: 'g1',
      logic: 'And',
      children: [{ id: 'c1', field: 'Code', operator: '=', value: { nested: 'y' } }],
    };
    initial.value = [nextSource];
    await nextTick();
    expect(state.filters.value).toHaveLength(1);
    expect(state.filters.value[0]).not.toBe(nextSource);
    expect(state.filters.value[0].children[0]).not.toBe(nextSource.children[0]);
    expect((state.filters.value[0].children[0] as any).field).toBe('Code');
    (state.filters.value[0].children[0] as any).value.nested = 'mutated';
    expect((nextSource.children[0] as any).value.nested).toBe('y');
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

describe('useSearch actions and helpers', () => {
  it('applies named filters, blocks duplicates, and clears/pops with trigger', () => {
    const onTrigger = vi.fn();
    const api = useSearch({ onTrigger, allowDuplicateNamedFilter: false });
    api.actions.applyNamedFilter({ name: 'Active', query: ['Active', '=', true] } as any);
    expect(api.state.filters.value).toHaveLength(1);
    expect(onTrigger).toHaveBeenCalledTimes(1);

    api.actions.applyNamedFilter({ name: 'Active', query: ['Active', '=', false] } as any);
    expect(api.state.filters.value).toHaveLength(1);

    expect(api.actions.popLastFilter(false)).toBe(true);
    expect(api.actions.popLastFilter(true)).toBe(true);
    expect(api.state.filters.value).toHaveLength(0);
    expect(api.actions.popLastFilter(true)).toBe(false);

    api.actions.applyNamedFilter({ name: 'X', query: ['Name', '=', 'a'] } as any);
    api.actions.clearAll();
    expect(api.state.filters.value).toHaveLength(0);
    expect(api.state.keyword.value).toBe('');
  });

  it('summarizes nested groups, fields, tooltips, and buildQuery', () => {
    const store = {
      fieldsMetadata: {
        Name: { type: 'varchar', string: 'Name' },
      },
    } as any;
    const { helpers } = useSearch({ attachStore: store });
    const nested: ConditionGroup = {
      id: 'g',
      logic: 'And',
      children: [
        {
          id: 'g2',
          logic: 'Or',
          children: [
            { id: 'c1', field: 'Name', operator: '=', value: 'a' },
            { id: 'c2', field: 'Code', operator: 'like', value: 'b' },
          ],
        },
      ],
    };
    expect(helpers.summarizeFilter(nested)).toContain('Name =');
    expect(helpers.summarizeFilter(nested, 0)).toBe('(empty)');
    expect(helpers.summarizeFilterFields(nested)).toContain('Name');
    expect(helpers.summarizeFilterFields(nested, 0)).toBe('Not configured');
    expect(helpers.filterTooltip(nested)).toContain('Name =');
    expect(helpers.fieldLabel('Name')).toBe('Name');
    expect(helpers.summarizeFilter(null as any)).toBe('(empty)');
    expect(helpers.summarizeFilterFields(null as any)).toBe('Not configured');
    expect(helpers.summarizeFilter({ id: 'e', logic: 'And', children: [] })).toBe('(empty)');
    expect(helpers.buildQuery(nested)).toBeTruthy();
  });

  it('editor setLogic / groups / update / delete through structured API', () => {
    const { state, editor } = useSearch({});
    editor.openNew();
    editor.addCondition();
    const condId = (editor.draftFilter.value!.root.children[0] as any).id;
    editor.updateCondition(condId, { field: 'Name', operator: '=', value: 'z' });
    editor.setLogic('Or');
    expect(editor.draftFilter.value!.root.logic).toBe('Or');
    editor.addGroup();
    expect(editor.draftFilter.value!.root.children.some((c: any) => Array.isArray(c.children))).toBe(true);
    expect(editor.saveDraft()).toBe(true);
    expect(state.filters.value).toHaveLength(1);
    const id = state.filters.value[0].id;
    editor.openEdit(id);
    expect(editor.isEditorOpen.value).toBe(true);
    editor.close(true);
    editor.deleteFilter(id);
    expect(state.filters.value).toHaveLength(0);
  });

  it('allows duplicate named filters and triggers explicitly', () => {
    const onTrigger = vi.fn();
    const api = useSearch({ onTrigger, allowDuplicateNamedFilter: true });
    api.actions.applyNamedFilter({ name: 'Dup', query: ['A', '=', 1] } as any);
    api.actions.applyNamedFilter({ name: 'Dup', query: ['A', '=', 2] } as any);
    expect(api.state.filters.value).toHaveLength(2);
    api.actions.trigger();
    expect(onTrigger.mock.calls.length).toBeGreaterThanOrEqual(3);
  });

  it('buildQuery returns undefined when conversion throws', () => {
    const { helpers } = useSearch({
      attachStore: {
        get fieldsMetadata() {
          throw new Error('boom');
        },
      } as any,
    });
    expect(helpers.buildQuery(createFilter('And', [createCondition('Name', '=', 'a')]))).toBeUndefined();
  });
});

describe('useSearchEditor nested draft ops', () => {
  it('removes nested groups/conditions and saves named edit drafts', () => {
    const filters = ref<ConditionGroup[]>([
      {
        id: 'root-1',
        name: 'Preset',
        logic: 'And',
        children: [{ id: 'cond-1', field: 'Name', operator: '=', value: 'a' }],
      },
    ]);
    const editor = useSearchEditor({
      filters,
      createGroupId: () => 'g-new',
      createCondId: () => 'c-new',
    });
    editor.openEditFilter('missing');
    expect(editor.draftFilter.value).toBeNull();

    editor.openEditFilter('root-1');
    editor.addDraftGroup('root-1');
    expect(editor.draftFilter.value!.root.children.some((c: any) => c.id === 'g-new')).toBe(true);
    editor.addDraftCondition('g-new');
    editor.updateDraftCondition('c-new', { field: 'Code', operator: '=', value: 'z' });
    editor.setDraftLogic('Or', 'g-new');
    editor.removeDraftCondition('cond-1');
    expect(editor.draftFilter.value!.root.children.every((c: any) => c.id !== 'cond-1')).toBe(true);
    editor.removeDraftGroup('g-new');
    expect(editor.draftFilter.value!.root.children.every((c: any) => c.id !== 'g-new')).toBe(true);
    editor.addDraftCondition();
    editor.updateDraftCondition('c-new', { field: 'Name', operator: '=', value: 'b' });
    expect(editor.saveDraft()).toBe(true);
    expect(filters.value[0].name).toBe('Preset');
    expect((filters.value[0].children[0] as any).value).toBe('b');

    editor.openNewFilter();
    editor.draftFilter.value!.root.name = 'Fresh';
    editor.addDraftCondition();
    const id = (editor.draftFilter.value!.root.children[0] as any).id;
    editor.updateDraftCondition(id, { field: 'X', operator: '=', value: 1 });
    expect(editor.saveDraft()).toBe(true);
    expect(filters.value.some(f => f.name === 'Fresh')).toBe(true);

    // baseId no longer in filters list → save fails
    editor.openEditFilter(filters.value[0].id);
    const goneId = filters.value[0].id;
    filters.value = filters.value.filter(f => f.id !== goneId);
    expect(editor.saveDraft()).toBe(false);
  });

  it('no-ops draft mutations when editor is closed', () => {
    const filters = ref<ConditionGroup[]>([]);
    const editor = useSearchEditor({ filters });
    editor.setDraftLogic('Or');
    editor.addDraftGroup();
    editor.addDraftCondition();
    editor.updateDraftCondition('x', { value: 1 });
    editor.removeDraftCondition('x');
    editor.removeDraftGroup('x');
    expect(editor.saveDraft()).toBe(false);
  });
});

describe('useFilterPresets', () => {
  it('toggles named presets and builds menu items from override/store', () => {
    const filters = ref<ConditionGroup[]>([]);
    const applyNamedFilter = vi.fn((nf: NamedFilter) => {
      filters.value.push({
        id: nf.name,
        name: nf.name,
        logic: 'And',
        children: [{ id: 'c', field: 'A', operator: '=', value: 1 }],
      });
    });
    const store = {
      state: {
        queryState: {
          defaultFilters: [{ name: 'FromStore', query: ['A', '=', 1] }],
        },
      },
    };
    const api = useFilterPresets({
      store,
      filtersRef: filters,
      applyNamedFilter,
      defaultFiltersOverride: () => [{ name: 'Active', query: ['Active', '=', true] }],
    });
    expect(api.defaultFilterItems.value.map(i => i.name)).toEqual(['Active']);
    const onChange = vi.fn();
    api.toggleDefaultFilter(api.defaultFilterItems.value[0]!, onChange);
    expect(applyNamedFilter).toHaveBeenCalled();
    expect(onChange).toHaveBeenCalledWith(true);
    expect(api.appliedFilterNameSet.value.has('Active')).toBe(true);
    api.toggleDefaultFilter({ name: 'Active', filter: ['Active', '=', true] }, onChange);
    expect(filters.value.some(f => f.name === 'Active')).toBe(false);
  });

  it('reads store defaultFilters and named filters without query', () => {
    const filters = ref<ConditionGroup[]>([]);
    const applyNamedFilter = vi.fn();
    const store = {
      state: {
        queryState: {
          defaultFilters: [
            { name: 'FromStore', query: ['A', '=', 1] },
            { name: '' },
            {
              name: 'AsGroup',
              // no query — falls through toFilters/normalizeFilters
              logic: 'And',
              children: [{ id: 'c', field: 'B', operator: '=', value: 2 }],
            } as any,
          ],
        },
      },
    };
    const api = useFilterPresets({ store, filtersRef: filters, applyNamedFilter });
    expect(api.defaultFilterItems.value.map(i => i.name)).toEqual(['FromStore', 'AsGroup']);
    const onChange = vi.fn();
    api.toggleDefaultFilter({ name: 'Missing', filter: [] }, onChange);
    // Already absent — remove is a no-op; apply still runs for missing names.
    expect(applyNamedFilter).toHaveBeenCalled();
  });
});

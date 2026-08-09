// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount, flushPromises } from '@vue/test-utils';
import { nextTick } from 'vue';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import OSearch from './OSearch.vue';

const { savedFiltersApi } = vi.hoisted(() => ({
  savedFiltersApi: {
    state: null as null | {
      favoriteMenuItems: any[];
      loading: boolean;
      loadError: string | null;
      defaultsForOpen: any[];
    },
    load: vi.fn(async () => {}),
    apply: vi.fn(),
    saveCurrent: vi.fn(async () => ({ Id: '1' })),
    remove: vi.fn(async () => {}),
    lastCodeDefaults: undefined as unknown,
  },
}));

vi.mock('@/web/web/i18n', async () => {
  const actual = await vi.importActual<typeof import('@/web/web/i18n')>('@/web/web/i18n');
  return {
    ...actual,
    createTranslate: () => ({
      _t: (msg: string, ...args: unknown[]) => (args.length ? `${msg}:${args.join(',')}` : msg),
    }),
  };
});

vi.mock('@/web/web/composables/search/useSavedFilters', async () => {
  const { reactive, toRef } = await import('vue');
  savedFiltersApi.state = reactive({
    favoriteMenuItems: [] as any[],
    loading: false,
    loadError: null as string | null,
    defaultsForOpen: [] as any[],
  });
  return {
    useSavedFilters: (params: { applyNamedFilter: (nf: any) => void; codeDefaults?: () => any }) => {
      savedFiltersApi.lastCodeDefaults = params.codeDefaults?.();
      savedFiltersApi.apply.mockImplementation((fav: { name: string; filter: any }) => {
        params.applyNamedFilter({ name: fav.name, query: fav.filter });
      });
      return {
        favoriteMenuItems: toRef(savedFiltersApi.state!, 'favoriteMenuItems'),
        loading: toRef(savedFiltersApi.state!, 'loading'),
        loadError: toRef(savedFiltersApi.state!, 'loadError'),
        defaultsForOpen: toRef(savedFiltersApi.state!, 'defaultsForOpen'),
        load: savedFiltersApi.load,
        apply: savedFiltersApi.apply,
        saveCurrent: savedFiltersApi.saveCurrent,
        remove: savedFiltersApi.remove,
      };
    },
  };
});

vi.mock('element-plus', async () => {
  const actual = await vi.importActual<any>('element-plus');
  return {
    ...actual,
    ElMessage: { warning: vi.fn(), success: vi.fn(), error: vi.fn() },
    ElMessageBox: { confirm: vi.fn(async () => true) },
  };
});

function makeStore() {
  return {
    storeId: 'demo.Widget',
    fieldsMetadata: {
      Name: { id: '1', type: 'varchar', string: 'Name' },
      CreatedAt: { id: '2', type: 'datetime', string: 'Created At' },
      Status: { id: '3', type: 'selection', string: 'Status' },
    },
    state: { queryState: { defaultFilters: [{ name: 'Active', query: ['Active', '=', true] }] } },
    getFieldsGetTranslatedString: () => undefined,
  } as any;
}

const elementStubs = {
  'el-tooltip': { template: `<div><slot /></div>` },
  'el-button': {
    emits: ['click'],
    template: `<button type="button" class="el-btn" @click="$emit('click', $event)"><slot /></button>`,
  },
  'el-tag': {
    props: { closable: { type: Boolean, default: false } },
    emits: ['close', 'click'],
    template: `<span class="el-tag" @click="$emit('click', $event)">
      <slot />
      <button v-if="closable" type="button" class="tag-close" @click.stop="$emit('close', $event)" />
    </span>`,
  },
  'el-popover': {
    props: ['visible'],
    emits: ['update:visible'],
    template: `<div class="el-popover"><slot name="reference" /><div class="pop"><slot /></div></div>`,
  },
  'el-dialog': {
    props: ['modelValue', 'title'],
    emits: ['update:modelValue', 'close'],
    template: `<div v-if="modelValue" class="el-dialog" :data-title="title"><slot /><slot name="footer" /></div>`,
  },
  'el-divider': true,
  'el-icon': { template: `<i><slot /></i>` },
  'el-form': { template: `<form><slot /></form>` },
  'el-form-item': { template: `<div class="form-item"><slot /></div>` },
  'el-input': {
    props: ['modelValue'],
    emits: ['update:modelValue'],
    template: `<input class="fav-name" :value="modelValue" @input="$emit('update:modelValue', $event.target.value)" />`,
  },
  'el-checkbox': {
    props: ['modelValue'],
    emits: ['update:modelValue'],
    template: `<label class="fav-check"><input type="checkbox" :checked="modelValue" @change="$emit('update:modelValue', $event.target.checked)" /><slot /></label>`,
  },
  'el-tree-select': {
    emits: ['change', 'update:modelValue'],
    template: `<button type="button" class="tree" @click="$emit('change', 'f:Status')" />`,
  },
  OSearchFilter: {
    props: ['store', 'draft', 'fields'],
    emits: ['cancel', 'confirm'],
    template: `<div class="filter-editor">
      <button type="button" class="confirm" @click="$emit('confirm')" />
      <button type="button" class="cancel" @click="$emit('cancel')" />
    </div>`,
  },
};

describe('OSearch behavior', () => {
  beforeEach(() => {
    const st = savedFiltersApi.state!;
    st.favoriteMenuItems = [];
    st.loading = false;
    st.loadError = null;
    st.defaultsForOpen = [{ name: 'CodeDefault', query: ['A', '=', 1] }];
    savedFiltersApi.load.mockClear();
    savedFiltersApi.apply.mockClear();
    savedFiltersApi.saveCurrent.mockClear();
    savedFiltersApi.remove.mockClear();
    savedFiltersApi.load.mockResolvedValue(undefined);
    savedFiltersApi.saveCurrent.mockResolvedValue({ Id: '1' });
    savedFiltersApi.remove.mockResolvedValue(undefined);
  });

  function mountSearch(props: Record<string, any> = {}) {
    return mount(OSearch as any, {
      props: {
        store: makeStore(),
        placeholder: 'Find…',
        defaultFilters: [{ name: 'Active', query: ['Active', '=', true] }],
        ...props,
      },
      global: { stubs: elementStubs },
    });
  }

  it('emits query-update on enter / search icon and syncs controlled keyword', async () => {
    const wrapper = mountSearch({ currentKeyword: 'hello' });
    await flushPromises();
    const input = wrapper.find('input.o-search__input');
    expect((input.element as HTMLInputElement).value).toBe('hello');

    await input.trigger('keydown.enter');
    expect(wrapper.emitted('query-update')?.length).toBeGreaterThan(0);

    const before = wrapper.emitted('query-update')!.length;
    await wrapper.findAll('.el-btn')[0]!.trigger('click');
    expect(wrapper.emitted('query-update')!.length).toBeGreaterThan(before);
  });

  it('shows grouping tag and clears grouping', async () => {
    const wrapper = mountSearch({
      currentAppliedGroups: [{ field: 'Status' }, { field: 'CreatedAt', granularity: 'month' }],
    });
    await nextTick();
    expect(wrapper.find('.o-search__grouptag').exists()).toBe(true);
    await wrapper.find('.o-search__grouptag .tag-close').trigger('click');
    const payload = wrapper.emitted('query-update')!.at(-1)![0] as any;
    expect(payload.appliedGroups).toEqual([]);
  });

  it('opens custom filter editor, warns on incomplete save, and cancels', async () => {
    const { ElMessage } = await import('element-plus');
    const wrapper = mountSearch();
    await nextTick();
    const custom = wrapper.findAll('.el-btn').find(b => b.text().includes('Custom filter'));
    expect(custom).toBeTruthy();
    await custom!.trigger('click');
    await nextTick();
    expect(wrapper.find('.filter-editor').exists()).toBe(true);

    await wrapper.find('.filter-editor .confirm').trigger('click');
    expect(ElMessage.warning).toHaveBeenCalled();

    await wrapper.find('.filter-editor .cancel').trigger('click');
    await nextTick();
    expect(wrapper.find('.filter-editor').exists()).toBe(false);
  });

  it('applies controlled filters and supports tag close / backspace delete', async () => {
    const filters = [
      {
        id: 'f1',
        name: 'Active',
        logic: 'And' as const,
        children: [{ id: 'c1', field: 'Active', operator: '=', value: true }],
      },
    ];
    const wrapper = mountSearch({ currentAppliedFilters: filters });
    await flushPromises();
    expect(wrapper.find('.o-search__tag').exists()).toBe(true);

    await wrapper.find('.o-search__tag').trigger('click');
    await nextTick();
    expect(wrapper.find('.filter-editor').exists()).toBe(true);
    await wrapper.find('.filter-editor .cancel').trigger('click');
    await nextTick();

    await wrapper.find('.o-search__tag .tag-close').trigger('click');
    expect(wrapper.emitted('query-update')?.length).toBeGreaterThan(0);

    // Acknowledge the clear echo, then push a new controlled snapshot.
    await wrapper.setProps({ currentAppliedFilters: [] });
    await flushPromises();
    await wrapper.setProps({
      currentAppliedFilters: [
        {
          id: 'f2',
          name: 'X',
          logic: 'And',
          children: [{ id: 'c2', field: 'Name', operator: '=', value: 'a' }],
        },
      ],
    });
    await flushPromises();
    expect(wrapper.find('.o-search__tag').text()).toContain('X');
    await wrapper.find('.o-search__tag').trigger('click');
    await nextTick();
    expect(wrapper.find('.filter-editor').exists()).toBe(true);
  });

  it('toggles default filter from menu', async () => {
    const wrapper = mountSearch();
    await nextTick();
    const item = wrapper.findAll('.el-btn').find(b => b.text().includes('Active'));
    expect(item).toBeTruthy();
    await item!.trigger('click');
    expect(wrapper.emitted('query-update')?.length).toBeGreaterThan(0);
  });

  it('applies tree select change for grouping', async () => {
    const wrapper = mountSearch();
    await nextTick();
    await wrapper.find('.tree').trigger('click');
    expect(wrapper.emitted('query-update')?.length).toBeGreaterThan(0);
    const payload = wrapper.emitted('query-update')!.at(-1)![0] as any;
    expect(payload.appliedGroups?.some((g: any) => g.field === 'Status' || g === 'Status')).toBe(true);
  });

  it('saves a complete edited draft and closes the editor', async () => {
    const wrapper = mountSearch({
      currentAppliedFilters: [
        {
          id: 'f1',
          name: 'Named',
          logic: 'And',
          children: [{ id: 'c1', field: 'Name', operator: '=', value: 'ok' }],
        },
      ],
    });
    await flushPromises();
    await wrapper.find('.o-search__tag').trigger('click');
    await nextTick();
    expect(wrapper.find('.filter-editor').exists()).toBe(true);
    const before = wrapper.emitted('query-update')?.length ?? 0;
    await wrapper.find('.filter-editor .confirm').trigger('click');
    await flushPromises();
    expect(wrapper.find('.filter-editor').exists()).toBe(false);
    expect((wrapper.emitted('query-update')?.length ?? 0)).toBeGreaterThan(before);
  });

  it('closes quietly when edited filter disappears before save', async () => {
    const { ElMessage } = await import('element-plus');
    (ElMessage.warning as any).mockClear?.();
    const wrapper = mountSearch({
      currentAppliedFilters: [
        {
          id: 'f-gone',
          logic: 'And',
          children: [{ id: 'c1', field: 'Name', operator: '=', value: 'a' }],
        },
      ],
    });
    await flushPromises();
    await wrapper.find('.o-search__tag').trigger('click');
    await nextTick();
    expect(wrapper.find('.filter-editor').exists()).toBe(true);

    // Parent clears tags while the editor is still open.
    await wrapper.setProps({ currentAppliedFilters: [] });
    await flushPromises();
    await wrapper.find('.filter-editor .confirm').trigger('click');
    await flushPromises();
    expect(wrapper.find('.filter-editor').exists()).toBe(false);
    expect(ElMessage.warning).not.toHaveBeenCalled();
  });

  it('opens grouping menu from group tag and toggles applied menu items', async () => {
    const wrapper = mountSearch({
      currentAppliedGroups: [{ field: 'Status' }, { field: 'CreatedAt', granularity: 'month' }],
    });
    await nextTick();
    await wrapper.find('.o-search__grouptag').trigger('click');
    const statusItem = wrapper.findAll('.el-btn').find(b => b.text().includes('Status'));
    expect(statusItem).toBeTruthy();
    await statusItem!.trigger('click');
    expect(wrapper.emitted('query-update')?.length).toBeGreaterThan(0);
  });

  it('pending-deletes last filter tag via backspace then removes it', async () => {
    const wrapper = mountSearch({
      currentAppliedFilters: [
        {
          id: 'f1',
          name: 'One',
          logic: 'And',
          children: [{ id: 'c1', field: 'Name', operator: '=', value: 'a' }],
        },
      ],
    });
    await flushPromises();
    const input = wrapper.find('input.o-search__input');
    const el = input.element as HTMLInputElement;
    Object.defineProperty(el, 'selectionStart', { configurable: true, get: () => 0 });
    Object.defineProperty(el, 'selectionEnd', { configurable: true, get: () => 0 });

    el.dispatchEvent(new KeyboardEvent('keydown', { key: 'Backspace', bubbles: true, cancelable: true }));
    await nextTick();
    expect(wrapper.find('.o-search__tag--pending-delete').exists()).toBe(true);

    el.dispatchEvent(new KeyboardEvent('keydown', { key: 'Backspace', bubbles: true, cancelable: true }));
    await flushPromises();
    expect(wrapper.find('.o-search__tag').exists()).toBe(false);

    // Typing clears pending marker when a tag remains.
    await wrapper.setProps({
      currentAppliedFilters: [
        {
          id: 'f2',
          name: 'Two',
          logic: 'And',
          children: [{ id: 'c2', field: 'Name', operator: '=', value: 'b' }],
        },
      ],
    });
    await flushPromises();
    el.dispatchEvent(new KeyboardEvent('keydown', { key: 'Backspace', bubbles: true, cancelable: true }));
    await nextTick();
    el.dispatchEvent(new KeyboardEvent('keydown', { key: 'a', bubbles: true, cancelable: true }));
    await nextTick();
    expect(wrapper.find('.o-search__tag--pending-delete').exists()).toBe(false);
  });

  it('focuses the keyword input when the shell is clicked', async () => {
    const wrapper = mountSearch();
    const input = wrapper.find('input.o-search__input');
    const focus = vi.spyOn(input.element as HTMLInputElement, 'focus');
    await wrapper.find('.o-search__main').trigger('click');
    expect(focus).toHaveBeenCalled();
  });

  it('loads favorites on mount and shows empty / error / retry states', async () => {
    const wrapper = mountSearch();
    await flushPromises();
    expect(savedFiltersApi.load).toHaveBeenCalled();
    expect(wrapper.emitted('defaults-ready')?.[0]?.[0]).toEqual([{ name: 'CodeDefault', query: ['A', '=', 1] }]);
    expect(wrapper.text()).toContain('No favorites yet');

    savedFiltersApi.state!.loadError = 'boom';
    await nextTick();
    expect(wrapper.text()).toContain('Failed to load favorites');
    const before = savedFiltersApi.load.mock.calls.length;
    const retry = wrapper.findAll('.el-btn').find(b => b.text().includes('Retry'));
    expect(retry).toBeTruthy();
    await retry!.trigger('click');
    expect(savedFiltersApi.load.mock.calls.length).toBeGreaterThan(before);
  });

  it('covers codeDefaults singleton/undefined branches', async () => {
    mountSearch({ defaultFilters: undefined as any });
    await flushPromises();
    expect(savedFiltersApi.lastCodeDefaults).toBeUndefined();

    mountSearch({ defaultFilters: { name: 'Solo', query: ['X', '=', 1] } as any });
    await flushPromises();
    expect(savedFiltersApi.lastCodeDefaults).toEqual([{ name: 'Solo', query: ['X', '=', 1] }]);
  });

  it('shows Check icon for applied favorite names', async () => {
    savedFiltersApi.state!.favoriteMenuItems = [
      {
        id: 'fav-check',
        name: 'Mine',
        shared: true,
        isDefault: false,
        canDelete: false,
        filter: {},
      },
    ];
    const wrapper = mountSearch({
      currentAppliedFilters: [
        {
          id: 'f-mine',
          name: 'Mine',
          logic: 'And',
          children: [{ id: 'c1', field: 'Active', operator: '=', value: true }],
        },
      ],
    });
    await flushPromises();
    expect(wrapper.find('.o-search__menu-icon--applied').exists()).toBe(true);
    expect(wrapper.text()).toContain('Shared');
  });

  it('applies and removes favorites (confirm), and saves with empty-name warning', async () => {
    const { ElMessage, ElMessageBox } = await import('element-plus');
    (ElMessageBox.confirm as any).mockResolvedValue(true);
    savedFiltersApi.state!.favoriteMenuItems = [
      {
        id: 'fav-1',
        name: 'Mine',
        shared: false,
        isDefault: false,
        canDelete: true,
        filter: { And: [['Active', '=', true]] },
      },
    ];
    const wrapper = mountSearch();
    await flushPromises();

    const applyBtn = wrapper.findAll('.el-btn').find(b => b.text().includes('Mine'));
    expect(applyBtn).toBeTruthy();
    const beforeEmit = wrapper.emitted('query-update')?.length ?? 0;
    await applyBtn!.trigger('click');
    expect(savedFiltersApi.apply).toHaveBeenCalledWith(
      expect.objectContaining({ name: 'Mine', filter: { And: [['Active', '=', true]] } })
    );
    expect((wrapper.emitted('query-update')?.length ?? 0)).toBeGreaterThan(beforeEmit);

    const del = wrapper.find('.o-search__menu-item-delete');
    await del.trigger('click');
    await flushPromises();
    expect(ElMessageBox.confirm).toHaveBeenCalled();
    expect(savedFiltersApi.remove).toHaveBeenCalledWith('fav-1');
    expect(ElMessage.success).toHaveBeenCalled();

    const saveOpen = wrapper.findAll('.el-btn').find(b => b.text().includes('Save current filters'));
    await saveOpen!.trigger('click');
    await nextTick();
    expect(wrapper.find('.el-dialog').exists()).toBe(true);

    const saveBtn = wrapper.findAll('.el-btn').find(b => b.text() === 'Save');
    await saveBtn!.trigger('click');
    expect(ElMessage.warning).toHaveBeenCalled();
    expect(savedFiltersApi.saveCurrent).not.toHaveBeenCalled();

    await wrapper.find('input.fav-name').setValue('NewFav');
    await saveBtn!.trigger('click');
    await flushPromises();
    expect(savedFiltersApi.saveCurrent).toHaveBeenCalledWith({
      name: 'NewFav',
      isDefault: false,
      shared: false,
    });
    expect(wrapper.emitted('defaults-ready')?.length).toBeGreaterThan(1);
  });

  it('cancels favorite delete when ElMessageBox rejects', async () => {
    const { ElMessage, ElMessageBox } = await import('element-plus');
    (ElMessageBox.confirm as any).mockRejectedValueOnce('cancel');
    (ElMessage.error as any).mockClear?.();
    (ElMessage.success as any).mockClear?.();
    savedFiltersApi.state!.favoriteMenuItems = [
      {
        id: 'fav-cancel',
        name: 'KeepMe',
        shared: false,
        isDefault: false,
        canDelete: true,
        filter: {},
      },
    ];
    const wrapper = mountSearch();
    await flushPromises();
    await wrapper.find('.o-search__menu-item-delete').trigger('click');
    await flushPromises();
    expect(savedFiltersApi.remove).not.toHaveBeenCalled();
    expect(ElMessage.success).not.toHaveBeenCalled();
    expect(ElMessage.error).not.toHaveBeenCalled();
  });

  it('shows ElMessage.error when remove or save fails', async () => {
    const { ElMessage, ElMessageBox } = await import('element-plus');
    (ElMessageBox.confirm as any).mockResolvedValue(true);
    (ElMessage.error as any).mockClear?.();
    savedFiltersApi.remove.mockRejectedValueOnce(new Error('delete failed'));
    savedFiltersApi.state!.favoriteMenuItems = [
      {
        id: 'fav-err',
        name: 'Bad',
        shared: false,
        isDefault: false,
        canDelete: true,
        filter: {},
      },
    ];
    const wrapper = mountSearch();
    await flushPromises();
    await wrapper.find('.o-search__menu-item-delete').trigger('click');
    await flushPromises();
    expect(ElMessage.error).toHaveBeenCalledWith('delete failed');

    (ElMessage.error as any).mockClear?.();
    savedFiltersApi.saveCurrent.mockRejectedValueOnce('save blew up');
    const saveOpen = wrapper.findAll('.el-btn').find(b => b.text().includes('Save current filters'));
    await saveOpen!.trigger('click');
    await nextTick();
    await wrapper.find('input.fav-name').setValue('FailFav');
    const saveBtn = wrapper.findAll('.el-btn').find(b => b.text() === 'Save');
    await saveBtn!.trigger('click');
    await flushPromises();
    expect(ElMessage.error).toHaveBeenCalledWith('save blew up');
  });

  it('stringifies non-Error remove failures and Error save failures', async () => {
    const { ElMessage, ElMessageBox } = await import('element-plus');
    (ElMessageBox.confirm as any).mockResolvedValue(true);
    (ElMessage.error as any).mockClear?.();
    savedFiltersApi.remove.mockRejectedValueOnce('delete-string');
    savedFiltersApi.state!.favoriteMenuItems = [
      {
        id: 'fav-str',
        name: 'Str',
        shared: false,
        isDefault: false,
        canDelete: true,
        filter: {},
      },
    ];
    const wrapper = mountSearch();
    await flushPromises();
    await wrapper.find('.o-search__menu-item-delete').trigger('click');
    await flushPromises();
    expect(ElMessage.error).toHaveBeenCalledWith('delete-string');

    (ElMessage.error as any).mockClear?.();
    savedFiltersApi.saveCurrent.mockRejectedValueOnce(new Error('save failed'));
    const saveOpen = wrapper.findAll('.el-btn').find(b => b.text().includes('Save current filters'));
    await saveOpen!.trigger('click');
    await nextTick();
    await wrapper.find('input.fav-name').setValue('ErrFav');
    const saveBtn = wrapper.findAll('.el-btn').find(b => b.text() === 'Save');
    await saveBtn!.trigger('click');
    await flushPromises();
    expect(ElMessage.error).toHaveBeenCalledWith('save failed');
  });

  it('guards re-entrant save while saveFavoriteSaving is true', async () => {
    let resolveSave!: (v: any) => void;
    savedFiltersApi.saveCurrent.mockImplementationOnce(
      () =>
        new Promise(resolve => {
          resolveSave = resolve;
        })
    );
    const wrapper = mountSearch();
    await flushPromises();
    const saveOpen = wrapper.findAll('.el-btn').find(b => b.text().includes('Save current filters'));
    await saveOpen!.trigger('click');
    await nextTick();
    await wrapper.find('input.fav-name').setValue('Once');
    const saveBtn = wrapper.findAll('.el-btn').find(b => b.text() === 'Save');
    await saveBtn!.trigger('click');
    await saveBtn!.trigger('click');
    await nextTick();
    expect(savedFiltersApi.saveCurrent).toHaveBeenCalledTimes(1);
    resolveSave!({ Id: '1' });
    await flushPromises();
  });

  it('does not emit query-update when applying a favorite leaves filter length unchanged', async () => {
    savedFiltersApi.apply.mockImplementationOnce(() => {
      /* no-op: filters length stays the same */
    });
    savedFiltersApi.state!.favoriteMenuItems = [
      {
        id: 'fav-noop',
        name: 'Noop',
        shared: false,
        isDefault: false,
        canDelete: true,
        filter: { And: [['Active', '=', true]] },
      },
    ];
    const wrapper = mountSearch();
    await flushPromises();
    const beforeEmit = wrapper.emitted('query-update')?.length ?? 0;
    const applyBtn = wrapper.findAll('.el-btn').find(b => b.text().includes('Noop'));
    await applyBtn!.trigger('click');
    expect(savedFiltersApi.apply).toHaveBeenCalled();
    expect(wrapper.emitted('query-update')?.length ?? 0).toBe(beforeEmit);
  });

  it('saves with isDefault and shared checkboxes enabled', async () => {
    const wrapper = mountSearch();
    await flushPromises();
    const saveOpen = wrapper.findAll('.el-btn').find(b => b.text().includes('Save current filters'));
    await saveOpen!.trigger('click');
    await nextTick();
    await wrapper.find('input.fav-name').setValue('DefaultOnly');
    const defaultCheck = wrapper.findAll('.fav-check').find(l => l.text().includes('Use by default'));
    expect(defaultCheck).toBeTruthy();
    await defaultCheck!.find('input').setValue(true);
    const saveBtn = wrapper.findAll('.el-btn').find(b => b.text() === 'Save');
    await saveBtn!.trigger('click');
    await flushPromises();
    expect(savedFiltersApi.saveCurrent).toHaveBeenCalledWith({
      name: 'DefaultOnly',
      isDefault: true,
      shared: false,
    });

    savedFiltersApi.saveCurrent.mockClear();
    await saveOpen!.trigger('click');
    await nextTick();
    await wrapper.find('input.fav-name').setValue('SharedOnly');
    const sharedCheck = wrapper.findAll('.fav-check').find(l => l.text().includes('Share with all users'));
    expect(sharedCheck).toBeTruthy();
    await sharedCheck!.find('input').setValue(true);
    await wrapper.findAll('.el-btn').find(b => b.text() === 'Save')!.trigger('click');
    await flushPromises();
    expect(savedFiltersApi.saveCurrent).toHaveBeenCalledWith({
      name: 'SharedOnly',
      isDefault: false,
      shared: true,
    });
  });
});

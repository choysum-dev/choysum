// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h, nextTick, reactive, toRef } from 'vue';
import {
  ElButton,
  ElCheckbox,
  ElDialog,
  ElDivider,
  ElForm,
  ElFormItem,
  ElIcon,
  ElInput,
  ElMessage,
  ElMessageBox,
  ElPopover,
  ElTag,
  ElTooltip,
  ElTreeSelect,
} from 'element-plus';

import { UseUserFiltersKey } from '@/web/web/composables/search/useUserFilters';
import { flushPromises, fnRecorder, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import OSearch from './OSearch.vue';
import OSearchFilter from './OSearchFilter.vue';

const savedFiltersApi = {
  state: null as null | {
    favoriteMenuItems: any[];
    loading: boolean;
    loadError: string | null;
    defaultsForOpen: any[];
  },
  load: fnRecorder(async () => {}),
  apply: fnRecorder(),
  saveCurrent: fnRecorder(async () => ({ Id: '1' })),
  updateMeta: fnRecorder(async () => {}),
  remove: fnRecorder(async () => {}),
  lastCodeDefaults: undefined as unknown,
  lastScopeKey: undefined as unknown,
};

const msgWarning = fnRecorder();
const msgSuccess = fnRecorder();
const msgError = fnRecorder();
const boxConfirm = fnRecorder(async () => true);

const epSfcs = [
  ElButton,
  ElTag,
  ElTooltip,
  ElDialog,
  ElDivider,
  ElIcon,
  ElPopover,
  ElTreeSelect,
  ElForm,
  ElFormItem,
  ElInput,
  ElCheckbox,
];

function installEpStubs() {
  (ElMessage as any).warning = msgWarning;
  (ElMessage as any).success = msgSuccess;
  (ElMessage as any).error = msgError;
  (ElMessageBox as any).confirm = boxConfirm;

  stubSfc(ElButton as any, {
    name: 'ElButton',
    inheritAttrs: false,
    emits: ['click'],
    setup(_, { slots, emit, attrs }: any) {
      return () =>
        h(
          'button',
          {
            type: 'button',
            ...attrs,
            class: ['el-btn', attrs.class],
            onClick: (e: any) => emit('click', e),
          },
          slots.default?.()
        );
    },
  });
  stubSfc(ElTag as any, {
    name: 'ElTag',
    inheritAttrs: false,
    props: { closable: { type: Boolean, default: false } },
    emits: ['close', 'click'],
    setup(props: any, { slots, emit, attrs }: any) {
      return () =>
        h(
          'span',
          {
            ...attrs,
            class: ['el-tag', attrs.class],
            onClick: (e: any) => emit('click', e),
          },
          [
            slots.default?.(),
            props.closable
              ? h('button', {
                  type: 'button',
                  class: 'tag-close',
                  onClick: (e: any) => {
                    e.stopPropagation?.();
                    emit('close', e);
                  },
                })
              : null,
          ]
        );
    },
  });
  stubSfc(ElTooltip as any, {
    name: 'ElTooltip',
    setup(_, { slots }: any) {
      return () => h('div', {}, slots.default?.());
    },
  });
  stubSfc(ElPopover as any, {
    name: 'ElPopover',
    setup(_, { slots }: any) {
      return () =>
        h('div', { class: 'el-popover' }, [slots.reference?.(), h('div', { class: 'pop' }, slots.default?.())]);
    },
  });
  stubSfc(ElDialog as any, {
    name: 'ElDialog',
    props: ['modelValue', 'title'],
    setup(props: any, { slots }: any) {
      return () =>
        props.modelValue
          ? h('div', { class: 'el-dialog', 'data-title': props.title }, [slots.default?.(), slots.footer?.()])
          : null;
    },
  });
  stubSfc(ElDivider as any, { name: 'ElDivider', setup: () => () => h('hr') });
  stubSfc(ElIcon as any, {
    name: 'ElIcon',
    inheritAttrs: false,
    setup(_, { slots, attrs }: any) {
      return () => h('i', { ...attrs, class: attrs.class }, slots.default?.());
    },
  });
  stubSfc(ElForm as any, {
    name: 'ElForm',
    setup(_, { slots }: any) {
      return () => h('form', {}, slots.default?.());
    },
  });
  stubSfc(ElFormItem as any, {
    name: 'ElFormItem',
    setup(_, { slots }: any) {
      return () => h('div', { class: 'form-item' }, slots.default?.());
    },
  });
  stubSfc(ElInput as any, {
    name: 'ElInput',
    props: ['modelValue'],
    emits: ['update:modelValue'],
    setup(props: any, { emit }: any) {
      return () =>
        h('input', {
          class: 'fav-name',
          value: props.modelValue,
          onInput: (e: any) => emit('update:modelValue', e.target.value),
        });
    },
  });
  stubSfc(ElCheckbox as any, {
    name: 'ElCheckbox',
    props: ['modelValue'],
    emits: ['update:modelValue'],
    setup(props: any, { slots, emit }: any) {
      return () =>
        h('label', { class: 'fav-check' }, [
          h('input', {
            type: 'checkbox',
            checked: !!props.modelValue,
            onChange: (e: any) => emit('update:modelValue', !!(e.target as HTMLInputElement).checked),
          }),
          slots.default?.(),
        ]);
    },
  });
  stubSfc(ElTreeSelect as any, {
    name: 'ElTreeSelect',
    emits: ['change', 'update:modelValue'],
    setup(_, { emit }: any) {
      return () =>
        h('button', {
          type: 'button',
          class: 'tree',
          onClick: () => emit('change', 'f:Status'),
        });
    },
  });
  stubSfc(OSearchFilter as any, {
    name: 'OSearchFilter',
    props: ['store', 'draft', 'fields'],
    emits: ['cancel', 'confirm'],
    setup(_, { emit }: any) {
      return () =>
        h('div', { class: 'filter-editor' }, [
          h('button', { type: 'button', class: 'confirm', onClick: () => emit('confirm') }),
          h('button', { type: 'button', class: 'cancel', onClick: () => emit('cancel') }),
        ]);
    },
  });
}

function restoreEpStubs() {
  for (const Comp of epSfcs) restoreSfc(Comp as any);
  restoreSfc(OSearchFilter as any);
}

function makeUserFiltersFactory() {
  const prev = savedFiltersApi.state;
  savedFiltersApi.state = reactive({
    favoriteMenuItems: (prev?.favoriteMenuItems || []).slice(),
    loading: false,
    loadError: prev?.loadError ?? null,
    defaultsForOpen: (prev?.defaultsForOpen || [{ name: 'CodeDefault', query: ['A', '=', 1] }]).slice(),
  });
  return (params: {
    applyNamedFilter: (nf: any) => void;
    codeDefaults?: () => any;
    scopeKey?: () => string;
  }) => {
    savedFiltersApi.lastScopeKey = params.scopeKey?.();
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
      updateMeta: savedFiltersApi.updateMeta,
      remove: savedFiltersApi.remove,
    };
  };
}

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

function btnByText(m: { qa: (s: string) => Element[] }, substr: string) {
  return m.qa('.el-btn').find(b => (b.textContent || '').includes(substr));
}

function setInputValue(el: Element | null, value: string) {
  const input = el as HTMLInputElement;
  input.value = value;
  input.dispatchEvent(new Event('input', { bubbles: true }));
}

describe('OSearch behavior', () => {
  beforeEach(() => {
    const st = savedFiltersApi.state || {
      favoriteMenuItems: [],
      loading: false,
      loadError: null,
      defaultsForOpen: [],
    };
    savedFiltersApi.state = st as any;
    st.favoriteMenuItems = [];
    st.loading = false;
    st.loadError = null;
    st.defaultsForOpen = [{ name: 'CodeDefault', query: ['A', '=', 1] }];
    savedFiltersApi.load.mockReset();
    savedFiltersApi.apply.mockReset();
    savedFiltersApi.saveCurrent.mockReset();
    savedFiltersApi.updateMeta.mockReset();
    savedFiltersApi.remove.mockReset();
    savedFiltersApi.load.mockImplementation(async () => {});
    savedFiltersApi.saveCurrent.mockImplementation(async () => ({ Id: '1' }));
    savedFiltersApi.updateMeta.mockImplementation(async () => {});
    savedFiltersApi.remove.mockImplementation(async () => {});
    savedFiltersApi.lastScopeKey = undefined;
    savedFiltersApi.lastCodeDefaults = undefined;
    msgWarning.mockClear();
    msgSuccess.mockClear();
    msgError.mockClear();
    boxConfirm.mockReset();
    boxConfirm.mockImplementation(async () => true);
  });

  afterEach(() => {
    restoreEpStubs();
  });

  function mountSearch(props: Record<string, any> = {}) {
    installEpStubs();
    const emitted: Record<string, any[][]> = {};
    const track = (name: string) => (...args: any[]) => {
      (emitted[name] ||= []).push(args);
    };
    const m = mountApp(OSearch as any, {
      reactiveProps: true,
      props: {
        store: makeStore(),
        placeholder: 'Find…',
        defaultFilters: [{ name: 'Active', query: ['Active', '=', true] }],
        ...props,
      },
      on: {
        onQueryUpdate: track('query-update'),
        onDefaultsReady: track('defaults-ready'),
      },
      provide: {
        [UseUserFiltersKey as symbol]: makeUserFiltersFactory(),
      },
    });
    return { ...m, emitted };
  }

  test('wires empty scopeKey when route inject is unavailable', async () => {
    const m = mountSearch();
    await flushPromises();
    expect(savedFiltersApi.lastScopeKey).toBe('');
    m.unmount();
  });

  test('emits query-update on enter / search icon and syncs controlled keyword', async () => {
    const m = mountSearch({ currentKeyword: 'hello' });
    await flushPromises();
    const input = m.q('input.o-search__input') as HTMLInputElement;
    expect(input.value).toBe('hello');

    m.setupState().onEnter();
    expect((m.emitted['query-update'] || []).length).toBeGreaterThan(0);

    const before = (m.emitted['query-update'] || []).length;
    m.qa('.el-btn')[0]!.dispatchEvent(new Event('click', { bubbles: true }));
    await nextTick();
    expect((m.emitted['query-update'] || []).length).toBeGreaterThan(before);
    m.unmount();
  });

  test('shows grouping tag and clears grouping', async () => {
    const m = mountSearch({
      currentAppliedGroups: [{ field: 'Status' }, { field: 'CreatedAt', granularity: 'month' }],
    });
    await nextTick();
    expect(m.q('.o-search__grouptag')).toBeTruthy();
    (m.q('.tag-close') as HTMLElement).click();
    await flushPromises();
    const payload = m.emitted['query-update']!.at(-1)![0] as any;
    expect(payload.appliedGroups).toEqual([]);
    m.unmount();
  });

  test('opens custom filter editor, warns on incomplete save, and cancels', async () => {
    const m = mountSearch();
    await nextTick();
    const custom = btnByText(m, 'Custom filter');
    expect(custom).toBeTruthy();
    (custom as HTMLElement).click();
    await nextTick();
    expect(m.q('.filter-editor')).toBeTruthy();

    (m.q('.confirm') as HTMLElement).click();
    await nextTick();
    expect(msgWarning.calls.length).toBeGreaterThan(0);

    (m.q('.cancel') as HTMLElement).click();
    await nextTick();
    expect(m.q('.filter-editor')).toBeFalsy();
    m.unmount();
  });

  test('applies controlled filters and supports tag close / backspace delete', async () => {
    const filters = [
      {
        id: 'f1',
        name: 'Active',
        logic: 'And' as const,
        children: [{ id: 'c1', field: 'Active', operator: '=', value: true }],
      },
    ];
    const m = mountSearch({ currentAppliedFilters: filters });
    await flushPromises();
    expect(m.q('.o-search__tag')).toBeTruthy();

    (m.q('.o-search__tag') as HTMLElement).click();
    await nextTick();
    expect(m.q('.filter-editor')).toBeTruthy();
    (m.q('.cancel') as HTMLElement).click();
    await nextTick();

    (m.q('.tag-close') as HTMLElement).click();
    expect((m.emitted['query-update'] || []).length).toBeGreaterThan(0);

    m.props.currentAppliedFilters = [];
    await flushPromises();
    m.props.currentAppliedFilters = [
      {
        id: 'f2',
        name: 'X',
        logic: 'And',
        children: [{ id: 'c2', field: 'Name', operator: '=', value: 'a' }],
      },
    ];
    await flushPromises();
    expect(m.q('.o-search__tag')?.textContent || '').toContain('X');
    (m.q('.o-search__tag') as HTMLElement).click();
    await nextTick();
    expect(m.q('.filter-editor')).toBeTruthy();
    (m.q('.cancel') as HTMLElement).click();
    m.unmount();
  });

  test('toggles default filter from menu', async () => {
    const m = mountSearch();
    await nextTick();
    const item = btnByText(m, 'Active');
    expect(item).toBeTruthy();
    (item as HTMLElement).click();
    await flushPromises();
    expect((m.emitted['query-update'] || []).length).toBeGreaterThan(0);
    m.unmount();
  });

  test('applies tree select change for grouping', async () => {
    const m = mountSearch();
    await nextTick();
    (m.q('.tree') as HTMLElement).click();
    await flushPromises();
    expect((m.emitted['query-update'] || []).length).toBeGreaterThan(0);
    const payload = m.emitted['query-update']!.at(-1)![0] as any;
    expect(payload.appliedGroups?.some((g: any) => g.field === 'Status' || g === 'Status')).toBe(true);
    m.unmount();
  });

  test('saves a complete edited draft and closes the editor', async () => {
    const m = mountSearch({
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
    (m.q('.o-search__tag') as HTMLElement).click();
    await nextTick();
    expect(m.q('.filter-editor')).toBeTruthy();
    const before = (m.emitted['query-update'] || []).length;
    (m.q('.confirm') as HTMLElement).click();
    await flushPromises();
    expect(m.q('.filter-editor')).toBeFalsy();
    expect((m.emitted['query-update'] || []).length).toBeGreaterThan(before);
    m.unmount();
  });

  test('closes quietly when edited filter disappears before save', async () => {
    msgWarning.mockClear();
    const m = mountSearch({
      currentAppliedFilters: [
        {
          id: 'f-gone',
          logic: 'And',
          children: [{ id: 'c1', field: 'Name', operator: '=', value: 'a' }],
        },
      ],
    });
    await flushPromises();
    (m.q('.o-search__tag') as HTMLElement).click();
    await nextTick();
    expect(m.q('.filter-editor')).toBeTruthy();

    m.props.currentAppliedFilters = [];
    await flushPromises();
    (m.q('.confirm') as HTMLElement).click();
    await flushPromises();
    expect(m.q('.filter-editor')).toBeFalsy();
    expect(msgWarning.calls.length).toBe(0);
    m.unmount();
  });

  test('opens grouping menu from group tag and toggles applied menu items', async () => {
    const m = mountSearch({
      currentAppliedGroups: [{ field: 'Status' }, { field: 'CreatedAt', granularity: 'month' }],
    });
    await nextTick();
    (m.q('.o-search__grouptag') as HTMLElement).click();
    await nextTick();
    const statusItem = btnByText(m, 'Status');
    expect(statusItem).toBeTruthy();
    (statusItem as HTMLElement).click();
    await flushPromises();
    expect((m.emitted['query-update'] || []).length).toBeGreaterThan(0);
    m.unmount();
  });

  test('pending-deletes last filter tag via backspace then removes it', async () => {
    const m = mountSearch({
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
    const fakeEvt = {
      key: 'Backspace',
      target: { selectionStart: 0, selectionEnd: 0 },
      preventDefault() {},
    };
    m.setupState().onInputKeydown(fakeEvt);
    await nextTick();
    expect(m.q('.o-search__tag--pending-delete')).toBeTruthy();

    m.setupState().onInputKeydown(fakeEvt);
    await flushPromises();
    expect(m.q('.o-search__tag')).toBeFalsy();

    m.props.currentAppliedFilters = [
      {
        id: 'f2',
        name: 'Two',
        logic: 'And',
        children: [{ id: 'c2', field: 'Name', operator: '=', value: 'b' }],
      },
    ];
    await flushPromises();
    m.setupState().onInputKeydown(fakeEvt);
    await nextTick();
    m.setupState().onInputKeydown({ key: 'a', target: { selectionStart: 0, selectionEnd: 0 }, preventDefault() {} });
    await nextTick();
    expect(m.q('.o-search__tag--pending-delete')).toBeFalsy();
    m.unmount();
  });

  test('focuses the keyword input when the shell is clicked', async () => {
    const m = mountSearch();
    const input = m.q('input.o-search__input') as HTMLInputElement;
    const focus = fnRecorder();
    input.focus = focus as any;
    (m.q('.o-search__main') as HTMLElement).click();
    expect(focus.calls.length).toBeGreaterThan(0);
    m.unmount();
  });

  test('loads favorites on mount and emits defaults-ready for OSearchView first frame', async () => {
    const m = mountSearch();
    await flushPromises();
    expect(savedFiltersApi.load.calls.length).toBeGreaterThan(0);
    expect(m.emitted['defaults-ready']?.[0]?.[0]).toEqual([{ name: 'CodeDefault', query: ['A', '=', 1] }]);
    expect(m.text()).toContain('No favorites yet');

    savedFiltersApi.state!.loadError = 'boom';
    await nextTick();
    expect(m.text()).toContain('Failed to load favorites');
    const before = savedFiltersApi.load.calls.length;
    const retry = btnByText(m, 'Retry');
    expect(retry).toBeTruthy();
    (retry as HTMLElement).click();
    expect(savedFiltersApi.load.calls.length).toBeGreaterThan(before);
    m.unmount();
  });

  test('covers codeDefaults singleton/undefined branches', async () => {
    const a = mountSearch({ defaultFilters: undefined as any });
    await flushPromises();
    expect(savedFiltersApi.lastCodeDefaults).toBeUndefined();
    a.unmount();

    const b = mountSearch({ defaultFilters: { name: 'Solo', query: ['X', '=', 1] } as any });
    await flushPromises();
    expect(savedFiltersApi.lastCodeDefaults).toEqual([{ name: 'Solo', query: ['X', '=', 1] }]);
    b.unmount();
  });

  test('shows Check icon for applied favorite names', async () => {
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
    const m = mountSearch({
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
    expect(m.q('.o-search__menu-icon--applied')).toBeTruthy();
    expect(m.text()).toContain('Shared');
    m.unmount();
  });

  test('applies and removes favorites (confirm), and saves with empty-name warning', async () => {
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
    const m = mountSearch();
    await flushPromises();

    const applyBtn = btnByText(m, 'Mine');
    expect(applyBtn).toBeTruthy();
    const beforeEmit = (m.emitted['query-update'] || []).length;
    (applyBtn as HTMLElement).click();
    await flushPromises();
    expect(savedFiltersApi.apply.calls[0]![0]).toMatchObject({
      name: 'Mine',
      filter: { And: [['Active', '=', true]] },
    });
    expect((m.emitted['query-update'] || []).length).toBeGreaterThan(beforeEmit);

    (m.q('.o-search__menu-item-delete') as HTMLElement).click();
    await flushPromises();
    expect(boxConfirm.calls.length).toBeGreaterThan(0);
    expect(savedFiltersApi.remove.calls[0]).toEqual(['fav-1']);
    expect(msgSuccess.calls.length).toBeGreaterThan(0);

    const saveOpen = btnByText(m, 'Save current filters');
    (saveOpen as HTMLElement).click();
    await nextTick();
    expect(m.q('.el-dialog')).toBeTruthy();

    const saveBtn = m.qa('.el-btn').find(b => (b.textContent || '').trim() === 'Save');
    setInputValue(m.q('input.fav-name'), '');
    await nextTick();
    (saveBtn as HTMLElement).click();
    await nextTick();
    expect(msgWarning.calls.length).toBeGreaterThan(0);
    expect(savedFiltersApi.saveCurrent.calls.length).toBe(0);

    setInputValue(m.q('input.fav-name'), 'NewFav');
    await nextTick();
    (saveBtn as HTMLElement).click();
    await flushPromises();
    expect(savedFiltersApi.saveCurrent.calls[0]![0]).toEqual({
      name: 'NewFav',
      isDefault: false,
      shared: false,
    });
    expect((m.emitted['defaults-ready'] || []).length).toBe(3);
    m.unmount();
  });

  test('cancels favorite delete when ElMessageBox rejects', async () => {
    boxConfirm.mockImplementation(async () => {
      throw 'cancel';
    });
    msgError.mockClear();
    msgSuccess.mockClear();
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
    const m = mountSearch();
    await flushPromises();
    (m.q('.o-search__menu-item-delete') as HTMLElement).click();
    await flushPromises();
    expect(savedFiltersApi.remove.calls.length).toBe(0);
    expect(msgSuccess.calls.length).toBe(0);
    expect(msgError.calls.length).toBe(0);
    m.unmount();
  });

  test('shows ElMessage.error when remove or save fails', async () => {
    boxConfirm.mockImplementation(async () => true);
    msgError.mockClear();
    savedFiltersApi.remove.mockImplementation(async () => {
      throw new Error('delete failed');
    });
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
    const m = mountSearch();
    await flushPromises();
    (m.q('.o-search__menu-item-delete') as HTMLElement).click();
    await flushPromises();
    expect(msgError.calls[0]).toEqual(['delete failed']);

    msgError.mockClear();
    savedFiltersApi.remove.mockImplementation(async () => {});
    savedFiltersApi.saveCurrent.mockImplementation(async () => {
      throw 'save blew up';
    });
    const saveOpen = btnByText(m, 'Save current filters');
    (saveOpen as HTMLElement).click();
    await nextTick();
    setInputValue(m.q('input.fav-name'), 'FailFav');
    await nextTick();
    const saveBtn = m.qa('.el-btn').find(b => (b.textContent || '').trim() === 'Save');
    (saveBtn as HTMLElement).click();
    await flushPromises();
    expect(msgError.calls[0]).toEqual(['save blew up']);
    m.unmount();
  });

  test('stringifies non-Error remove failures and Error save failures', async () => {
    boxConfirm.mockImplementation(async () => true);
    msgError.mockClear();
    savedFiltersApi.remove.mockImplementation(async () => {
      throw 'delete-string';
    });
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
    const m = mountSearch();
    await flushPromises();
    (m.q('.o-search__menu-item-delete') as HTMLElement).click();
    await flushPromises();
    expect(msgError.calls[0]).toEqual(['delete-string']);

    msgError.mockClear();
    savedFiltersApi.remove.mockImplementation(async () => {});
    savedFiltersApi.saveCurrent.mockImplementation(async () => {
      throw new Error('save failed');
    });
    const saveOpen = btnByText(m, 'Save current filters');
    (saveOpen as HTMLElement).click();
    await nextTick();
    setInputValue(m.q('input.fav-name'), 'ErrFav');
    await nextTick();
    const saveBtn = m.qa('.el-btn').find(b => (b.textContent || '').trim() === 'Save');
    (saveBtn as HTMLElement).click();
    await flushPromises();
    expect(msgError.calls[0]).toEqual(['save failed']);
    m.unmount();
  });

  test('guards re-entrant save while saveFavoriteSaving is true', async () => {
    let resolveSave!: (v: any) => void;
    savedFiltersApi.saveCurrent.mockImplementation(
      () =>
        new Promise(resolve => {
          resolveSave = resolve;
        })
    );
    const m = mountSearch();
    await flushPromises();
    const saveOpen = btnByText(m, 'Save current filters');
    (saveOpen as HTMLElement).click();
    await nextTick();
    setInputValue(m.q('input.fav-name'), 'Once');
    await nextTick();
    const saveBtn = m.qa('.el-btn').find(b => (b.textContent || '').trim() === 'Save') as HTMLElement;
    saveBtn.click();
    saveBtn.click();
    await nextTick();
    expect(savedFiltersApi.saveCurrent.calls.length).toBe(1);
    resolveSave!({ Id: '1' });
    await flushPromises();
    m.unmount();
  });

  test('does not emit query-update when applying a favorite leaves filter length unchanged', async () => {
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
    const m = mountSearch();
    await flushPromises();
    // Override after mount: factory wires applyNamedFilter during setup.
    savedFiltersApi.apply.mockImplementation(() => {
      /* no-op: filters length stays the same */
    });
    const beforeEmit = (m.emitted['query-update'] || []).length;
    const applyBtn = btnByText(m, 'Noop');
    (applyBtn as HTMLElement).click();
    await flushPromises();
    expect(savedFiltersApi.apply.calls.length).toBeGreaterThan(0);
    expect((m.emitted['query-update'] || []).length).toBe(beforeEmit);
    m.unmount();
  });

  test('saves with isDefault and shared checkboxes enabled', async () => {
    const m = mountSearch();
    await flushPromises();
    const saveOpen = btnByText(m, 'Save current filters');
    (saveOpen as HTMLElement).click();
    await nextTick();
    setInputValue(m.q('input.fav-name'), 'DefaultOnly');
    await nextTick();
    const defaultCheck = m.qa('.fav-check').find(l => (l.textContent || '').includes('Use by default'));
    expect(defaultCheck).toBeTruthy();
    const defaultInput = defaultCheck!.querySelector('input') as HTMLInputElement;
    defaultInput.checked = true;
    defaultInput.dispatchEvent(new Event('change', { bubbles: true }));
    await nextTick();
    const saveBtn = m.qa('.el-btn').find(b => (b.textContent || '').trim() === 'Save') as HTMLElement;
    saveBtn.click();
    await flushPromises();
    expect(savedFiltersApi.saveCurrent.calls[0]![0]).toEqual({
      name: 'DefaultOnly',
      isDefault: true,
      shared: false,
    });

    savedFiltersApi.saveCurrent.mockClear();
    (saveOpen as HTMLElement).click();
    await nextTick();
    setInputValue(m.q('input.fav-name'), 'SharedOnly');
    await nextTick();
    const sharedCheck = m.qa('.fav-check').find(l => (l.textContent || '').includes('Share with all users'));
    expect(sharedCheck).toBeTruthy();
    const sharedInput = sharedCheck!.querySelector('input') as HTMLInputElement;
    sharedInput.checked = true;
    sharedInput.dispatchEvent(new Event('change', { bubbles: true }));
    await nextTick();
    (m.qa('.el-btn').find(b => (b.textContent || '').trim() === 'Save') as HTMLElement).click();
    await flushPromises();
    expect(savedFiltersApi.saveCurrent.calls[0]![0]).toEqual({
      name: 'SharedOnly',
      isDefault: false,
      shared: true,
    });
    m.unmount();
  });

  test('edits favorite metadata via dialog without create', async () => {
    savedFiltersApi.state!.favoriteMenuItems = [
      {
        id: 'fav-edit',
        name: 'Company Management',
        shared: true,
        isDefault: true,
        canDelete: true,
        filter: { And: [['X', '=', 1]] },
      },
      {
        id: 'fav-readonly',
        name: 'Others',
        shared: true,
        isDefault: false,
        canDelete: false,
        filter: {},
      },
    ];
    const m = mountSearch();
    await flushPromises();
    expect(m.qa('.o-search__menu-item-edit').length).toBe(1);
    expect(m.qa('.o-search__menu-item-delete').length).toBe(1);

    (m.q('.o-search__menu-item-edit') as HTMLElement).click();
    await nextTick();
    const dialog = m.q('.el-dialog');
    expect(dialog).toBeTruthy();
    expect(dialog!.getAttribute('data-title')).toBe('Edit favorite');
    expect((m.q('input.fav-name') as HTMLInputElement).value).toBe('Company Management');
    const defaultCheck = m.qa('.fav-check').find(l => (l.textContent || '').includes('Use by default'));
    const sharedCheck = m.qa('.fav-check').find(l => (l.textContent || '').includes('Share with all users'));
    expect((defaultCheck!.querySelector('input') as HTMLInputElement).checked).toBe(true);
    expect((sharedCheck!.querySelector('input') as HTMLInputElement).checked).toBe(true);

    setInputValue(m.q('input.fav-name'), 'Renamed Fav');
    await nextTick();
    const sharedInput = sharedCheck!.querySelector('input') as HTMLInputElement;
    sharedInput.checked = false;
    sharedInput.dispatchEvent(new Event('change', { bubbles: true }));
    await nextTick();
    const beforeReady = (m.emitted['defaults-ready'] || []).length;
    (m.qa('.el-btn').find(b => (b.textContent || '').trim() === 'Save') as HTMLElement).click();
    await flushPromises();
    expect(savedFiltersApi.updateMeta.calls[0]).toEqual([
      'fav-edit',
      { name: 'Renamed Fav', isDefault: true, shared: false },
    ]);
    expect(savedFiltersApi.saveCurrent.calls.length).toBe(0);
    expect(msgSuccess.calls[0]).toEqual(['Favorite updated']);
    expect((m.emitted['defaults-ready'] || []).length).toBe(beforeReady + 1);
    m.unmount();
  });
});

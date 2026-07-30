// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h, inject, nextTick, reactive, ref } from 'vue';
import { mount, flushPromises } from '@vue/test-utils';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import OListView from '@/web/web/components/view/OListView.vue';
import { LIST_HANDLE_API_KEY } from '@/web/web/composables/useListHandleReorder';

const visibleNodes = ref<any[]>([
  { kind: 'record', key: '1', payload: { Id: '1', Name: 'A', Sequence: 1 } },
  { kind: 'record', key: '2', payload: { Id: '2', Name: 'B', Sequence: 2 } },
]);
const resultKind = ref<'search' | 'group'>('search');
const applyMock = vi.fn(async () => {});
const expandGroupMock = vi.fn();
const loadMoreGroupChildrenMock = vi.fn(async () => {});
const loadMoreGroupRecordsMock = vi.fn(async () => {});
let lastRowClickPayload: any = null;

vi.mock('vue-router', () => ({ useRouter: () => ({ push: vi.fn() }) }));

vi.mock('@/web/web/controllers/listController', () => ({
  createListController: vi.fn(() => ({
    vm: reactive({
      get visibleNodes() {
        return visibleNodes.value;
      },
      get result() {
        return { kind: resultKind.value, total: 2 };
      },
      expandedKeys: new Set(),
    }),
    apply: applyMock,
    paginate: vi.fn(async () => {}),
    sort: vi.fn(async () => {}),
    expandGroup: expandGroupMock,
    loadMoreGroupChildren: loadMoreGroupChildrenMock,
    loadMoreGroupRecords: loadMoreGroupRecordsMock,
  })),
}));

vi.mock('@/web/web/composables/useAdaptiveHeight', () => ({
  useAdaptiveHeight: () => ({
    height: ref(400),
    pxHeight: ref('400px'),
    recompute: vi.fn(),
  }),
}));

vi.mock('@/web/web/query/utils/registry/fieldReady', () => ({
  awaitFieldSelection: vi.fn(async () => {}),
}));

vi.mock('@/web/web/i18n', async () => {
  const actual = await vi.importActual<typeof import('@/web/web/i18n')>('@/web/web/i18n');
  return {
    ...actual,
    createTranslate: () => ({ _t: (msg: string) => msg }),
  };
});

vi.mock('element-plus', async () => {
  const actual = await vi.importActual<any>('element-plus');
  return {
    ...actual,
    ElMessage: { success: vi.fn(), error: vi.fn(), warning: vi.fn() },
    ElMessageBox: { confirm: vi.fn() },
  };
});

vi.mock('@/web/web/composables/useOnchange', () => {
  const ctrl = {
    flush: vi.fn(async () => {}),
    reset: vi.fn(),
    pause: vi.fn(),
    force: vi.fn(),
    running: ref(false),
  };
  return {
    provideOnchange: vi.fn(() => ctrl),
    useProvidedOnchange: vi.fn(() => ctrl),
  };
});

const OVTableStub = defineComponent({
  name: 'OVTableStub',
  emits: ['row-click', 'selection-change', 'sort-change'],
  setup(_, { slots, emit }) {
    const formRoot = inject<any>('form-root', null);
    const viewMode = inject<any>('view-mode', null);
    const handleApi = inject<any>(LIST_HANDLE_API_KEY, null);
    (globalThis as any).__listTableFormRoot = formRoot;
    (globalThis as any).__listTableViewMode = viewMode;
    return () =>
      h('div', { class: 'ov-table-stub' }, [
        h(
          'button',
          {
            class: 'row-click-1',
            onClick: () =>
              emit('row-click', {
                rowData: visibleNodes.value[0],
                rowIndex: 0,
                rowKey: '1',
                event: new MouseEvent('click'),
              }),
          },
          'row1'
        ),
        h(
          'button',
          {
            class: 'row-click-custom',
            onClick: () => {
              if (!lastRowClickPayload) return;
              emit('row-click', lastRowClickPayload);
            },
          },
          'custom'
        ),
        h(
          'button',
          {
            class: 'mutate-draft',
            onClick: () => formRoot?.setField('Name', 'Edited'),
          },
          'mutate'
        ),
        h(
          'button',
          {
            class: 'reorder-handle',
            onClick: async () => {
              handleApi?.onDragStart(0, {
                preventDefault: vi.fn(),
                dataTransfer: { effectAllowed: '', setData: vi.fn() },
              });
              await handleApi?.onDrop(1, { preventDefault: vi.fn() });
            },
          },
          'reorder'
        ),
        slots.default?.(),
        h('div', { class: 'empty-slot' }, slots.empty?.()),
      ]);
  },
});

const OVColumnStub = defineComponent({
  name: 'OVColumnStub',
  props: { type: String, colKey: String },
  setup(props, { slots }) {
    return () => {
      const kids =
        props.colKey === '__group_label'
          ? [
              slots.default?.({ row: { kind: 'group', key: 'g1', depth: 1, label: 'Group', count: 3 } }),
              slots.default?.({ row: { kind: 'group', key: 'g2', depth: 0, label: 'NoCount' } }),
              slots.default?.({
                row: { kind: 'more', key: 'more:g1', groupKey: 'g1', remain: 2, target: 'records' },
              }),
              slots.default?.({
                row: { kind: 'more', key: 'more:g2', groupKey: 'g2', target: 'groups' },
              }),
              slots.default?.({ row: { kind: 'record', key: '1', payload: { Id: '1' } } }),
            ]
          : [];
      return h('div', { class: 'ov-column-stub', 'data-type': props.type, 'data-key': props.colKey }, kids);
    };
  },
});

function makeStore() {
  return {
    state: { queryState: { pagination: { limit: 20, offset: 0 } }, result: { total: 2 } },
    fieldsMetadata: {
      Name: { id: '1', type: 'varchar', typeAnnotation: '' },
      Sequence: { id: '2', type: 'int', typeAnnotation: '', isReadonly: false },
    },
    UpdateById: vi.fn(async () => ({})),
    Delete: vi.fn(async () => ({})),
  };
}

async function mountList(opts?: {
  editable?: boolean;
  showHandle?: boolean;
  handleField?: string;
  offset?: number | null;
  clickToSelect?: boolean;
  searchView?: any;
  orderBy?: any;
}) {
  const store = makeStore();
  if (opts && 'offset' in (opts as object)) {
    (store.state.queryState.pagination as any).offset = opts.offset;
  }
  if (opts && 'orderBy' in (opts as object)) {
    (store.state.queryState as any).orderBy = opts.orderBy;
  }
  const wrapper = mount(OListView, {
    props: {
      store: store as any,
      editable: opts?.editable ?? true,
      showHandle: opts?.showHandle ?? true,
      ...(opts?.handleField != null ? { handleField: opts.handleField } : {}),
      clickToSelect: opts?.clickToSelect ?? false,
      showPaginate: false,
      refreshAction: false,
      deleteAction: false,
      ...(opts?.searchView != null ? { searchView: opts.searchView } : {}),
    },
    global: {
      stubs: {
        OViewContainer: { template: '<div><slot name="header" /><slot /></div>' },
        OVTable: OVTableStub,
        OVColumn: OVColumnStub,
        OPagination: true,
        ElButton: { template: '<button v-bind="$attrs" @click="$attrs.onClick"><slot /></button>' },
        ElIcon: true,
      },
    },
  });
  await flushPromises();
  await nextTick();
  mountedWrappers.push(wrapper);
  return { wrapper, store };
}

const mountedWrappers: ReturnType<typeof mount>[] = [];

describe('OListView editable / handle', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    resultKind.value = 'search';
    lastRowClickPayload = null;
    (globalThis as any).__listTableFormRoot = undefined;
    (globalThis as any).__listTableViewMode = undefined;
    visibleNodes.value = [
      { kind: 'record', key: '1', payload: { Id: '1', Name: 'A', Sequence: 1 } },
      { kind: 'record', key: '2', payload: { Id: '2', Name: 'B', Sequence: 2 } },
    ];
  });

  afterEach(() => {
    while (mountedWrappers.length) {
      mountedWrappers.pop()!.unmount();
    }
  });

  it('hides handle when showHandle is false or Sequence metadata is missing', async () => {
    const hidden = await mountList({ showHandle: false });
    expect(hidden.wrapper.find('.ov-column-stub[data-type="handle"]').exists()).toBe(false);

    const store = makeStore();
    delete (store.fieldsMetadata as any).Sequence;
    const remount = mount(OListView, {
      props: {
        store: store as any,
        editable: true,
        showHandle: true,
        showPaginate: false,
        refreshAction: false,
        deleteAction: false,
      },
      global: {
        stubs: {
          OViewContainer: { template: '<div><slot name="header" /><slot /></div>' },
          OVTable: OVTableStub,
          OVColumn: OVColumnStub,
          OPagination: true,
          ElButton: { template: '<button v-bind="$attrs" @click="$attrs.onClick"><slot /></button>' },
          ElIcon: true,
        },
      },
    });
    await flushPromises();
    expect(remount.find('.ov-column-stub[data-type="handle"]').exists()).toBe(false);
    remount.unmount();
  });

  it('hides handle column in group mode and renders group/more cells', async () => {
    resultKind.value = 'group';
    visibleNodes.value = [
      { kind: 'group', key: 'g1', depth: 0, label: 'G', count: 1 },
      { kind: 'record', key: '1', payload: { Id: '1', Name: 'A', Sequence: 1 } },
    ];
    const { wrapper } = await mountList();
    expect(wrapper.find('.ov-column-stub[data-type="handle"]').exists()).toBe(false);
    expect(wrapper.find('.o-group-cell').exists()).toBe(true);
    expect(wrapper.find('.o-more-cell').exists()).toBe(true);
    expect(wrapper.find('.ovtable__empty').exists()).toBe(true);
    await wrapper.find('.o-group-cell__caret').trigger('click');
    expect(expandGroupMock).toHaveBeenCalledWith('g1', true);
  });

  it('enters inline edit on row click and shows save/discard actions', async () => {
    const { wrapper } = await mountList();
    expect(wrapper.find('.ov-column-stub[data-type="handle"]').exists()).toBe(true);
    await wrapper.find('.row-click-1').trigger('click');
    await flushPromises();
    expect(wrapper.text()).toContain('Save');
    expect(wrapper.text()).toContain('Discard');
  });

  it('save and discard inline edit via header buttons', async () => {
    const { wrapper, store } = await mountList();
    await wrapper.find('.row-click-1').trigger('click');
    await flushPromises();
    await wrapper.find('.mutate-draft').trigger('click');
    await wrapper.find('button[type="success"]').trigger('click');
    await flushPromises();
    expect(store.UpdateById).toHaveBeenCalledWith('1', { Name: 'Edited' });

    await wrapper.find('.row-click-1').trigger('click');
    await flushPromises();
    await wrapper.find('.mutate-draft').trigger('click');
    const discardBtn = wrapper.findAll('button').find(b => b.text() === 'Discard');
    await discardBtn!.trigger('click');
    await flushPromises();
    expect(wrapper.text()).not.toContain('Discard');
  });

  it('discards edit when edited row leaves visible items', async () => {
    const { ElMessage } = await import('element-plus');
    const { wrapper } = await mountList();
    await wrapper.find('.row-click-1').trigger('click');
    await flushPromises();
    await wrapper.find('.mutate-draft').trigger('click');

    visibleNodes.value = [{ kind: 'record', key: '2', payload: { Id: '2', Name: 'B', Sequence: 2 } }];
    await nextTick();
    await flushPromises();

    expect(ElMessage.warning).toHaveBeenCalled();
    expect(wrapper.text()).not.toContain('Discard');
  });

  it('persists handle reorder via provided api', async () => {
    const { wrapper, store } = await mountList();
    await wrapper.find('.reorder-handle').trigger('click');
    await flushPromises();
    expect(store.UpdateById).toHaveBeenCalled();
    expect(applyMock).toHaveBeenCalled();
  });

  it('rolls back handle reorder when UpdateById fails', async () => {
    const { ElMessage } = await import('element-plus');
    const { wrapper, store } = await mountList();
    store.UpdateById = vi.fn(async () => {
      throw new Error('fail');
    });
    await wrapper.find('.reorder-handle').trigger('click');
    await flushPromises();
    expect(ElMessage.error).toHaveBeenCalledWith('Failed to reorder rows');
  });

  it('swallows save errors in handleInlineSave', async () => {
    const { ElMessage } = await import('element-plus');
    const { wrapper, store } = await mountList();
    await wrapper.find('.row-click-1').trigger('click');
    await flushPromises();
    await wrapper.find('.mutate-draft').trigger('click');
    store.UpdateById = vi.fn(async () => {
      throw new Error('save fail');
    });
    await wrapper.find('button[type="success"]').trigger('click');
    await flushPromises();
    expect((wrapper.vm as any).inlineEdit.isEditing.value).toBe(true);
    expect(ElMessage.error).toHaveBeenCalledWith('Failed to save row');
  });

  it('hides handle column when not editable', async () => {
    const { wrapper } = await mountList({ editable: false });
    expect(wrapper.find('.ov-column-stub[data-type="handle"]').exists()).toBe(false);
  });

  it('enters inline edit from grouped record row click', async () => {
    resultKind.value = 'group';
    visibleNodes.value = [
      { kind: 'group', key: 'g1' },
      { kind: 'record', key: '1', payload: { Id: '1', Name: 'A', Sequence: 1 } },
    ];
    const { wrapper } = await mountList();
    lastRowClickPayload = {
      rowData: visibleNodes.value[1],
      rowIndex: 1,
      rowKey: '1',
      event: new MouseEvent('click'),
    };
    await wrapper.find('.row-click-custom').trigger('click');
    await flushPromises();
    expect((wrapper.vm as any).inlineEdit.isEditing.value).toBe(true);
    expect((wrapper.vm as any).inlineEdit.editingRowId.value).toBe('1');
  });

  it('skips enterEdit when editable but row has no id in group mode', async () => {
    resultKind.value = 'group';
    const { wrapper } = await mountList();
    lastRowClickPayload = {
      rowData: { kind: 'record', key: 'x', payload: { Name: 'no-id' } },
      rowIndex: 0,
      rowKey: 'x',
      event: new MouseEvent('click'),
    };
    await wrapper.find('.row-click-custom').trigger('click');
    await flushPromises();
    expect((wrapper.vm as any).inlineEdit.isEditing.value).toBe(false);
  });

  it('does not enter edit on group record click when not editable', async () => {
    resultKind.value = 'group';
    const { wrapper } = await mountList({ editable: false });
    lastRowClickPayload = {
      rowData: { kind: 'record', key: '1', payload: { Id: '1', Name: 'A' } },
      rowIndex: 0,
      rowKey: '1',
      event: new MouseEvent('click'),
    };
    await wrapper.find('.row-click-custom').trigger('click');
    await flushPromises();
    expect((wrapper.vm as any).inlineEdit.isEditing.value).toBe(false);
  });

  it('does not enter edit on plain row click when not editable', async () => {
    const { wrapper } = await mountList({ editable: false });
    await wrapper.find('.row-click-1').trigger('click');
    await flushPromises();
    expect((wrapper.vm as any).inlineEdit.isEditing.value).toBe(false);
  });

  it('does not enter edit when record row has no id in flat mode', async () => {
    const { wrapper } = await mountList();
    lastRowClickPayload = {
      rowData: { kind: 'record', key: 'x', payload: { Name: 'no-id' } },
      rowIndex: 0,
      rowKey: 'x',
      event: new MouseEvent('click'),
    };
    await wrapper.find('.row-click-custom').trigger('click');
    await flushPromises();
    expect((wrapper.vm as any).inlineEdit.isEditing.value).toBe(false);
  });

  it('skips persist when handle reorder yields no writes', async () => {
    const persist = await import('@/web/web/composables/listViewHandlePersist');
    const spy = vi.spyOn(persist, 'buildHandleReorderWrites').mockReturnValue([]);
    try {
      const { wrapper, store } = await mountList();
      // Exposed refs are auto-unwrapped on the public instance.
      const flat = () => (wrapper.vm as any).flatRows as any[];
      const before = flat().map((r: any) => r.payload.Id);
      store.UpdateById.mockClear();
      applyMock.mockClear();
      await wrapper.find('.reorder-handle').trigger('click');
      await flushPromises();
      expect(store.UpdateById).not.toHaveBeenCalled();
      expect(applyMock).not.toHaveBeenCalled();
      expect(flat().map((r: any) => r.payload.Id)).toEqual(before);
    } finally {
      spy.mockRestore();
    }
  });

  it('uses pagination offset fallback when offset is null', async () => {
    const { wrapper, store } = await mountList({ offset: null });
    await wrapper.find('.reorder-handle').trigger('click');
    await flushPromises();
    expect(store.UpdateById).toHaveBeenCalled();
  });

  it('enters inline edit even when clickToSelect is enabled', async () => {
    const { wrapper } = await mountList({ clickToSelect: true });
    await wrapper.find('.row-click-1').trigger('click');
    await flushPromises();
    expect((wrapper.vm as any).inlineEdit.isEditing.value).toBe(true);
    expect((wrapper.vm as any).inlineEdit.editingRowId.value).toBe('1');
  });

  it('scopes form-root and edit view-mode to the table, not the header search', async () => {
    let headerFormRoot: any = 'unset';
    let headerViewMode: any = 'unset';
    const HeaderSearchProbe = defineComponent({
      setup() {
        headerFormRoot = inject('form-root', null);
        headerViewMode = inject('view-mode', null);
        return () => h('span', { class: 'header-search-probe' });
      },
    });

    const { wrapper } = await mountList({ searchView: HeaderSearchProbe });
    expect(wrapper.find('.header-search-probe').exists()).toBe(true);
    expect(headerFormRoot).toBeNull();
    expect(headerViewMode?.value ?? headerViewMode).toBe('display');

    expect((globalThis as any).__listTableFormRoot).toBeTruthy();
    expect((globalThis as any).__listTableViewMode?.value ?? (globalThis as any).__listTableViewMode).toBe('display');

    await wrapper.find('.row-click-1').trigger('click');
    await flushPromises();

    // Header remains isolated while the table enters edit with a live draft.
    expect(headerFormRoot).toBeNull();
    expect(headerViewMode?.value ?? headerViewMode).toBe('display');
    expect((globalThis as any).__listTableViewMode?.value ?? (globalThis as any).__listTableViewMode).toBe('edit');
    expect((globalThis as any).__listTableFormRoot?.draft?.Id).toBe('1');
  });

  it('discards undirty edit silently when the row leaves visible items', async () => {
    const { ElMessage } = await import('element-plus');
    (ElMessage.warning as any).mockClear();
    const { wrapper } = await mountList();
    await wrapper.find('.row-click-1').trigger('click');
    await flushPromises();
    expect((wrapper.vm as any).inlineEdit.isDirty()).toBe(false);

    visibleNodes.value = [{ kind: 'record', key: '2', payload: { Id: '2', Name: 'B', Sequence: 2 } }];
    await nextTick();
    await flushPromises();

    expect(ElMessage.warning).not.toHaveBeenCalled();
    expect((wrapper.vm as any).inlineEdit.isEditing.value).toBe(false);
  });

  it('keeps editing when the visible set still contains the row', async () => {
    const { wrapper } = await mountList();
    await wrapper.find('.row-click-1').trigger('click');
    await flushPromises();
    visibleNodes.value = [
      { kind: 'record', key: '1', payload: { Id: '1', Name: 'A', Sequence: 1 } },
      { kind: 'record', key: '2', payload: { Id: '2', Name: 'B', Sequence: 2 } },
      { kind: 'record', key: '3', payload: { Id: '3', Name: 'C', Sequence: 3 } },
    ];
    await nextTick();
    await flushPromises();
    expect((wrapper.vm as any).inlineEdit.isEditing.value).toBe(true);
    expect((wrapper.vm as any).inlineEdit.editingRowId.value).toBe('1');
  });

  it('disables handle reorder while editing', async () => {
    const { wrapper, store } = await mountList();
    await wrapper.find('.row-click-1').trigger('click');
    await flushPromises();
    store.UpdateById.mockClear();
    await wrapper.find('.reorder-handle').trigger('click');
    await flushPromises();
    expect(store.UpdateById).not.toHaveBeenCalled();
  });

  it('disables handle reorder when sorted by a non-handle or non-asc field', async () => {
    const byName = await mountList({ orderBy: [{ field: 'Name', direction: 'asc' }] });
    byName.store.UpdateById.mockClear();
    await byName.wrapper.find('.reorder-handle').trigger('click');
    await flushPromises();
    expect(byName.store.UpdateById).not.toHaveBeenCalled();

    const byDesc = await mountList({ orderBy: [{ field: 'Sequence', direction: 'desc' }] });
    byDesc.store.UpdateById.mockClear();
    await byDesc.wrapper.find('.reorder-handle').trigger('click');
    await flushPromises();
    expect(byDesc.store.UpdateById).not.toHaveBeenCalled();

    const byAsc = await mountList({ orderBy: { field: 'Sequence', direction: 'ASC' } });
    byAsc.store.UpdateById.mockClear();
    await byAsc.wrapper.find('.reorder-handle').trigger('click');
    await flushPromises();
    expect(byAsc.store.UpdateById).toHaveBeenCalled();

    // Missing direction is treated as non-asc and must disable reorder.
    const noDir = await mountList({ orderBy: [{ field: 'Sequence' }] });
    noDir.store.UpdateById.mockClear();
    await noDir.wrapper.find('.reorder-handle').trigger('click');
    await flushPromises();
    expect(noDir.store.UpdateById).not.toHaveBeenCalled();
  });

  it('reports error when handle refresh fails after successful writes', async () => {
    const { ElMessage } = await import('element-plus');
    const { wrapper, store } = await mountList();
    applyMock.mockRejectedValueOnce(new Error('refresh fail'));
    await wrapper.find('.reorder-handle').trigger('click');
    await flushPromises();
    expect(store.UpdateById).toHaveBeenCalled();
    expect(ElMessage.error).toHaveBeenCalledWith('Failed to refresh after reorder');
  });

  it('calls apply after successful inline save', async () => {
    const { wrapper } = await mountList();
    await wrapper.find('.row-click-1').trigger('click');
    await flushPromises();
    await wrapper.find('.mutate-draft').trigger('click');
    applyMock.mockClear();
    await wrapper.find('button[type="success"]').trigger('click');
    await flushPromises();
    expect(applyMock).toHaveBeenCalled();
  });

  it('falls back to clickToSelect when enterEdit fails in flat mode', async () => {
    const { wrapper } = await mountList({ clickToSelect: true });
    lastRowClickPayload = {
      rowData: { kind: 'record', key: 'x', payload: { Name: 'no-id' } },
      rowIndex: 0,
      rowKey: 'x',
      event: new MouseEvent('click'),
    };
    await wrapper.find('.row-click-custom').trigger('click');
    await flushPromises();
    expect((wrapper.vm as any).inlineEdit.isEditing.value).toBe(false);
  });

  it('falls back to clickToSelect when enterEdit fails in group mode', async () => {
    resultKind.value = 'group';
    const { wrapper } = await mountList({ clickToSelect: true });
    lastRowClickPayload = {
      rowData: { kind: 'record', key: 'x', payload: { Name: 'no-id' } },
      rowIndex: 0,
      rowKey: 'x',
      event: new MouseEvent('click'),
    };
    await wrapper.find('.row-click-custom').trigger('click');
    await flushPromises();
    expect((wrapper.vm as any).inlineEdit.isEditing.value).toBe(false);
  });

  it('emits row-click for group and flat records when not editable', async () => {
    resultKind.value = 'group';
    const grouped = await mountList({ editable: false });
    lastRowClickPayload = {
      rowData: { kind: 'record', key: '1', payload: { Id: '1', Name: 'A' } },
      rowIndex: 1,
      rowKey: '1',
      event: new MouseEvent('click'),
    };
    await grouped.wrapper.find('.row-click-custom').trigger('click');
    await flushPromises();
    expect(grouped.wrapper.emitted('row-click')?.length).toBeGreaterThan(0);

    resultKind.value = 'search';
    const flat = await mountList({ editable: false });
    await flat.wrapper.find('.row-click-1').trigger('click');
    await flushPromises();
    expect(flat.wrapper.emitted('row-click')?.length).toBeGreaterThan(0);
  });

  it('expands groups and loads more rows from group row clicks', async () => {
    resultKind.value = 'group';
    const { wrapper } = await mountList({ editable: false });

    lastRowClickPayload = {
      rowData: { kind: 'group', key: 'g1' },
      rowIndex: 0,
      rowKey: 'g1',
      event: new MouseEvent('click'),
    };
    await wrapper.find('.row-click-custom').trigger('click');
    await flushPromises();
    expect(expandGroupMock).toHaveBeenCalledWith('g1', true);

    lastRowClickPayload = {
      rowData: { kind: 'more', key: 'more:g1', groupKey: 'g1', target: 'records' },
      rowIndex: 1,
      rowKey: 'more:g1',
      event: new MouseEvent('click'),
    };
    await wrapper.find('.row-click-custom').trigger('click');
    await flushPromises();
    expect(loadMoreGroupRecordsMock).toHaveBeenCalledWith('g1');

    lastRowClickPayload = {
      rowData: { kind: 'more', key: 'more-g:g1', groupKey: 'g1', target: 'groups' },
      rowIndex: 2,
      rowKey: 'more-g:g1',
      event: new MouseEvent('click'),
    };
    await wrapper.find('.row-click-custom').trigger('click');
    await flushPromises();
    expect(loadMoreGroupChildrenMock).toHaveBeenCalledWith('g1');
  });
});

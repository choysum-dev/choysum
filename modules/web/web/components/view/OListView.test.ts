// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h, inject, nextTick, reactive, ref } from 'vue';
import { mount, flushPromises } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import OListView from '@/web/web/components/view/OListView.vue';
import { LIST_HANDLE_API_KEY } from '@/web/web/composables/useListHandleReorder';

const visibleNodes = ref<any[]>([
  { kind: 'record', key: '1', payload: { Id: '1', Name: 'A', Sequence: 1 } },
  { kind: 'record', key: '2', payload: { Id: '2', Name: 'B', Sequence: 2 } },
]);
const resultKind = ref<'search' | 'group'>('search');
const applyMock = vi.fn(async () => {});
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
    expandGroup: vi.fn(),
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
  const ctrl = { flush: vi.fn(async () => {}), reset: vi.fn(), force: vi.fn(), running: ref(false) };
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
    const handleApi = inject<any>(LIST_HANDLE_API_KEY, null);
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
      ]);
  },
});

const OVColumnStub = defineComponent({
  name: 'OVColumnStub',
  props: { type: String },
  setup(props) {
    return () => h('div', { class: 'ov-column-stub', 'data-type': props.type });
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

async function mountList(opts?: { editable?: boolean; showHandle?: boolean }) {
  const store = makeStore();
  const wrapper = mount(OListView, {
    props: {
      store: store as any,
      editable: opts?.editable ?? true,
      showHandle: opts?.showHandle ?? true,
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
  await nextTick();
  return { wrapper, store };
}

describe('OListView editable / handle', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    resultKind.value = 'search';
    lastRowClickPayload = null;
    visibleNodes.value = [
      { kind: 'record', key: '1', payload: { Id: '1', Name: 'A', Sequence: 1 } },
      { kind: 'record', key: '2', payload: { Id: '2', Name: 'B', Sequence: 2 } },
    ];
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
    const { wrapper, store } = await mountList();
    await wrapper.find('.row-click-1').trigger('click');
    await flushPromises();
    await wrapper.find('.mutate-draft').trigger('click');
    store.UpdateById = vi.fn(async () => {
      throw new Error('save fail');
    });
    await expect(wrapper.find('button[type="success"]').trigger('click')).resolves.toBeUndefined();
    await flushPromises();
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
});

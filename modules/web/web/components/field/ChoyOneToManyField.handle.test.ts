// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, defineComponent, h, inject, nextTick, ref } from 'vue';

import type { UseField } from '@/web/web/composables/useField';
import { LIST_HANDLE_API_KEY } from '@/web/web/composables/useListHandleReorder';
import { flushPromises, fnRecorder, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import FieldBase from './FieldBase.vue';
import ChoyOneToManyField from './ChoyOneToManyField.vue';
import ChoyTableColumn from '@/web/web/components/table/ChoyTableColumn.vue';
import ChoyTableHost from '@/web/web/components/internal/ChoyTableHost.vue';
import ChoyViewScope from '@/web/web/components/view/ChoyViewScope.vue';
import ChoyButton from '@/web/web/components/layout/ChoyButton.vue';

function makeBinding(opts: {
  items?: any[];
  isEditMode?: boolean;
  sequenceMeta?: boolean;
}) {
  const items = ref(
    opts.items ?? [
      { Id: '1', Sequence: 1, __rowKey: '1' },
      { Id: '2', Sequence: 2, __rowKey: '2' },
    ]
  );
  const fieldValue = ref(items.value.slice());
  const binding: UseField = {
    env: {
      isForm: true,
      isEditMode: opts.isEditMode ?? true,
      viewMode: opts.isEditMode === false ? 'readonly' : 'edit',
      fieldPrefix: null,
    },
    prop: 'Lines',
    meta: { type: 'oneToMany', typeAnnotation: '' } as any,
    fieldRef: () => fieldValue as any,
    fieldRefOf: () => fieldValue as any,
    recordRef: () => computed(() => ({ Id: 'parent' })) as any,
    registerFields: () => {},
    relationStore: {
      fieldsMetadata:
        opts.sequenceMeta === false
          ? { Name: { id: '1', type: 'string', typeAnnotation: '' } }
          : { Sequence: { id: '2', type: 'int', typeAnnotation: '', isReadonly: false } },
    } as any,
    asMutableArray: () => ({
      getItems: () => items.value,
      insertItem: fnRecorder(),
      removeItemAt: fnRecorder(),
    }),
    store: undefined,
    asView: () => ({ fieldValue: () => fieldValue }) as any,
  } as any;
  return { binding, fieldValue, items };
}

const capturedHandleApi: { current: any } = { current: null };

function installStubs() {
  stubSfc(FieldBase as any, {
    name: 'FieldBase',
    props: { binding: { type: Object, required: true } },
    setup(props: any, { slots }: any) {
      return () =>
        h('div', { class: 'field-base-stub' }, [
          props.binding.env.isEditMode ? slots.edit?.({}) : slots.display?.({}),
        ]);
    },
  });
  stubSfc(ChoyTableColumn as any, {
    name: 'ChoyTableColumn',
    props: { type: String, colKey: String },
    setup(props: any) {
      const cls = props.type === 'handle' ? 'ov-column-handle' : 'ov-column-stub';
      return () => h('div', { class: cls, 'data-type': props.type, 'data-key': props.colKey });
    },
  });
  stubSfc(ChoyTableHost as any, {
    name: 'ChoyTableHost',
    setup(_: any, { slots }: any) {
      capturedHandleApi.current = inject(LIST_HANDLE_API_KEY, null);
      return () => h('div', { class: 'ov-table-stub' }, slots.default?.());
    },
  });
  stubSfc(ChoyViewScope as any, {
    name: 'ChoyViewScope',
    setup(_: any, { slots }: any) {
      return () => h('div', { class: 'view-scope-stub' }, slots.default?.());
    },
  });
}

describe('ChoyOneToManyField handle column', () => {
  afterEach(() => {
    restoreSfc(FieldBase as any);
    restoreSfc(ChoyTableColumn as any);
    restoreSfc(ChoyTableHost as any);
    restoreSfc(ChoyViewScope as any);
    restoreSfc(ChoyButton as any);
    capturedHandleApi.current = null;
  });

  test('shows handle column in edit mode when Sequence metadata exists', async () => {
    installStubs();
    const { binding } = makeBinding({});
    const m = mountApp(ChoyOneToManyField as any, {
      props: { binding, showHandle: true },
      stubs: {
        'el-button': defineComponent({
          name: 'ElButtonStub',
          setup(_, { slots }) {
            return () => h('button', {}, slots.default?.());
          },
        }),
      },
    });
    await nextTick();
    expect(m.q('.ov-column-handle')).toBeTruthy();
    m.unmount();
  });

  test('hides handle column when showHandle is false or metadata lacks Sequence', async () => {
    installStubs();
    const noMeta = makeBinding({ sequenceMeta: false });
    const w1 = mountApp(ChoyOneToManyField as any, {
      props: { binding: noMeta.binding, showHandle: true },
      stubs: {
        'el-button': defineComponent({
          name: 'ElButtonStub',
          setup(_, { slots }) {
            return () => h('button', {}, slots.default?.());
          },
        }),
      },
    });
    expect(w1.q('.ov-column-handle')).toBeFalsy();
    w1.unmount();

    const withMeta = makeBinding({});
    const w2 = mountApp(ChoyOneToManyField as any, {
      props: { binding: withMeta.binding, showHandle: false },
      stubs: {
        'el-button': defineComponent({
          name: 'ElButtonStub',
          setup(_, { slots }) {
            return () => h('button', {}, slots.default?.());
          },
        }),
      },
    });
    expect(w2.q('.ov-column-handle')).toBeFalsy();
    w2.unmount();
  });

  test('onReorder assigns reordered rows to fieldRef', async () => {
    installStubs();
    capturedHandleApi.current = null;
    const { binding, fieldValue } = makeBinding({});
    const m = mountApp(ChoyOneToManyField as any, {
      props: { binding, showHandle: true },
      stubs: {
        'el-button': defineComponent({
          name: 'ElButtonStub',
          setup(_, { slots }) {
            return () => h('button', {}, slots.default?.());
          },
        }),
      },
    });
    await nextTick();
    await flushPromises();

    const api = capturedHandleApi.current;
    expect(api).toBeTruthy();
    api.onDragStart(0, {
      preventDefault: fnRecorder(),
      dataTransfer: { effectAllowed: '', setData: fnRecorder() },
    });
    await api.onDrop(1, { preventDefault: fnRecorder() });
    expect(fieldValue.value.map((r: any) => r.Id)).toEqual(['2', '1']);
    expect(fieldValue.value.map((r: any) => r.Sequence)).toEqual([1, 2]);
    m.unmount();
  });

  test('shows handle column with explicit handleField prop', async () => {
    installStubs();
    const { binding } = makeBinding({});
    const m = mountApp(ChoyOneToManyField as any, {
      props: { binding, showHandle: true, handleField: 'Sequence' },
      stubs: {
        'el-button': defineComponent({
          name: 'ElButtonStub',
          setup(_, { slots }) {
            return () => h('button', {}, slots.default?.());
          },
        }),
      },
    });
    await nextTick();
    expect(m.q('.ov-column-handle')).toBeTruthy();
    m.unmount();
  });

  test('Add row / Delete / display height and defaultRecord hydrate keys', async () => {
    installStubs();
    const scrollToRow = fnRecorder();
    stubSfc(ChoyTableHost as any, {
      name: 'ChoyTableHost',
      props: { data: Array, tableHeight: Number },
      setup(props: any, { slots, expose }: any) {
        capturedHandleApi.current = inject(LIST_HANDLE_API_KEY, null);
        expose({ scrollToRow });
        return () =>
          h(
            'div',
            {
              class: 'ov-table-stub',
              'data-height': String(props.tableHeight ?? ''),
              'data-rows': String((props.data || []).length),
            },
            slots.default?.()
          );
      },
    });

    const items = ref([{ Id: '1', Name: 'a', __rowKey: '1' }]);
    const fieldValue = ref(items.value.slice());
    const insertItem = fnRecorder((row: any) => {
      items.value = [...items.value, row];
      fieldValue.value = items.value.slice();
    });
    const removeItemAt = fnRecorder((i: number) => {
      items.value = items.value.filter((_: any, idx: number) => idx !== i);
      fieldValue.value = items.value.slice();
    });
    const binding: UseField = {
      env: { isForm: true, isEditMode: true, viewMode: 'edit', fieldPrefix: null },
      prop: 'Lines',
      meta: { type: 'oneToMany', typeAnnotation: '' } as any,
      fieldRef: () => fieldValue as any,
      fieldRefOf: () => fieldValue as any,
      recordRef: () => computed(() => ({ Id: 'parent' })) as any,
      registerFields: () => {},
      relationStore: {
        fieldsMetadata: { Sequence: { id: '2', type: 'int', typeAnnotation: '', isReadonly: false } },
      } as any,
      asMutableArray: () => ({
        getItems: () => items.value,
        insertItem,
        removeItemAt,
      }),
      store: undefined,
      asView: () => ({ fieldValue: () => fieldValue }) as any,
    } as any;

    const m = mountApp(ChoyOneToManyField as any, {
      props: {
        binding,
        showHandle: false,
        defaultRecord: () => ({ Name: 'new', __rowKey: 'seed-new' }),
        minTableHeight: 80,
        maxTableHeight: 400,
      },
    });
    await nextTick();
    const ss = m.setupState();
    expect(ss).toBeTruthy();
    await ss.handleAddItem();
    await flushPromises();
    expect(insertItem.calls.length).toBe(1);
    expect(insertItem.calls[0]?.[0]?.Name).toBe('new');
    expect(Object.prototype.propertyIsEnumerable.call(insertItem.calls[0]?.[0], '__rowKey')).toBe(false);
    expect(scrollToRow.calls.length).toBe(1);

    ss.onRemove(0);
    expect(removeItemAt.calls.length).toBe(1);
    m.unmount();

    // Display mode covers tableHeightDisplay.
    restoreSfc(FieldBase as any);
    stubSfc(FieldBase as any, {
      name: 'FieldBase',
      props: { binding: { type: Object, required: true } },
      setup(_: any, { slots }: any) {
        return () => h('div', { class: 'field-base-stub' }, slots.display?.({}));
      },
    });
    const displayItems = ref([{ Id: 'd1' }, { Id: 'd2' }]);
    const displayBinding: UseField = {
      env: { isForm: true, isEditMode: false, viewMode: 'display', fieldPrefix: null },
      prop: 'Lines',
      meta: { type: 'oneToMany' } as any,
      fieldRef: () => displayItems as any,
      fieldRefOf: () => displayItems as any,
      recordRef: () => computed(() => ({ Id: 'p' })) as any,
      registerFields: () => {},
      relationStore: { fieldsMetadata: {} } as any,
      asMutableArray: () => ({
        getItems: () => displayItems.value,
        insertItem: fnRecorder(),
        removeItemAt: fnRecorder(),
      }),
      store: undefined,
      asView: () => ({ fieldValue: () => displayItems }) as any,
    } as any;
    const d = mountApp(ChoyOneToManyField as any, {
      props: {
        binding: displayBinding,
        minTableHeight: 50,
        maxTableHeight: 300,
        rowHeightDisplay: 40,
      },
    });
    await nextTick();
    expect(d.q('.ov-table-stub')?.getAttribute('data-rows')).toBe('2');
    expect(d.q('.ov-table-stub')?.getAttribute('data-height')).toBeTruthy();
    d.unmount();

    // ensureArrayInitialized when fieldRef is not an array.
    restoreSfc(FieldBase as any);
    installStubs();
    stubSfc(ChoyTableHost as any, {
      name: 'ChoyTableHost',
      props: { data: Array, tableHeight: Number },
      setup(_: any, { expose }: any) {
        expose({ scrollToRow: fnRecorder() });
        return () => h('div', { class: 'ov-table-stub' });
      },
    });
    const nullItems = ref<any>(null);
    const ensureInsert = fnRecorder((row: any) => {
      if (!Array.isArray(nullItems.value)) nullItems.value = [];
      nullItems.value = [...nullItems.value, row];
    });
    const ensureBinding: UseField = {
      env: { isForm: true, isEditMode: true, viewMode: 'edit', fieldPrefix: null },
      prop: 'Lines',
      meta: { type: 'oneToMany' } as any,
      fieldRef: () => nullItems as any,
      fieldRefOf: () => nullItems as any,
      recordRef: () => computed(() => ({})) as any,
      registerFields: () => {},
      relationStore: { fieldsMetadata: {} } as any,
      asMutableArray: () => ({
        getItems: () => (Array.isArray(nullItems.value) ? nullItems.value : []),
        insertItem: ensureInsert,
        removeItemAt: fnRecorder(),
      }),
      store: undefined,
      asView: () => ({ fieldValue: () => nullItems }) as any,
    } as any;
    const e = mountApp(ChoyOneToManyField as any, {
      props: { binding: ensureBinding, defaultRecord: { Name: 'from-obj' } },
    });
    await nextTick();
    await e.setupState().handleAddItem();
    await flushPromises();
    expect(Array.isArray(nullItems.value)).toBe(true);
    expect(ensureInsert.calls.length).toBe(1);
    e.unmount();
  });
});

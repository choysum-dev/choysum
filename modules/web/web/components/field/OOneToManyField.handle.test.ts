// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, defineComponent, h, inject, nextTick, ref } from 'vue';

import type { UseField } from '@/web/web/composables/useField';
import { LIST_HANDLE_API_KEY } from '@/web/web/composables/useListHandleReorder';
import { flushPromises, fnRecorder, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import OFieldBase from './OFieldBase.vue';
import OOneToManyField from './OOneToManyField.vue';
import OVColumn from '@/web/web/components/vtable/OVColumn.vue';
import OVTable from '@/web/web/components/vtable/OVTable.vue';
import OViewScope from '@/web/web/components/view/OViewScope.vue';

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
  stubSfc(OFieldBase as any, {
    name: 'OFieldBase',
    props: { binding: { type: Object, required: true } },
    setup(props: any, { slots }: any) {
      return () =>
        h('div', { class: 'field-base-stub' }, [
          props.binding.env.isEditMode ? slots.edit?.({}) : slots.display?.({}),
        ]);
    },
  });
  stubSfc(OVColumn as any, {
    name: 'OVColumn',
    props: { type: String, colKey: String },
    setup(props: any) {
      const cls = props.type === 'handle' ? 'ov-column-handle' : 'ov-column-stub';
      return () => h('div', { class: cls, 'data-type': props.type, 'data-key': props.colKey });
    },
  });
  stubSfc(OVTable as any, {
    name: 'OVTable',
    setup(_: any, { slots }: any) {
      capturedHandleApi.current = inject(LIST_HANDLE_API_KEY, null);
      return () => h('div', { class: 'ov-table-stub' }, slots.default?.());
    },
  });
  stubSfc(OViewScope as any, {
    name: 'OViewScope',
    setup(_: any, { slots }: any) {
      return () => h('div', { class: 'view-scope-stub' }, slots.default?.());
    },
  });
}

describe('OOneToManyField handle column', () => {
  afterEach(() => {
    restoreSfc(OFieldBase as any);
    restoreSfc(OVColumn as any);
    restoreSfc(OVTable as any);
    restoreSfc(OViewScope as any);
    capturedHandleApi.current = null;
  });

  test('shows handle column in edit mode when Sequence metadata exists', async () => {
    installStubs();
    const { binding } = makeBinding({});
    const m = mountApp(OOneToManyField as any, {
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
    const w1 = mountApp(OOneToManyField as any, {
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
    const w2 = mountApp(OOneToManyField as any, {
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
    const m = mountApp(OOneToManyField as any, {
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
    const m = mountApp(OOneToManyField as any, {
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
});

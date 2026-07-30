// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, defineComponent, h, inject, nextTick, ref } from 'vue';
import { mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';
import type { UseField } from '@/web/web/composables/useField';
import OOneToManyField from '@/web/web/components/field/OOneToManyField.vue';
import { LIST_HANDLE_API_KEY } from '@/web/web/composables/useListHandleReorder';

vi.mock('@/web/web/i18n', async () => {
  const actual = await vi.importActual<typeof import('@/web/web/i18n')>('@/web/web/i18n');
  return {
    ...actual,
    createTranslate: () => ({ _t: (msg: string) => msg }),
  };
});

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
      insertItem: vi.fn(),
      removeItemAt: vi.fn(),
    }),
    store: undefined,
    asView: () => ({ fieldValue: () => fieldValue }) as any,
  } as any;
  return { binding, fieldValue };
}

const OFieldBaseStub = defineComponent({
  name: 'OFieldBaseStub',
  props: { binding: { type: Object, required: true } },
  setup(props, { slots }) {
    return () =>
      h('div', { class: 'field-base-stub' }, [
        props.binding.env.isEditMode ? slots.edit?.({}) : slots.display?.({}),
      ]);
  },
});

const OVColumnStub = defineComponent({
  name: 'OVColumnStub',
  props: { type: String, colKey: String },
  setup(props) {
    return () => h('div', { class: 'ov-column-stub', 'data-type': props.type, 'data-key': props.colKey });
  },
});

const OVTableStub = defineComponent({
  name: 'OVTableStub',
  setup(_, { slots }) {
    const handleApi = inject(LIST_HANDLE_API_KEY, null);
    (globalThis as any).__o2mHandleApi = handleApi;
    return () => h('div', { class: 'ov-table-stub' }, slots.default?.());
  },
});

const globalStubs = {
  OFieldBase: OFieldBaseStub,
  OVTable: OVTableStub,
  OVColumn: OVColumnStub,
  OViewScope: { template: '<div><slot /></div>' },
  ElButton: { template: '<button><slot /></button>' },
};

describe('OOneToManyField handle column', () => {
  it('shows handle column in edit mode when Sequence metadata exists', async () => {
    const { binding } = makeBinding({});
    const wrapper = mount(OOneToManyField, {
      props: { binding, showHandle: true },
      global: { stubs: globalStubs },
    });
    await nextTick();
    expect(wrapper.find('.ov-column-stub[data-type="handle"]').exists()).toBe(true);
  });

  it('hides handle column when showHandle is false or metadata lacks Sequence', async () => {
    const noMeta = makeBinding({ sequenceMeta: false });
    const w1 = mount(OOneToManyField, {
      props: { binding: noMeta.binding, showHandle: true },
      global: { stubs: globalStubs },
    });
    expect(w1.find('.ov-column-stub[data-type="handle"]').exists()).toBe(false);

    const withMeta = makeBinding({});
    const w2 = mount(OOneToManyField, {
      props: { binding: withMeta.binding, showHandle: false },
      global: { stubs: globalStubs },
    });
    expect(w2.find('.ov-column-stub[data-type="handle"]').exists()).toBe(false);
  });

  it('onReorder assigns reordered rows to fieldRef', async () => {
    const { binding, fieldValue } = makeBinding({});
    (globalThis as any).__o2mHandleApi = undefined;

    mount(OOneToManyField, {
      props: { binding, showHandle: true },
      global: { stubs: { ...globalStubs, OVTable: OVTableStub } },
    });
    await nextTick();

    const capturedApi = (globalThis as any).__o2mHandleApi;
    expect(capturedApi).toBeTruthy();
    capturedApi.onDragStart(0, {
      preventDefault: vi.fn(),
      dataTransfer: { effectAllowed: '', setData: vi.fn() },
    });
    await capturedApi.onDrop(1, { preventDefault: vi.fn() });
    expect(fieldValue.value.map((r: any) => r.Id)).toEqual(['2', '1']);
    expect(fieldValue.value.map((r: any) => r.Sequence)).toEqual([1, 2]);
  });

  it('shows handle column with explicit handleField prop', async () => {
    const { binding } = makeBinding({});
    const wrapper = mount(OOneToManyField, {
      props: { binding, showHandle: true, handleField: 'Sequence' },
      global: { stubs: globalStubs },
    });
    await nextTick();
    expect(wrapper.find('.ov-column-stub[data-type="handle"]').exists()).toBe(true);
  });
});

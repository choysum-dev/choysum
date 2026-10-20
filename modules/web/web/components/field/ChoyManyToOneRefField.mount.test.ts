// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, h, nextTick, ref } from 'vue';
import { createPinia, setActivePinia } from 'pinia';

import type { UseField } from '@/web/web/composables/useField';
import { useAuthStore } from '@/auth/web/stores/auth';
import { flushPromises, fnRecorder, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import ChoyFieldBase from './ChoyFieldBase.vue';
import ChoyManyToOneRefField from './ChoyManyToOneRefField.vue';
import RelationCombobox from '@/web/web/components/internal/RelationCombobox.vue';

function makeBinding(opts?: {
  value?: any;
  prop?: string;
  relationStore?: any;
  meta?: Record<string, unknown>;
}): { binding: UseField; value: any } {
  const value = ref(opts?.value ?? null);
  const record = ref({ Id: '1' });
  const binding = {
    env: {
      isForm: true,
      isEditMode: false,
      viewMode: 'display',
      fieldPrefix: null,
    },
    prop: opts?.prop || 'CurrencyId',
    meta: (opts?.meta ?? { type: 'ManyToOneRef', relationModel: 'base.Currency' }) as any,
    fieldRef: () => value as any,
    fieldRefOf: () => value as any,
    recordRef: () => computed(() => record.value) as any,
    registerFields: () => {},
    relationStore: opts?.relationStore,
    store: undefined,
    asView: () => ({ fieldValue: () => value }) as any,
  } as any;
  return { binding, value };
}

function installStubs() {
  stubSfc(ChoyFieldBase as any, {
    name: 'ChoyFieldBase',
    inheritAttrs: false,
    props: {
      binding: { type: Object, required: true },
      toView: { type: Function, default: undefined },
      fromView: { type: Function, default: undefined },
      rules: { type: Array, default: undefined },
      label: { type: String, default: undefined },
      formItemProps: { type: Object, default: undefined },
      orientation: { type: String, default: undefined },
      vColumnProps: { type: Object, default: undefined },
      required: { type: [Boolean, Function, Object], default: undefined },
      readonly: { type: [Boolean, Function, Object], default: undefined },
      visible: { type: [Boolean, Function, Object], default: undefined },
      cellVisible: { type: [Boolean, Function, Object], default: undefined },
      renderMode: { type: String, default: undefined },
      showInlineError: { type: Boolean, default: undefined },
    },
    setup(p: any, { slots }: any) {
      const fieldValue = () => (p.binding as UseField).fieldRef();
      const rootRef = (p.binding as UseField).recordRef() as any;
      const record = rootRef && typeof rootRef === 'object' && 'value' in rootRef ? rootRef.value : rootRef;
      return () =>
        h('div', { class: 'ob' }, [
          slots.edit?.({ fieldValue, record, inputId: 'f1', describedBy: '', invalid: false }),
          slots.display?.({ fieldValue, record }),
        ]);
    },
  });
  stubSfc(RelationCombobox as any, {
    name: 'RelationComboboxStub',
    inheritAttrs: false,
    props: {
      modelValue: { type: [String, null] as any, default: null },
      selectedOption: { type: Object, default: null },
      search: { type: Function, default: undefined },
      searchKey: { type: String, default: '' },
      clearable: { type: Boolean, default: true },
      placeholder: { type: String, default: '' },
      pageSize: { type: Number, default: 20 },
      searchMore: { type: Boolean, default: false },
    },
    setup(props: any) {
      return () =>
        h('div', {
          class: 'select-stub',
          'data-selected-label': String(props.selectedOption?.label ?? ''),
        });
    },
  });
}

function mountRef(binding: UseField) {
  return mountApp(ChoyManyToOneRefField as any, {
    props: { binding, renderMode: 'form' },
  });
}

describe('ChoyManyToOneRefField display hydrate', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    useAuthStore().identity = { metadata: { activeCompanyId: 'c1' } } as any;
    installStubs();
  });
  afterEach(() => {
    restoreSfc();
  });

  test('bare currency id Browses relation store and shows DisplayName', async () => {
    const Browse = fnRecorder(async (id: string) => ({ Id: id, DisplayName: '人民币', Code: 'CNY' }));
    const { binding } = makeBinding({
      value: 'cur-cny-1',
      relationStore: { Browse, fullModelName: 'base.Currency', storeId: 'base.Currency:1' },
    });

    const m = mountRef(binding);
    await flushPromises();
    await nextTick();

    expect(Browse.calls.length).toBe(1);
    expect(Browse.calls[0]?.[0]).toBe('cur-cny-1');
    expect(Browse.calls[0]?.[1]).toEqual(['Id', 'DisplayName']);
    expect(m.text()).toContain('人民币');
    m.unmount();
  });

  test('id object without DisplayName Browses and fills label via local options', async () => {
    const Browse = fnRecorder(async (id: string) => ({ Id: id, DisplayName: 'US Dollar' }));
    const { binding } = makeBinding({
      value: { Id: 'cur-usd-1' },
      relationStore: { Browse, fullModelName: 'base.Currency', storeId: 'base.Currency:1' },
    });

    const m = mountRef(binding);
    await flushPromises();
    await nextTick();

    expect(Browse.calls.length).toBe(1);
    expect(Browse.calls[0]?.[0]).toBe('cur-usd-1');
    expect(m.text()).toContain('US Dollar');
    m.unmount();
  });

  test('id-less currency shell does not Browse (no FK to resolve)', async () => {
    const Browse = fnRecorder(async () => ({ Id: 'x', DisplayName: 'X' }));
    const { binding } = makeBinding({
      // Failure mode before field-selection fix: nested companions left only DecimalDigits.
      value: { DecimalDigits: 2 },
      relationStore: { Browse, fullModelName: 'base.Currency', storeId: 'base.Currency:1' },
    });

    const m = mountRef(binding);
    await flushPromises();
    await nextTick();

    expect(Browse.calls.length).toBe(0);
    expect(m.text()).not.toContain('X');
    m.unmount();
  });
});

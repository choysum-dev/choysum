// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h, nextTick, reactive, unref } from 'vue';

import { flushPromises, fnRecorder, mountApp, stub, stubSfc, restoreSfc } from '@/web/web/__tests__/mountApp';
import ChoyButton from '@/web/web/components/layout/ChoyButton.vue';
import SearchFilterCondition from './SearchFilterCondition.vue';

import ChoyVarcharField from '@/web/web/components/field/ChoyVarcharField.vue';
import ChoyTextField from '@/web/web/components/field/ChoyTextField.vue';
import ChoyIntField from '@/web/web/components/field/ChoyIntField.vue';
import ChoyBigintField from '@/web/web/components/field/ChoyBigintField.vue';
import ChoyNumberField from '@/web/web/components/field/ChoyNumberField.vue';
import ChoyDecimalField from '@/web/web/components/field/ChoyDecimalField.vue';
import ChoyMonetaryField from '@/web/web/components/field/ChoyMonetaryField.vue';
import ChoyBooleanField from '@/web/web/components/field/ChoyBooleanField.vue';
import ChoyDateField from '@/web/web/components/field/ChoyDateField.vue';
import ChoyTimeField from '@/web/web/components/field/ChoyTimeField.vue';
import ChoyDatetimeField from '@/web/web/components/field/ChoyDatetimeField.vue';
import ChoyJsonField from '@/web/web/components/field/ChoyJsonField.vue';
import ChoyManyToOneField from '@/web/web/components/field/ChoyManyToOneField.vue';
import ChoyManyToOneRefField from '@/web/web/components/field/ChoyManyToOneRefField.vue';
import ChoyBinaryField from '@/web/web/components/field/ChoyBinaryField.vue';
import ChoyImageField from '@/web/web/components/field/ChoyImageField.vue';
import ChoySelectionField from '@/web/web/components/field/ChoySelectionField.vue';


const fieldSfcs = [
  ChoyVarcharField,
  ChoyTextField,
  ChoyIntField,
  ChoyBigintField,
  ChoyNumberField,
  ChoyDecimalField,
  ChoyMonetaryField,
  ChoyBooleanField,
  ChoyDateField,
  ChoyTimeField,
  ChoyDatetimeField,
  ChoyJsonField,
  ChoyManyToOneField,
  ChoyManyToOneRefField,
  ChoyBinaryField,
  ChoyImageField,
  ChoySelectionField,
];

const fieldStubClass: Array<[any, string]> = [
  [ChoyVarcharField, 'f-varchar'],
  [ChoyTextField, 'f-text'],
  [ChoyIntField, 'f-int'],
  [ChoyBigintField, 'f-bigint'],
  [ChoyNumberField, 'f-number'],
  [ChoyDecimalField, 'f-decimal'],
  [ChoyMonetaryField, 'f-monetary'],
  [ChoyBooleanField, 'f-bool'],
  [ChoyDateField, 'f-date'],
  [ChoyTimeField, 'f-time'],
  [ChoyDatetimeField, 'f-dt'],
  [ChoyJsonField, 'f-json'],
  [ChoyManyToOneField, 'f-m2o'],
  [ChoyManyToOneRefField, 'f-m2oref'],
  [ChoyBinaryField, 'f-bin'],
  [ChoyImageField, 'f-img'],
  [ChoySelectionField, 'f-selection'],
];

function installFieldStubs() {
  for (const [Comp, cls] of fieldStubClass) {
    stubSfc(Comp, {
      name: (Comp as any).__name || cls,
      setup() {
        return () => h('div', { class: cls });
      },
    });
  }
  stubSfc(ChoyButton as any, {
    name: 'ChoyButton',
    inheritAttrs: false,
    emits: ['click'],
    setup(_: any, { slots, emit, attrs }: any) {
      return () =>
        h(
          'button',
          {
            type: 'button',
            class: attrs.class || 'rm',
            onClick: () => emit('click'),
          },
          slots.default?.()
        );
    },
  });
}

function restoreFieldStubs() {
  for (const Comp of fieldSfcs) restoreSfc(Comp);
  restoreSfc(ChoyButton as any);
}

const epStubs = {
  ElSelect: stub('ElSelect'),
  ElOption: stub('ElOption'),
  ElButton: stub('ElButton'),
  ElInput: stub('ElInput'),
};

function installChoyButtonStub() {
  stubSfc(ChoyButton as any, {
    name: 'ChoyButton',
    props: {
      class: { type: [String, Object, Array], default: undefined },
      disabled: { type: Boolean, default: false },
      variant: { type: String, default: 'default' },
      size: { type: String, default: 'default' },
    },
    emits: ['click'],
    setup(props: any, { slots, emit, attrs }: any) {
      return () =>
        h(
          'button',
          {
            class: props.class || attrs.class || 'rm',
            type: 'button',
            onClick: () => emit('click'),
          },
          slots.default?.(),
        );
    },
  });
}

describe('SearchFilterCondition', () => {
  beforeEach(() => {
    installFieldStubs();
    installChoyButtonStub();
  });
  afterEach(() => {
    restoreFieldStubs();
    restoreSfc(ChoyButton as any);
  });

  function mountRow(condition: any, extras: Record<string, any> = {}) {
    const onUpdateCondition = fnRecorder();
    const onRemoveCondition = fnRecorder();
    const store = {
      fieldsMetadata: {
        Name: { type: 'varchar', string: 'Name' },
        Status: { type: 'selection', string: 'Status' },
        PartnerId: { type: 'manytoone', relationModel: 'base.Partner', string: 'Partner' },
        Amount: { type: 'monetary', currencyField: 'CurrencyId', string: 'Amount' },
        CreatedAt: { type: 'datetime', string: 'Created At', isReadonly: true },
        DisplayName: { type: 'varchar', string: 'Display Name', isReadonly: true },
      },
      getFieldMeta(name: string) {
        return this.fieldsMetadata[name];
      },
      ensureFieldsGet: fnRecorder(async () => ({})),
      getFieldsGetTranslatedString: () => undefined,
      ...extras.store,
    };
    const m = mountApp(SearchFilterCondition as any, {
      props: {
        condition,
        fields: [
          { prop: 'Name', label: '名称' },
          { prop: 'Status', label: '状态' },
          { prop: 'PartnerId', label: '合作伙伴' },
          { prop: 'Amount', label: '金额' },
          { prop: 'CreatedAt', label: '创建时间' },
          { prop: 'DisplayName', label: '显示名称' },
        ],
        store,
        onUpdateCondition,
        onRemoveCondition,
      },
      stubs: epStubs,
    });
    return { ...m, onUpdateCondition, onRemoveCondition, store };
  }

  test('patches field change with first operator and clears value', async () => {
    const condition = reactive<any>({ id: 'c1', field: 'Name', operator: '=', value: 'x' });
    const { unmount, setupState, onUpdateCondition, q } = mountRow(condition);
    await setupState().onFieldChange('Status');
    expect(onUpdateCondition.calls.length).toBeGreaterThan(0);
    const patch = onUpdateCondition.calls[0]![1] as any;
    expect(patch.field).toBe('Status');
    expect(patch.value).toBeUndefined();
    expect(typeof patch.operator).toBe('string');

    // Controlled <select> dispatches through onFieldChange without mutating the prop in place first.
    onUpdateCondition.calls.length = 0;
    condition.field = 'Name';
    await nextTick();
    const fieldSelect = q('.w-field') as HTMLSelectElement | null;
    expect(fieldSelect).toBeTruthy();
    fieldSelect!.value = 'Status';
    fieldSelect!.dispatchEvent(new Event('change', { bubbles: true }));
    await nextTick();
    expect(onUpdateCondition.calls.length).toBeGreaterThan(0);
    expect((onUpdateCondition.calls[0]![1] as any).field).toBe('Status');
    unmount();
  });

  test('maps null / multi-value / default operators', async () => {
    const condition = reactive<any>({ id: 'c1', field: 'Name', operator: '=', value: 'x' });
    const { unmount, setupState, onUpdateCondition } = mountRow(condition);

    await setupState().onOperatorChange('is');
    expect(onUpdateCondition.calls[onUpdateCondition.calls.length - 1]![1]).toEqual({ operator: 'is', value: null });

    condition.value = 'solo';
    await setupState().onOperatorChange('in');
    expect(onUpdateCondition.calls[onUpdateCondition.calls.length - 1]![1]).toEqual({ operator: 'in', value: ['solo'] });

    condition.value = null;
    condition.field = 'Name';
    await setupState().onOperatorChange('=');
    const last = onUpdateCondition.calls[onUpdateCondition.calls.length - 1]![1] as any;
    expect(last.operator).toBe('=');
    unmount();
  });

  test('renders multi-value select for in operator on scalars', async () => {
    const condition = reactive<any>({ id: 'c1', field: 'Name', operator: 'in', value: ['a', 'b'] });
    const { unmount, setupState, onUpdateCondition, qa } = mountRow(condition);
    await nextTick();
    const multi = qa('.w-value').find(s => (s as HTMLInputElement).tagName === 'INPUT' && !(s as HTMLInputElement).disabled);
    expect(multi).toBeTruthy();
    await setupState().onMultiValuesChange(['x', 'y']);
    expect(onUpdateCondition.calls[onUpdateCondition.calls.length - 1]).toEqual(['c1', { value: ['x', 'y'] }]);
    unmount();
  });

  test('keeps value editor writable for form-readonly fields', async () => {
    const condition = reactive<any>({ id: 'c1', field: 'DisplayName', operator: '=', value: '' });
    const { unmount, setupState, q } = mountRow(condition);
    await flushPromises();
    expect(q('.f-varchar')).toBeTruthy();
    const meta = unref(setupState().fieldMeta);
    expect(meta.isReadonly).toBe(false);
    unmount();
  });

  test('uses manytoone component for relation fields', async () => {
    const condition = reactive<any>({ id: 'c1', field: 'PartnerId', operator: '=', value: null });
    const { unmount, q } = mountRow(condition);
    await nextTick();
    expect(q('.f-m2o')).toBeTruthy();
    unmount();
  });

  test('removes the condition row', async () => {
    const condition = reactive<any>({ id: 'c1', field: 'Name', operator: '=', value: '' });
    const { unmount, setupState, onRemoveCondition } = mountRow(condition);
    setupState().onRemove();
    expect(onRemoveCondition.calls[0]).toEqual(['c1']);
    unmount();
  });

  test('renders NULL flag and selection / datetime field components', async () => {
    const nullCond = reactive<any>({ id: 'c1', field: 'Name', operator: 'is', value: null });
    const nullRow = mountRow(nullCond);
    expect(nullRow.q('.choy-null-flag')?.textContent).toBe('NULL');
    nullRow.unmount();

    const sel = reactive<any>({ id: 'c2', field: 'Status', operator: '=', value: 'a' });
    const selRow = mountRow(sel);
    expect(selRow.q('.f-selection')).toBeTruthy();
    selRow.unmount();

    const dt = reactive<any>({ id: 'c3', field: 'CreatedAt', operator: '=', value: null });
    const dtRow = mountRow(dt);
    expect(dtRow.q('.f-dt')).toBeTruthy();
    dtRow.unmount();
  });

  test('exposes toView/fromView helpers for manytoone value binding', async () => {
    const condition = reactive<any>({ id: 'c1', field: 'PartnerId', operator: '=', value: 'p1' });
    const { unmount, setupState } = mountRow(condition);
    await nextTick();
    const extras = unref(setupState().extraProps);
    expect(extras.toView(null)).toBeNull();
    expect(extras.toView({ Id: 'x' })).toEqual({ Id: 'x' });
    expect(extras.toView('y')).toEqual({ Id: 'y' });
    expect(extras.fromView(null)).toBeNull();
    expect(extras.fromView({ Id: 'z' })).toBe('z');
    expect(extras.fromView('w')).toBe('w');
    unmount();
  });

  test('normalizes multiValues from scalar and empty values', async () => {
    const condition = reactive<any>({ id: 'c1', field: 'Name', operator: 'in', value: 'solo' });
    const { unmount, setupState } = mountRow(condition);
    expect(unref(setupState().multiValues)).toEqual(['solo']);
    condition.value = '';
    await nextTick();
    expect(unref(setupState().multiValues)).toEqual([]);
    condition.value = ['', 'a', null];
    await nextTick();
    expect(unref(setupState().multiValues)).toEqual(['a']);
    unmount();
  });

  test('maps every field type to a value editor and placeholder', async () => {
    const types: Array<[string, string, string]> = [
      ['Char', 'char', 'f-varchar'],
      ['Text', 'text', 'f-text'],
      ['Int', 'int', 'f-int'],
      ['Big', 'bigint', 'f-bigint'],
      ['Num', 'number', 'f-number'],
      ['Dec', 'decimal', 'f-decimal'],
      ['Mon', 'monetary', 'f-monetary'],
      ['Bool', 'boolean', 'f-bool'],
      ['Date', 'date', 'f-date'],
      ['Time', 'time', 'f-time'],
      ['Json', 'jsonobject', 'f-json'],
      ['Ref', 'manytooneref', 'f-m2oref'],
      ['Bin', 'binary', 'f-bin'],
      ['Img', 'image', 'f-img'],
      ['Html', 'html', 'f-varchar'],
      ['Unk', 'weird', 'f-varchar'],
    ];
    const fieldsMetadata: Record<string, any> = Object.fromEntries(
      types.map(([prop, type]) => [prop, { type, string: prop, relationModel: type.includes('many') ? 'base.X' : undefined }])
    );
    for (const [prop, , cls] of types) {
      const condition = reactive<any>({ id: 'c1', field: prop, operator: '=', value: null });
      const { unmount, q, setupState } = mountRow(condition, {
        store: {
          fieldsMetadata,
          getFieldMeta: undefined,
        },
      });
      await nextTick();
      expect(`${prop}:${q(`.${cls}`) ? 'found' : 'missing'}`).toBe(`${prop}:found`);
      const ph = unref(setupState().valuePlaceholder);
      expect(typeof ph).toBe('string');
      expect(ph.length).toBeGreaterThan(0);
      unmount();
    }
  });

  test('applies boolean default value and keeps multi-value arrays intact', async () => {
    const condition = reactive<any>({ id: 'c1', field: 'Bool', operator: 'is', value: null });
    const { unmount, setupState, onUpdateCondition } = mountRow(condition, {
      store: {
        fieldsMetadata: {
          Bool: { type: 'boolean', string: 'Bool' },
          Name: { type: 'varchar', string: 'Name' },
        },
      },
    });
    await setupState().onOperatorChange('=');
    expect(onUpdateCondition.calls[onUpdateCondition.calls.length - 1]![1]).toEqual({ operator: '=', value: false });

    condition.field = 'Name';
    condition.value = ['a', 'b'];
    await setupState().onOperatorChange('in');
    expect(onUpdateCondition.calls[onUpdateCondition.calls.length - 1]![1]).toEqual({ operator: 'in' });

    await setupState().onMultiValuesChange('not-array' as any);
    expect(onUpdateCondition.calls[onUpdateCondition.calls.length - 1]).toEqual(['c1', { value: [] }]);
    unmount();
  });

  test('falls back getFieldMeta via fieldsMetadata and clears relationStore on scalar fields', async () => {
    const condition = reactive<any>({ id: 'c1', field: 'PartnerId', operator: '=', value: null });
    const { unmount, setupState } = mountRow(condition, {
      store: {
        getFieldMeta: undefined,
        // FE host stubs createStoreByModel, so relationStore comes from that path
        // (getRelationStore fallback is covered in useFilterEditorBindings tests).
        getRelationStore: fnRecorder(() => ({ destroy: fnRecorder() })),
      },
    });
    await nextTick();
    expect(setupState().binding.store.getFieldMeta('PartnerId')?.type).toBe('manytoone');
    expect(setupState().binding.store.getFieldMeta('Missing')).toBeUndefined();
    expect(setupState().binding.relationStore).toBeTruthy();
    condition.field = 'Name';
    await nextTick();
    expect(setupState().binding.relationStore).toBeUndefined();
    unmount();
  });

  test('uses tempId as condition identity when present', async () => {
    const condition = reactive<any>({ id: 'c1', tempId: 'tmp-9', field: 'Name', operator: '=', value: 'x' });
    const { unmount, setupState, onUpdateCondition, onRemoveCondition } = mountRow(condition);
    await setupState().onFieldChange('Status');
    expect(onUpdateCondition.calls[0]![0]).toBe('tmp-9');
    setupState().onRemove();
    expect(onRemoveCondition.calls[0]).toEqual(['tmp-9']);
    unmount();
  });
});

// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h, nextTick, reactive, unref } from 'vue';

import { flushPromises, fnRecorder, mountApp, stub, stubSfc, restoreSfc } from '@/web/web/__tests__/mountApp';
import OSearchFilterCondition from './OSearchFilterCondition.vue';

import OCharField from '@/web/web/components/field/OCharField.vue';
import OVarCharField from '@/web/web/components/field/OVarCharField.vue';
import OTextField from '@/web/web/components/field/OTextField.vue';
import OIntField from '@/web/web/components/field/OIntField.vue';
import OBigintField from '@/web/web/components/field/OBigintField.vue';
import ONumberField from '@/web/web/components/field/ONumberField.vue';
import ODecimalField from '@/web/web/components/field/ODecimalField.vue';
import OMonetaryField from '@/web/web/components/field/OMonetaryField.vue';
import OBooleanField from '@/web/web/components/field/OBooleanField.vue';
import ODateField from '@/web/web/components/field/ODateField.vue';
import OTimeField from '@/web/web/components/field/OTimeField.vue';
import ODatetimeField from '@/web/web/components/field/ODatetimeField.vue';
import OJsonobjectField from '@/web/web/components/field/OJsonobjectField.vue';
import OManyToOneField from '@/web/web/components/field/OManyToOneField.vue';
import OManyToOneRefField from '@/web/web/components/field/OManyToOneRefField.vue';
import OBinaryField from '@/web/web/components/field/OBinaryField.vue';
import OImageField from '@/web/web/components/field/OImageField.vue';
import OSelectionField from '@/web/web/components/field/OSelectionField.vue';

const fieldSfcs = [
  OCharField,
  OVarCharField,
  OTextField,
  OIntField,
  OBigintField,
  ONumberField,
  ODecimalField,
  OMonetaryField,
  OBooleanField,
  ODateField,
  OTimeField,
  ODatetimeField,
  OJsonobjectField,
  OManyToOneField,
  OManyToOneRefField,
  OBinaryField,
  OImageField,
  OSelectionField,
];

const fieldStubClass: Array<[any, string]> = [
  [OCharField, 'f-char'],
  [OVarCharField, 'f-varchar'],
  [OTextField, 'f-text'],
  [OIntField, 'f-int'],
  [OBigintField, 'f-bigint'],
  [ONumberField, 'f-number'],
  [ODecimalField, 'f-decimal'],
  [OMonetaryField, 'f-monetary'],
  [OBooleanField, 'f-bool'],
  [ODateField, 'f-date'],
  [OTimeField, 'f-time'],
  [ODatetimeField, 'f-dt'],
  [OJsonobjectField, 'f-json'],
  [OManyToOneField, 'f-m2o'],
  [OManyToOneRefField, 'f-m2oref'],
  [OBinaryField, 'f-bin'],
  [OImageField, 'f-img'],
  [OSelectionField, 'f-selection'],
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
}

function restoreFieldStubs() {
  for (const Comp of fieldSfcs) restoreSfc(Comp);
}

const epStubs = {
  ElSelect: {
    name: 'ElSelect',
    props: {
      modelValue: { default: undefined },
      disabled: { type: Boolean, default: false },
      multiple: { type: Boolean, default: false },
    },
    emits: ['update:modelValue'],
    setup(props: any, { slots, emit }: any) {
      return () =>
        h(
          'div',
          {
            class: 'el-select',
            'data-disabled': String(!!props.disabled),
            'data-multi': String(!!props.multiple),
            onClick: () => emit('update:modelValue', props.modelValue),
          },
          slots.default?.()
        );
    },
  },
  ElOption: stub('ElOption'),
  ElButton: {
    name: 'ElButton',
    emits: ['click'],
    setup(_, { slots, emit }: any) {
      return () =>
        h('button', { class: 'rm', type: 'button', onClick: () => emit('click') }, slots.default?.());
    },
  },
  ElInput: stub('ElInput'),
};

describe('OSearchFilterCondition', () => {
  beforeEach(() => {
    installFieldStubs();
  });
  afterEach(() => {
    restoreFieldStubs();
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
    const m = mountApp(OSearchFilterCondition as any, {
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
    const condition = reactive({ id: 'c1', field: 'Name', operator: '=', value: 'x' });
    const { unmount, setupState, onUpdateCondition } = mountRow(condition);
    await setupState().onFieldChange('Status');
    expect(onUpdateCondition.calls.length).toBeGreaterThan(0);
    const patch = onUpdateCondition.calls[0]![1] as any;
    expect(patch.field).toBe('Status');
    expect(patch.value).toBeUndefined();
    expect(typeof patch.operator).toBe('string');
    unmount();
  });

  test('maps null / multi-value / default operators', async () => {
    const condition = reactive({ id: 'c1', field: 'Name', operator: '=', value: 'x' });
    const { unmount, setupState, onUpdateCondition } = mountRow(condition);

    await setupState().onOperatorChange('is');
    expect(onUpdateCondition.calls.at(-1)![1]).toEqual({ operator: 'is', value: null });

    condition.value = 'solo';
    await setupState().onOperatorChange('in');
    expect(onUpdateCondition.calls.at(-1)![1]).toEqual({ operator: 'in', value: ['solo'] });

    condition.value = null;
    condition.field = 'Name';
    await setupState().onOperatorChange('=');
    const last = onUpdateCondition.calls.at(-1)![1] as any;
    expect(last.operator).toBe('=');
    unmount();
  });

  test('renders multi-value select for in operator on scalars', async () => {
    const condition = reactive({ id: 'c1', field: 'Name', operator: 'in', value: ['a', 'b'] });
    const { unmount, setupState, onUpdateCondition, qa } = mountRow(condition);
    await nextTick();
    const multi = qa('.el-select').find(s => s.getAttribute('data-multi') === 'true');
    expect(multi).toBeTruthy();
    await setupState().onMultiValuesChange(['x', 'y']);
    expect(onUpdateCondition.calls.at(-1)).toEqual(['c1', { value: ['x', 'y'] }]);
    unmount();
  });

  test('keeps value editor writable for form-readonly fields', async () => {
    const condition = reactive({ id: 'c1', field: 'DisplayName', operator: '=', value: '' });
    const { unmount, setupState, q } = mountRow(condition);
    await flushPromises();
    expect(q('.f-varchar')).toBeTruthy();
    const meta = unref(setupState().fieldMeta);
    expect(meta.isReadonly).toBe(false);
    unmount();
  });

  test('uses manytoone component for relation fields', async () => {
    const condition = reactive({ id: 'c1', field: 'PartnerId', operator: '=', value: null });
    const { unmount, q } = mountRow(condition);
    await nextTick();
    expect(q('.f-m2o')).toBeTruthy();
    unmount();
  });

  test('removes the condition row', async () => {
    const condition = reactive({ id: 'c1', field: 'Name', operator: '=', value: '' });
    const { unmount, click, onRemoveCondition } = mountRow(condition);
    click('.rm');
    expect(onRemoveCondition.calls[0]).toEqual(['c1']);
    unmount();
  });

  test('renders NULL flag and selection / datetime field components', async () => {
    const nullCond = reactive({ id: 'c1', field: 'Name', operator: 'is', value: null });
    const nullRow = mountRow(nullCond);
    expect(nullRow.q('.o-null-flag')?.textContent).toBe('NULL');
    nullRow.unmount();

    const sel = reactive({ id: 'c2', field: 'Status', operator: '=', value: 'a' });
    const selRow = mountRow(sel);
    expect(selRow.q('.f-selection')).toBeTruthy();
    selRow.unmount();

    const dt = reactive({ id: 'c3', field: 'CreatedAt', operator: '=', value: null });
    const dtRow = mountRow(dt);
    expect(dtRow.q('.f-dt')).toBeTruthy();
    dtRow.unmount();
  });

  test('exposes toView/fromView helpers for manytoone value binding', async () => {
    const condition = reactive({ id: 'c1', field: 'PartnerId', operator: '=', value: 'p1' });
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
    const condition = reactive({ id: 'c1', field: 'Name', operator: 'in', value: 'solo' });
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
      ['Char', 'char', 'f-char'],
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
      ['Html', 'html', 'f-char'],
      ['Unk', 'weird', 'f-varchar'],
    ];
    const fieldsMetadata: Record<string, any> = Object.fromEntries(
      types.map(([prop, type]) => [prop, { type, string: prop, relationModel: type.includes('many') ? 'base.X' : undefined }])
    );
    for (const [prop, , cls] of types) {
      const condition = reactive({ id: 'c1', field: prop, operator: '=', value: null });
      const { unmount, q, setupState } = mountRow(condition, {
        store: {
          fieldsMetadata,
          getFieldMeta: undefined,
        },
      });
      await nextTick();
      expect(q(`.${cls}`), prop).toBeTruthy();
      const ph = unref(setupState().valuePlaceholder);
      expect(typeof ph).toBe('string');
      expect(ph.length).toBeGreaterThan(0);
      unmount();
    }
  });

  test('applies boolean default value and keeps multi-value arrays intact', async () => {
    const condition = reactive({ id: 'c1', field: 'Bool', operator: 'is', value: null });
    const { unmount, setupState, onUpdateCondition } = mountRow(condition, {
      store: {
        fieldsMetadata: {
          Bool: { type: 'boolean', string: 'Bool' },
          Name: { type: 'varchar', string: 'Name' },
        },
      },
    });
    await setupState().onOperatorChange('=');
    expect(onUpdateCondition.calls.at(-1)![1]).toEqual({ operator: '=', value: false });

    condition.field = 'Name';
    condition.value = ['a', 'b'];
    await setupState().onOperatorChange('in');
    expect(onUpdateCondition.calls.at(-1)![1]).toEqual({ operator: 'in' });

    await setupState().onMultiValuesChange('not-array' as any);
    expect(onUpdateCondition.calls.at(-1)).toEqual(['c1', { value: [] }]);
    unmount();
  });

  test('falls back getFieldMeta via fieldsMetadata and clears relationStore on scalar fields', async () => {
    const condition = reactive({ id: 'c1', field: 'PartnerId', operator: '=', value: null });
    const { unmount, setupState, store } = mountRow(condition, {
      store: {
        getFieldMeta: undefined,
        getRelationStore: fnRecorder(() => ({ destroy: fnRecorder() })),
      },
    });
    await nextTick();
    expect(setupState().binding.store.getFieldMeta('PartnerId')?.type).toBe('manytoone');
    expect(setupState().binding.store.getFieldMeta('Missing')).toBeUndefined();
    condition.field = 'Name';
    await nextTick();
    expect(setupState().binding.relationStore).toBeUndefined();
    expect((store.getRelationStore as any).calls.length).toBeGreaterThan(0);
    unmount();
  });

  test('uses tempId as condition identity when present', async () => {
    const condition = reactive({ id: 'c1', tempId: 'tmp-9', field: 'Name', operator: '=', value: 'x' });
    const { unmount, setupState, click, onUpdateCondition, onRemoveCondition } = mountRow(condition);
    await setupState().onFieldChange('Status');
    expect(onUpdateCondition.calls[0]![0]).toBe('tmp-9');
    click('.rm');
    expect(onRemoveCondition.calls[0]).toEqual(['tmp-9']);
    unmount();
  });
});

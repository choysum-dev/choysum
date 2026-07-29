// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount, flushPromises } from '@vue/test-utils';
import { nextTick, reactive } from 'vue';
import { describe, expect, it, vi } from 'vitest';

import OSearchFilterCondition from './OSearchFilterCondition.vue';

vi.mock('@/web/web/i18n', async () => {
  const actual = await vi.importActual<typeof import('@/web/web/i18n')>('@/web/web/i18n');
  return {
    ...actual,
    createTranslate: () => ({
      _t: (msg: string) => msg,
    }),
  };
});

describe('OSearchFilterCondition', () => {
  function mountRow(condition: any, extras: Record<string, any> = {}) {
    const onUpdateCondition = vi.fn();
    const onRemoveCondition = vi.fn();
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
      ensureFieldsGet: vi.fn(async () => ({})),
      getFieldsGetTranslatedString: () => undefined,
      ...extras.store,
    };
    const wrapper = mount(OSearchFilterCondition as any, {
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
      global: {
        stubs: {
          'el-select': {
            props: {
              modelValue: { default: undefined },
              disabled: { type: Boolean, default: false },
              multiple: { type: Boolean, default: false },
            },
            emits: ['update:modelValue'],
            template: `<div class="el-select" :data-disabled="String(!!disabled)" :data-multi="String(!!multiple)" @click="$emit('update:modelValue', modelValue)"><slot /></div>`,
          },
          'el-option': true,
          'el-button': {
            template: `<button class="rm" @click="$emit('click')"><slot /></button>`,
          },
          'el-input': true,
          OVarCharField: { template: `<div class="f-varchar" />` },
          OSelectionField: { template: `<div class="f-selection" />` },
          OManyToOneField: { template: `<div class="f-m2o" />` },
          ODatetimeField: { template: `<div class="f-dt" />` },
          OCharField: { template: `<div class="f-char" />` },
          OTextField: { template: `<div class="f-text" />` },
          OIntField: { template: `<div class="f-int" />` },
          OBigintField: { template: `<div class="f-bigint" />` },
          ONumberField: { template: `<div class="f-number" />` },
          ODecimalField: { template: `<div class="f-decimal" />` },
          OMonetaryField: { template: `<div class="f-monetary" />` },
          OBooleanField: { template: `<div class="f-bool" />` },
          ODateField: { template: `<div class="f-date" />` },
          OTimeField: { template: `<div class="f-time" />` },
          OJsonobjectField: { template: `<div class="f-json" />` },
          OManyToOneRefField: { template: `<div class="f-m2oref" />` },
          OBinaryField: { template: `<div class="f-bin" />` },
          OImageField: { template: `<div class="f-img" />` },
        },
      },
    });
    return { wrapper, onUpdateCondition, onRemoveCondition, store };
  }

  it('patches field change with first operator and clears value', async () => {
    const condition = reactive({ id: 'c1', field: 'Name', operator: '=', value: 'x' });
    const { wrapper, onUpdateCondition } = mountRow(condition);
    await (wrapper.vm as any).onFieldChange('Status');
    expect(onUpdateCondition).toHaveBeenCalled();
    const patch = onUpdateCondition.mock.calls[0][1];
    expect(patch.field).toBe('Status');
    expect(patch.value).toBeUndefined();
    expect(typeof patch.operator).toBe('string');
  });

  it('maps null / multi-value / default operators', async () => {
    const condition = reactive({ id: 'c1', field: 'Name', operator: '=', value: 'x' });
    const { wrapper, onUpdateCondition } = mountRow(condition);

    await (wrapper.vm as any).onOperatorChange('is');
    expect(onUpdateCondition.mock.calls.at(-1)![1]).toEqual({ operator: 'is', value: null });

    condition.value = 'solo';
    await (wrapper.vm as any).onOperatorChange('in');
    expect(onUpdateCondition.mock.calls.at(-1)![1]).toEqual({ operator: 'in', value: ['solo'] });

    condition.value = null;
    condition.field = 'Name';
    await (wrapper.vm as any).onOperatorChange('=');
    const last = onUpdateCondition.mock.calls.at(-1)![1];
    expect(last.operator).toBe('=');
  });

  it('renders multi-value select for in operator on scalars', async () => {
    const condition = reactive({ id: 'c1', field: 'Name', operator: 'in', value: ['a', 'b'] });
    const { wrapper, onUpdateCondition } = mountRow(condition);
    await nextTick();
    const multi = wrapper.findAll('.el-select').find(s => s.attributes('data-multi') === 'true');
    expect(multi).toBeTruthy();
    await (wrapper.vm as any).onMultiValuesChange(['x', 'y']);
    expect(onUpdateCondition).toHaveBeenCalledWith('c1', { value: ['x', 'y'] });
  });

  it('keeps value editor writable for form-readonly fields', async () => {
    const condition = reactive({ id: 'c1', field: 'DisplayName', operator: '=', value: '' });
    const { wrapper } = mountRow(condition);
    await flushPromises();
    expect(wrapper.find('.f-varchar').exists()).toBe(true);
    const meta = (wrapper.vm as any).fieldMeta;
    expect(meta.isReadonly).toBe(false);
  });

  it('uses manytoone component for relation fields', async () => {
    const condition = reactive({ id: 'c1', field: 'PartnerId', operator: '=', value: null });
    const { wrapper } = mountRow(condition);
    await nextTick();
    expect(wrapper.find('.f-m2o').exists()).toBe(true);
  });

  it('removes the condition row', async () => {
    const condition = reactive({ id: 'c1', field: 'Name', operator: '=', value: '' });
    const { wrapper, onRemoveCondition } = mountRow(condition);
    await wrapper.find('.rm').trigger('click');
    expect(onRemoveCondition).toHaveBeenCalledWith('c1');
  });

  it('renders NULL flag and selection / datetime field components', async () => {
    const nullCond = reactive({ id: 'c1', field: 'Name', operator: 'is', value: null });
    const { wrapper: nullRow } = mountRow(nullCond);
    expect(nullRow.find('.o-null-flag').text()).toBe('NULL');

    const sel = reactive({ id: 'c2', field: 'Status', operator: '=', value: 'a' });
    const { wrapper: selRow } = mountRow(sel);
    expect(selRow.find('.f-selection').exists()).toBe(true);

    const dt = reactive({ id: 'c3', field: 'CreatedAt', operator: '=', value: null });
    const { wrapper: dtRow } = mountRow(dt);
    expect(dtRow.find('.f-dt').exists()).toBe(true);
  });

  it('exposes toView/fromView helpers for manytoone value binding', async () => {
    const condition = reactive({ id: 'c1', field: 'PartnerId', operator: '=', value: 'p1' });
    const { wrapper } = mountRow(condition);
    await nextTick();
    const extras = (wrapper.vm as any).extraProps;
    expect(extras.toView(null)).toBeNull();
    expect(extras.toView({ Id: 'x' })).toEqual({ Id: 'x' });
    expect(extras.toView('y')).toEqual({ Id: 'y' });
    expect(extras.fromView(null)).toBeNull();
    expect(extras.fromView({ Id: 'z' })).toBe('z');
    expect(extras.fromView('w')).toBe('w');
  });

  it('normalizes multiValues from scalar and empty values', async () => {
    const condition = reactive({ id: 'c1', field: 'Name', operator: 'in', value: 'solo' });
    const { wrapper } = mountRow(condition);
    expect((wrapper.vm as any).multiValues).toEqual(['solo']);
    condition.value = '';
    await nextTick();
    expect((wrapper.vm as any).multiValues).toEqual([]);
    condition.value = ['', 'a', null];
    await nextTick();
    expect((wrapper.vm as any).multiValues).toEqual(['a']);
  });

  it('maps every field type to a value editor and placeholder', async () => {
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
      const { wrapper } = mountRow(condition, {
        store: {
          fieldsMetadata,
          getFieldMeta: undefined,
        },
      });
      await nextTick();
      expect(wrapper.find(`.${cls}`).exists(), prop).toBe(true);
      const ph = (wrapper.vm as any).valuePlaceholder;
      expect(typeof ph).toBe('string');
      expect(ph.length).toBeGreaterThan(0);
    }
  });

  it('applies boolean default value and keeps multi-value arrays intact', async () => {
    const condition = reactive({ id: 'c1', field: 'Bool', operator: 'is', value: null });
    const { wrapper, onUpdateCondition } = mountRow(condition, {
      store: {
        fieldsMetadata: {
          Bool: { type: 'boolean', string: 'Bool' },
          Name: { type: 'varchar', string: 'Name' },
        },
      },
    });
    await (wrapper.vm as any).onOperatorChange('=');
    expect(onUpdateCondition.mock.calls.at(-1)![1]).toEqual({ operator: '=', value: false });

    condition.field = 'Name';
    condition.value = ['a', 'b'];
    await (wrapper.vm as any).onOperatorChange('in');
    expect(onUpdateCondition.mock.calls.at(-1)![1]).toEqual({ operator: 'in' });

    await (wrapper.vm as any).onMultiValuesChange('not-array' as any);
    expect(onUpdateCondition).toHaveBeenCalledWith('c1', { value: [] });
  });

  it('falls back getFieldMeta via fieldsMetadata and clears relationStore on scalar fields', async () => {
    const condition = reactive({ id: 'c1', field: 'PartnerId', operator: '=', value: null });
    const { wrapper, store } = mountRow(condition, {
      store: {
        getFieldMeta: undefined,
        getRelationStore: vi.fn(() => ({ destroy: vi.fn() })),
      },
    });
    await nextTick();
    expect((wrapper.vm as any).binding.store.getFieldMeta('PartnerId')?.type).toBe('manytoone');
    expect((wrapper.vm as any).binding.store.getFieldMeta('Missing')).toBeUndefined();
    condition.field = 'Name';
    await nextTick();
    expect((wrapper.vm as any).binding.relationStore).toBeUndefined();
    expect(store.getRelationStore).toHaveBeenCalled();
  });

  it('uses tempId as condition identity when present', async () => {
    const condition = reactive({ id: 'c1', tempId: 'tmp-9', field: 'Name', operator: '=', value: 'x' });
    const { wrapper, onUpdateCondition, onRemoveCondition } = mountRow(condition);
    await (wrapper.vm as any).onFieldChange('Status');
    expect(onUpdateCondition.mock.calls[0][0]).toBe('tmp-9');
    await wrapper.find('.rm').trigger('click');
    expect(onRemoveCondition).toHaveBeenCalledWith('tmp-9');
  });
});

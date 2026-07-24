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
          OCharField: true,
          OTextField: true,
          OIntField: true,
          OBigintField: true,
          ONumberField: true,
          ODecimalField: true,
          OBooleanField: true,
          ODateField: true,
          OTimeField: true,
          OJsonobjectField: true,
          OManyToOneRefField: true,
          OBinaryField: true,
          OImageField: true,
        },
      },
    });
    return { wrapper, onUpdateCondition, onRemoveCondition, store };
  }

  it('patches field change with first operator and clears value', async () => {
    const condition = reactive({ id: 'c1', field: 'Name', operator: '=', value: 'x' });
    const { wrapper, onUpdateCondition } = mountRow(condition);
    const fieldSelect = wrapper.findAll('.el-select')[0]!;
    // Directly invoke handler through component vm
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
});

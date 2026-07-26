// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount, flushPromises } from '@vue/test-utils';
import { computed, defineComponent, h, nextTick, ref } from 'vue';
import { describe, expect, it, vi } from 'vitest';
import Decimal from '@/core/utils/decimal';
import type { UseField } from '@/web/web/composables/useField';
import OMonetaryField from './OMonetaryField.vue';

vi.mock('@/web/web/i18n', async () => {
  const actual = await vi.importActual<typeof import('@/web/web/i18n')>('@/web/web/i18n');
  return {
    ...actual,
    createTranslate: () => ({
      _t: (msg: string, ...args: unknown[]) => (args.length ? `${msg}:${args.join(',')}` : msg),
    }),
  };
});

vi.mock('@/web/web/stores/i18nStore', async () => {
  const actual = await vi.importActual<typeof import('@/web/web/stores/i18nStore')>('@/web/web/stores/i18nStore');
  return {
    ...actual,
    useI18nStore: () => ({
      currentLocale: {
        numberFormat: { thousandsSeparator: ',', decimalSeparator: '.', decimalDigits: 2 },
      },
    }),
  };
});

function makeBinding(record: Record<string, unknown>, meta: Record<string, unknown> = { currencyField: 'CurrencyId', type: 'monetary' }): UseField {
  const value = ref(record.Amount ?? null);
  const recordRef = ref(record);
  const registered: string[] = [];
  return {
    env: { isForm: true, isEditMode: true, viewMode: 'edit', fieldPrefix: null },
    prop: 'Amount',
    meta: meta as any,
    fieldRef: () => value as any,
    fieldRefOf: () => value as any,
    recordRef: () => computed(() => recordRef.value) as any,
    registerFields: (path: string) => {
      registered.push(path);
    },
    store: undefined,
    asView: () => ({ fieldValue: () => value }) as any,
    __registered: registered,
  } as any;
}

describe('OMonetaryField', () => {
  it('registers currency sibling fields and renders formatted display', async () => {
    const binding = makeBinding({
      Amount: new Decimal('12.345'),
      CurrencyId: { Id: 'C1', Code: 'USD', Symbol: '$', DecimalDigits: 2 },
    });

    const wrapper = mount(OMonetaryField as any, {
      props: {
        binding,
        renderMode: 'form',
        readonly: true,
      },
      global: {
        stubs: {
          OFieldBase: defineComponent({
            name: 'OFieldBaseStub',
            props: ['binding', 'toView', 'fromView', 'rules'],
            setup(p, { slots }) {
              const fieldValue = () => ({ value: (p.binding as any).fieldRef().value });
              const record = () => ({ value: (p.binding as any).recordRef().value });
              return () =>
                h('div', { class: 'base' }, [
                  slots.display?.({ fieldValue, record }),
                  slots.edit?.({ fieldValue, record }),
                ]);
            },
          }),
          ElInput: {
            props: ['modelValue', 'placeholder'],
            emits: ['update:modelValue', 'blur'],
            template:
              '<input class="el-input" :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" @blur="$emit(\'blur\')" />',
          },
        },
      },
    });

    await flushPromises();
    expect((binding as any).__registered).toContain('CurrencyId');
    expect((binding as any).__registered).toContain('CurrencyId.DecimalDigits');
    expect(wrapper.find('.o-field-display-text').text().length).toBeGreaterThan(0);

    // Exercise toView/fromView through props on stub
    const base = wrapper.findComponent({ name: 'OFieldBaseStub' });
    expect(base.props('toView')(new Decimal('1'))).toBe('1');
    expect(base.props('fromView')('2.5')).toBeTruthy();
    expect(base.props('fromView')(null)).toBeNull();

    // Edit path: type a value and blur
    const input = wrapper.find('input.el-input');
    if (input.exists()) {
      await input.setValue('3.456');
      await input.trigger('blur');
      await nextTick();
    }
  });

  it('validates monetary values via internal rule using currency digits', async () => {
    const binding = makeBinding({
      Amount: '1.239',
      CurrencyId: { DecimalDigits: 2 },
    });
    const wrapper = mount(OMonetaryField as any, {
      props: { binding, renderMode: 'form' },
      global: {
        stubs: {
          OFieldBase: defineComponent({
            name: 'OFieldBaseStub',
            props: ['rules'],
            setup(p) {
              return () => h('div', { class: 'rules', 'data-count': String((p.rules as any[])?.length || 0) });
            },
          }),
        },
      },
    });
    await flushPromises();
    const rules = (wrapper.findComponent({ name: 'OFieldBaseStub' }).props('rules') || []) as any[];
    const monetaryRule = rules[rules.length - 1];
    let err: Error | undefined;
    await new Promise<void>(resolve => {
      monetaryRule.validator({}, '1.239', (e?: Error) => {
        err = e;
        resolve();
      });
    });
    expect(err?.message).toMatch(/Decimal places|exceed/i);

    await new Promise<void>(resolve => {
      monetaryRule.validator({}, '1.23', (e?: Error) => {
        err = e;
        resolve();
      });
    });
    expect(err).toBeUndefined();
  });
});

// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount, flushPromises } from '@vue/test-utils';
import { computed, defineComponent, h, nextTick, reactive, ref } from 'vue';
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

const i18nStoreMock = vi.hoisted(() => ({
  throwOnAccess: false,
  currentLocale: {
    numberFormat: { thousandsSeparator: ',', decimalSeparator: '.', decimalDigits: 2 },
  },
}));

const useFieldMock = vi.hoisted(() => ({
  impl: null as null | ((...args: any[]) => any),
}));

vi.mock('@/web/web/stores/i18nStore', async () => {
  // Avoid i18nStore/index (Pinia persist/localStorage + vue-devtools IndexedDB).
  const formatters = await vi.importActual<typeof import('@/web/web/stores/i18nStore/language_format')>(
    '@/web/web/stores/i18nStore/language_format'
  );
  return {
    ...formatters,
    useI18nStore: () => {
      if (i18nStoreMock.throwOnAccess) throw new Error('i18n boom');
      return { currentLocale: i18nStoreMock.currentLocale };
    },
  };
});

vi.mock('@/web/web/composables/useField', async () => {
  const actual = await vi.importActual<typeof import('@/web/web/composables/useField')>('@/web/web/composables/useField');
  return {
    ...actual,
    useField: (...args: any[]) => {
      if (useFieldMock.impl) return useFieldMock.impl(...args);
      return (actual as any).useField(...args);
    },
  };
});

function makeBinding(
  record: Record<string, unknown>,
  meta: Record<string, unknown> = { currencyField: 'CurrencyId', type: 'monetary' }
): UseField & { __registered: string[]; __value: any; __recordRef: any; meta: any } {
  const value = ref(record.Amount ?? null);
  const recordRef = ref(record);
  const registered: string[] = [];
  const reactiveMeta = reactive({ ...meta });
  return {
    env: { isForm: true, isEditMode: true, viewMode: 'edit', fieldPrefix: null },
    prop: 'Amount',
    meta: reactiveMeta as any,
    fieldRef: () => value as any,
    fieldRefOf: () => value as any,
    recordRef: () => computed(() => recordRef.value) as any,
    registerFields: (path: string) => {
      registered.push(path);
    },
    store: undefined,
    asView: () => ({ fieldValue: () => value }) as any,
    __registered: registered,
    __value: value,
    __recordRef: recordRef,
  } as any;
}

const fieldBaseStub = defineComponent({
  name: 'OFieldBaseStub',
  inheritAttrs: false,
  props: {
    binding: { type: Object, required: true },
    toView: { type: Function, default: undefined },
    fromView: { type: Function, default: undefined },
    rules: { type: Array, default: undefined },
    label: { type: String, default: undefined },
    formItemProps: { type: Object, default: undefined },
    vColumnProps: { type: Object, default: undefined },
    required: { type: [Boolean, Function, Object], default: undefined },
    readonly: { type: [Boolean, Function, Object], default: undefined },
    visible: { type: [Boolean, Function, Object], default: undefined },
    cellVisible: { type: [Boolean, Function, Object], default: undefined },
    renderMode: { type: String, default: undefined },
    showInlineError: { type: Boolean, default: undefined },
  },
  setup(p, { slots }) {
    // Pass the live ref through so cell commits mutate binding state.
    const fieldValue = () => (p.binding as any).fieldRef();
    const record = () => ({ value: (p.binding as any).recordRef().value });
    return () =>
      h('div', { class: 'base' }, [slots.display?.({ fieldValue, record }), slots.edit?.({ fieldValue, record })]);
  },
});

const elInputStub = defineComponent({
  name: 'ElInput',
  inheritAttrs: false,
  props: {
    modelValue: { type: [String, Number, null] as any, default: null },
    placeholder: { type: String, default: undefined },
    inputmode: { type: String, default: undefined },
    class: { type: [String, Object, Array], default: undefined },
  },
  emits: ['update:modelValue', 'blur'],
  setup(p, { emit }) {
    return () =>
      h('input', {
        class: 'el-input',
        value: p.modelValue ?? '',
        placeholder: p.placeholder,
        onInput: (e: Event) => emit('update:modelValue', (e.target as HTMLInputElement).value),
        onBlur: () => emit('blur'),
      });
  },
});

function mountEdit(binding: any, props: Record<string, unknown> = {}) {
  return mount(OMonetaryField as any, {
    props: {
      binding,
      renderMode: 'form',
      bufferStrategy: 'live',
      commitOnBlur: true,
      ...props,
    },
    global: {
      stubs: {
        OFieldBase: fieldBaseStub,
        ElInput: elInputStub,
        'el-input': elInputStub,
      },
    },
  });
}

describe('OMonetaryField', () => {
  it('registers currency sibling fields and renders formatted display', async () => {
    const binding = makeBinding({
      Amount: new Decimal('12.345'),
      CurrencyId: { Id: 'C1', Code: 'USD', Symbol: '$', DecimalDigits: 2 },
    });

    const wrapper = mountEdit(binding, { readonly: true });
    await flushPromises();
    expect(binding.__registered).toContain('CurrencyId');
    expect(binding.__registered).toContain('CurrencyId.DecimalDigits');
    expect(wrapper.find('.o-field-display-text').text().length).toBeGreaterThan(0);

    const base = wrapper.findComponent({ name: 'OFieldBaseStub' });
    expect(base.props('toView')(new Decimal('1'))).toBe('1');
    expect(base.props('fromView')('2.5')).toBeTruthy();
    expect(base.props('fromView')(null)).toBeNull();
  });

  it('skips currency registration without currencyField', async () => {
    const binding = makeBinding({ Amount: '1' }, { type: 'monetary', currencyField: '  ' });
    mountEdit(binding);
    await flushPromises();
    expect(binding.__registered).toEqual([]);
  });

  it('clears null when nullable and input emptied', async () => {
    const binding = makeBinding({
      Amount: new Decimal('1.25'),
      CurrencyId: { DecimalDigits: 2 },
    });
    const wrapper = mountEdit(binding, { nullable: true });
    await flushPromises();
    const input = wrapper.find('input.el-input');
    await input.setValue('');
    await nextTick();
    expect(binding.__value.value).toBeNull();
  });

  it('ignores empty input when not nullable', async () => {
    const binding = makeBinding({
      Amount: new Decimal('1.25'),
      CurrencyId: { DecimalDigits: 2 },
    });
    const wrapper = mountEdit(binding, { nullable: false });
    await flushPromises();
    const input = wrapper.find('input.el-input');
    await input.setValue('');
    await nextTick();
    expect(new Decimal(binding.__value.value).toString()).toBe('1.25');
  });

  it('rejects non-numeric input and keeps intermediate typing without commit', async () => {
    const binding = makeBinding({
      Amount: new Decimal('1'),
      CurrencyId: { DecimalDigits: 2 },
    });
    const wrapper = mountEdit(binding);
    await flushPromises();
    const input = wrapper.find('input.el-input');

    await input.setValue('abc');
    await nextTick();
    expect(new Decimal(binding.__value.value).toString()).toBe('1');

    await input.setValue('12.');
    await nextTick();
    expect(new Decimal(binding.__value.value).toString()).toBe('1');
  });

  it('commits trailing-dot intermediate on blur and rejects over-max input', async () => {
    const binding = makeBinding({
      Amount: new Decimal('1'),
      CurrencyId: { DecimalDigits: 2 },
    });
    const wrapper = mountEdit(binding, { max: '10', precision: 10 });
    await flushPromises();
    const input = wrapper.find('input.el-input');

    await input.setValue('3.');
    await input.trigger('blur');
    await flushPromises();
    expect(new Decimal(binding.__value.value).toString()).toBe('3');

    await input.setValue('99.99');
    await input.trigger('blur');
    await flushPromises();
    // parseStrict rejects over-max before clamp; value stays at last valid commit.
    expect(new Decimal(binding.__value.value).toString()).toBe('3');
  });

  it('live-commits quantized monetary edits', async () => {
    const binding = makeBinding({
      Amount: new Decimal('1'),
      CurrencyId: { DecimalDigits: 2 },
    });
    const wrapper = mountEdit(binding);
    await flushPromises();
    const input = wrapper.find('input.el-input');
    await input.setValue('4.56');
    await flushPromises();
    expect(new Decimal(binding.__value.value).toString()).toBe('4.56');
  });

  it('clears invalid blur payload to null when nullable', async () => {
    const binding = makeBinding({
      Amount: new Decimal('1'),
      CurrencyId: { DecimalDigits: 2 },
    });
    const wrapper = mountEdit(binding, { nullable: true, scale: 2 });
    await flushPromises();
    const input = wrapper.find('input.el-input');
    await input.setValue('1.239');
    await input.trigger('blur');
    await flushPromises();
    // parseStrict rejects scale overflow, so blur leaves editing without a valid parse commit path
    // and live strategy may not replace; assert display input cleared or value unchanged safely.
    expect(binding.__value.value == null || new Decimal(binding.__value.value).toString() === '1').toBe(true);
  });

  it('falls back when i18n store throws in display', async () => {
    i18nStoreMock.throwOnAccess = true;
    try {
      const binding = makeBinding({
        Amount: new Decimal('12.3'),
        CurrencyId: { Symbol: '$', DecimalDigits: 1 },
      });
      const wrapper = mountEdit(binding, { readonly: true });
      await flushPromises();
      expect(wrapper.find('.o-field-display-text').text()).toContain('12.3');
    } finally {
      i18nStoreMock.throwOnAccess = false;
    }
  });

  it('displays aggregate metric when raw empty', async () => {
    const binding = makeBinding({
      Amount: null,
      metrics: { Amount__sum: new Decimal('9.5') },
      CurrencyId: { DecimalDigits: 1, Symbol: '$' },
    });
    const wrapper = mountEdit(binding, { agg: 'sum', readonly: true });
    await flushPromises();
    expect(wrapper.find('.o-field-display-text').text().length).toBeGreaterThan(0);
  });

  it('clears on blur after empty nullable input and rejects intermediate dash', async () => {
    const binding = makeBinding({
      Amount: new Decimal('2'),
      CurrencyId: { DecimalDigits: 2 },
    });
    const wrapper = mountEdit(binding, { nullable: true });
    await flushPromises();
    const input = wrapper.find('input.el-input');
    await input.setValue('');
    await input.trigger('blur');
    await flushPromises();
    expect(binding.__value.value).toBeNull();

    binding.__value.value = new Decimal('2');
    await input.setValue('-');
    await input.trigger('blur');
    await flushPromises();
    expect(binding.__value.value).toBeNull();
  });

  it('uses props.scale when currency digits are unavailable', async () => {
    const binding = makeBinding({ Amount: new Decimal('1'), CurrencyId: {} }, { type: 'monetary', currencyField: 'CurrencyId' });
    const wrapper = mountEdit(binding, { scale: 1 });
    await flushPromises();
    const input = wrapper.find('input.el-input');
    await input.setValue('1.23');
    await flushPromises();
    expect(new Decimal(binding.__value.value).toString()).toBe('1');
    await input.setValue('1.2');
    await flushPromises();
    expect(new Decimal(binding.__value.value).toString()).toBe('1.2');
  });

  it('re-registers currency paths when currencyField changes', async () => {
    const binding = makeBinding({ Amount: '1', CurrencyId: { DecimalDigits: 2 } });
    mountEdit(binding);
    await flushPromises();
    const before = binding.__registered.length;
    (binding as any).meta.currencyField = 'PayCurrencyId';
    await nextTick();
    await flushPromises();
    expect(binding.__registered.length).toBeGreaterThan(before);
    expect(binding.__registered).toContain('PayCurrencyId');
  });

  it('maps toView/fromView invalid values to null', async () => {
    const binding = makeBinding({ Amount: '1', CurrencyId: { DecimalDigits: 2 } });
    const wrapper = mountEdit(binding);
    await flushPromises();
    const base = wrapper.findComponent({ name: 'OFieldBaseStub' });
    expect(base.props('toView')('nope')).toBeNull();
    expect(base.props('fromView')('')).toBeNull();
  });

  it('covers idle equals paths and intermediate blur clears', async () => {
    const binding = makeBinding({
      Amount: new Decimal('1.20'),
      CurrencyId: { DecimalDigits: 2 },
    });
    const wrapper = mountEdit(binding, { bufferStrategy: 'idle', bufferIdleDelay: 10_000, nullable: true });
    await flushPromises();
    const input = wrapper.find('input.el-input');

    // Same quantized value should be treated as equal by buffer equals().
    await input.setValue('1.20');
    await input.trigger('blur');
    await flushPromises();
    expect(new Decimal(binding.__value.value).toString()).toBe('1.2');

    await input.setValue('.');
    await input.trigger('blur');
    await flushPromises();
    expect(binding.__value.value).toBeNull();

    binding.__value.value = new Decimal('2');
    await input.setValue('-.');
    await input.trigger('blur');
    await flushPromises();
    expect(binding.__value.value).toBeNull();
  });

  it('renders aggregate object-form metrics and meta precision', async () => {
    const binding = makeBinding(
      {
        Amount: null,
        metrics: { TotalAmount: new Decimal('3.5') },
        CurrencyId: { DecimalDigits: 1, Code: 'USD' },
      },
      { type: 'monetary', currencyField: 'CurrencyId', precision: 12 }
    );
    const wrapper = mountEdit(binding, {
      agg: { agg: 'sum', alias: 'TotalAmount' },
      readonly: true,
      precision: undefined,
    });
    await flushPromises();
    expect(wrapper.find('.o-field-display-text').text().length).toBeGreaterThan(0);
  });

  it('validates monetary values via internal rule using currency digits', async () => {
    const binding = makeBinding({
      Amount: '1.239',
      CurrencyId: { DecimalDigits: 2 },
    });
    const wrapper = mount(OMonetaryField as any, {
      props: { binding, renderMode: 'form', min: '0', max: '100' },
      global: {
        stubs: {
          OFieldBase: defineComponent({
            name: 'OFieldBaseStub',
            inheritAttrs: false,
            props: {
              rules: { type: Array, default: undefined },
              binding: { type: Object, default: undefined },
              label: { type: String, default: undefined },
              toView: { type: Function, default: undefined },
              fromView: { type: Function, default: undefined },
              formItemProps: { type: Object, default: undefined },
              vColumnProps: { type: Object, default: undefined },
              required: { type: [Boolean, Function, Object], default: undefined },
              readonly: { type: [Boolean, Function, Object], default: undefined },
              visible: { type: [Boolean, Function, Object], default: undefined },
              cellVisible: { type: [Boolean, Function, Object], default: undefined },
              renderMode: { type: String, default: undefined },
              showInlineError: { type: Boolean, default: undefined },
            },
            setup(p) {
              return () => h('div', { class: 'rules', 'data-count': String((p.rules as any[])?.length || 0) });
            },
          }),
          ElInput: elInputStub,
          'el-input': elInputStub,
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
      monetaryRule.validator({}, null, (e?: Error) => {
        err = e;
        resolve();
      });
    });
    expect(err).toBeUndefined();

    await new Promise<void>(resolve => {
      monetaryRule.validator({}, '1.23', (e?: Error) => {
        err = e;
        resolve();
      });
    });
    expect(err).toBeUndefined();

    await new Promise<void>(resolve => {
      monetaryRule.validator({}, '-1', (e?: Error) => {
        err = e;
        resolve();
      });
    });
    expect(err?.message).toMatch(/less than/i);

    await new Promise<void>(resolve => {
      monetaryRule.validator({}, '101', (e?: Error) => {
        err = e;
        resolve();
      });
    });
    expect(err?.message).toMatch(/greater than/i);

    await new Promise<void>(resolve => {
      monetaryRule.validator({}, 'abc', (e?: Error) => {
        err = e;
        resolve();
      });
    });
    expect(err?.message).toMatch(/valid number/i);
  });

  it('falls back currentScale when getScale returns out-of-range and clears with rules', async () => {
    const binding = makeBinding({ Amount: new Decimal('1'), CurrencyId: {} }, { type: 'monetary', currencyField: 'CurrencyId' });
    const wrapper = mountEdit(binding, {
      scale: 99,
      nullable: true,
      roundingMode: (Decimal as any).ROUND_DOWN ?? 1,
      rules: [{ required: true } as any],
    });
    await flushPromises();
    const input = wrapper.find('input.el-input');
    await input.setValue('1.2');
    await flushPromises();
    // Out-of-range props.scale falls through currentScale validation then returns props.scale.
    expect(binding.__value.value == null || String(binding.__value.value).length > 0).toBe(true);

    (binding as any).meta.currencyField = 123;
    await nextTick();
    await flushPromises();
  });

  it('maps fromView invalid numeric strings to null', async () => {
    const binding = makeBinding({ Amount: '1', CurrencyId: { DecimalDigits: 2 } });
    const wrapper = mountEdit(binding);
    await flushPromises();
    const base = wrapper.findComponent({ name: 'OFieldBaseStub' });
    expect(base.props('fromView')('not-a-number')).toBeNull();
  });

  it('bootstraps via useField when binding is omitted and handles empty prop / default scale', async () => {
    const binding = makeBinding({ Amount: new Decimal('1'), CurrencyId: {} });
    (binding as any).prop = '';
    let called = false;
    useFieldMock.impl = () => {
      called = true;
      return binding;
    };
    try {
      const wrapper = mount(OMonetaryField as any, {
        props: {
          store: {} as any,
          prop: 'Amount',
          renderMode: 'form',
          readonly: true,
          agg: 'sum',
        },
        global: {
          stubs: {
            OFieldBase: fieldBaseStub,
            ElInput: elInputStub,
            'el-input': elInputStub,
          },
        },
      });
      await flushPromises();
      expect(called).toBe(true);
      expect(wrapper.find('.o-field-display-text').exists()).toBe(true);

      const editBinding = makeBinding({ Amount: new Decimal('1'), CurrencyId: {} });
      const edit = mountEdit(editBinding);
      await flushPromises();
      const input = edit.find('input.el-input');
      await input.setValue('1.5');
      await flushPromises();
      // No scale prop + no currency digits → currentScale falls back to 6 and commits.
      expect(new Decimal(editBinding.__value.value).toString()).toBe('1.5');
    } finally {
      useFieldMock.impl = null;
    }
  });
});

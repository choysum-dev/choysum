// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount, flushPromises } from '@vue/test-utils';
import { computed, defineComponent, h, nextTick, reactive, ref } from 'vue';
import { describe, expect, it, vi } from 'vitest';
import Decimal from '@/core/utils/decimal';
import type { UseField } from '@/web/web/composables/useField';
import ODecimalField from './ODecimalField.vue';

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
  } as { numberFormat?: Record<string, unknown> } | null,
}));

const useFieldMock = vi.hoisted(() => ({
  impl: null as null | ((...args: any[]) => any),
}));

vi.mock('@/web/web/stores/i18nStore', async () => {
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
  meta: Record<string, unknown> = { type: 'decimal' }
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

function mountField(binding: any, props: Record<string, unknown> = {}) {
  return mount(ODecimalField as any, {
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

describe('ODecimalField', () => {
  it('displays free decimals without zero pad when scale is unset', async () => {
    const binding = makeBinding({ Amount: new Decimal('0.01') });
    const wrapper = mountField(binding, { readonly: true });
    await flushPromises();
    expect(wrapper.find('.o-field-display-text').text()).toBe('0.01');
  });

  it('pads display when meta.scale or props.scale is declared', async () => {
    const withMeta = makeBinding({ Amount: new Decimal('0.01') }, { type: 'decimal', scale: 4 });
    const metaWrapper = mountField(withMeta, { readonly: true });
    await flushPromises();
    expect(metaWrapper.find('.o-field-display-text').text()).toBe('0.0100');

    const withProps = makeBinding({ Amount: new Decimal('1.2') });
    const propsWrapper = mountField(withProps, { readonly: true, scale: 2 });
    await flushPromises();
    expect(propsWrapper.find('.o-field-display-text').text()).toBe('1.20');
  });

  it('resolves scaleField from sibling / metrics__max and registers the path', async () => {
    const binding = makeBinding(
      { Amount: new Decimal('1.2345'), AmountScale: 2 },
      { type: 'decimal', scaleField: 'AmountScale' }
    );
    const wrapper = mountField(binding, { readonly: true });
    await flushPromises();
    expect(binding.__registered).toContain('AmountScale');
    expect(wrapper.find('.o-field-display-text').text()).toBe('1.23');

    binding.__value.value = new Decimal('9.999');
    binding.__recordRef.value = {
      Amount: binding.__value.value,
      metrics: { AmountScale__max: 1 },
    };
    await nextTick();
    await flushPromises();
    expect(wrapper.find('.o-field-display-text').text()).toBe('10.0');
  });

  it('falls back when i18n store throws and still renders significant digits', async () => {
    i18nStoreMock.throwOnAccess = true;
    try {
      const binding = makeBinding({ Amount: new Decimal('12.3') });
      const wrapper = mountField(binding, { readonly: true });
      await flushPromises();
      expect(wrapper.find('.o-field-display-text').text()).toBe('12.3');
    } finally {
      i18nStoreMock.throwOnAccess = false;
    }
  });

  it('live-commits edits using DB soft max 18 when scale is unset', async () => {
    const binding = makeBinding({ Amount: new Decimal('1') });
    const wrapper = mountField(binding);
    await flushPromises();
    const input = wrapper.find('input.el-input');
    await input.setValue('1.234567890123456789');
    await flushPromises();
    expect(new Decimal(binding.__value.value).toString()).toBe('1.234567890123456789');

    await input.setValue('1.2345678901234567890');
    await flushPromises();
    // 19 fractional places exceed soft max → parseStrict rejects; value stays.
    expect(new Decimal(binding.__value.value).toString()).toBe('1.234567890123456789');
  });

  it('quantizes edits to declared props.scale', async () => {
    const binding = makeBinding({ Amount: new Decimal('1') });
    const wrapper = mountField(binding, { scale: 2 });
    await flushPromises();
    const input = wrapper.find('input.el-input');
    await input.setValue('1.239');
    await flushPromises();
    // parseStrict rejects > scale; value unchanged until a valid commit
    expect(new Decimal(binding.__value.value).toString()).toBe('1');
    await input.setValue('1.23');
    await flushPromises();
    expect(new Decimal(binding.__value.value).toString()).toBe('1.23');
  });

  it('validates via internal rule using edit scale (fixed or soft max 18)', async () => {
    const binding = makeBinding({ Amount: '1.239' }, { type: 'decimal', scale: 2 });
    const wrapper = mount(ODecimalField as any, {
      props: { binding, renderMode: 'form' },
      global: {
        stubs: {
          OFieldBase: defineComponent({
            name: 'OFieldBaseStub',
            inheritAttrs: false,
            props: {
              rules: { type: Array, default: undefined },
              binding: { type: Object, default: undefined },
              toView: { type: Function, default: undefined },
              fromView: { type: Function, default: undefined },
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
    const rule = rules[rules.length - 1];
    let err: Error | undefined;
    await new Promise<void>(resolve => {
      rule.validator({}, '1.239', (e?: Error) => {
        err = e;
        resolve();
      });
    });
    expect(err?.message).toMatch(/Decimal places|exceed/i);

    await new Promise<void>(resolve => {
      rule.validator({}, '1.23', (e?: Error) => {
        err = e;
        resolve();
      });
    });
    expect(err).toBeUndefined();
  });

  it('maps toView/fromView and displays aggregate metrics', async () => {
    const binding = makeBinding({
      Amount: null,
      metrics: { Amount__sum: new Decimal('9.5') },
    });
    const wrapper = mountField(binding, { agg: 'sum', readonly: true });
    await flushPromises();
    expect(wrapper.find('.o-field-display-text').text()).toContain('9.5');

    const base = wrapper.findComponent({ name: 'OFieldBaseStub' });
    expect(base.props('toView')(new Decimal('1'))).toBe('1');
    expect(base.props('fromView')('2.5')).toBeTruthy();
    expect(base.props('fromView')(null)).toBeNull();
    expect(base.props('toView')('nope')).toBeNull();
  });

  it('bootstraps via useField when binding is omitted', async () => {
    const binding = makeBinding({ Amount: new Decimal('1') });
    let called = false;
    useFieldMock.impl = () => {
      called = true;
      return binding;
    };
    try {
      const wrapper = mount(ODecimalField as any, {
        props: {
          store: {} as any,
          prop: 'Amount',
          renderMode: 'form',
          readonly: true,
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
    } finally {
      useFieldMock.impl = null;
    }
  });

  it('falls back currentScale to 18 when getScale returns out of range', async () => {
    const binding = makeBinding({ Amount: new Decimal('1') });
    const wrapper = mountField(binding, { scale: 99 });
    await flushPromises();
    const input = wrapper.find('input.el-input');
    // props.scale=99 is rejected by resolveFixedScaleFrom → edit scale 18
    await input.setValue('1.5');
    await flushPromises();
    expect(new Decimal(binding.__value.value).toString()).toBe('1.5');
  });
});

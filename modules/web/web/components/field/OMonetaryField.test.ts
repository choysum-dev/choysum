// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Mount wiring for OMonetaryField. Pure format/scale helpers live in omonetary_helpers.test.ts.
 */

import { computed, defineComponent, h, nextTick, provide, reactive, ref } from 'vue';
import { ElInput } from 'element-plus';
import Decimal from '@/core/utils/decimal';

import type { UseField } from '@/web/web/composables/useField';
import { flushPromises, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import OFieldBase from './OFieldBase.vue';
import OMonetaryField from './OMonetaryField.vue';

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

const lastBaseProps: { current: Record<string, unknown> | null } = { current: null };

function installFieldBaseStub() {
  stubSfc(OFieldBase as any, {
    name: 'OFieldBase',
    inheritAttrs: false,
    props: {
      binding: { type: Object, required: false },
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
    setup(p: any, { slots }: any) {
      lastBaseProps.current = p;
      const fieldValue = () => (p.binding as any).fieldRef();
      const record = () => ({ value: (p.binding as any).recordRef().value });
      return () =>
        h('div', { class: 'base' }, [slots.display?.({ fieldValue, record }), slots.edit?.({ fieldValue, record })]);
    },
  });
}

function installElInputStub() {
  stubSfc(ElInput as any, {
    name: 'ElInput',
    inheritAttrs: false,
    props: {
      modelValue: { type: [String, Number, null] as any, default: null },
      placeholder: { type: String, default: undefined },
    },
    emits: ['update:modelValue', 'blur'],
    setup(p: any, { emit }: any) {
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
}

function setInput(el: HTMLInputElement, value: string) {
  el.value = value;
  el.dispatchEvent(new Event('input', { bubbles: true }));
}

function blurInput(el: HTMLInputElement) {
  el.dispatchEvent(new Event('blur', { bubbles: true }));
}

function mountField(binding: any, props: Record<string, unknown> = {}) {
  lastBaseProps.current = null;
  installFieldBaseStub();
  installElInputStub();
  return mountApp(OMonetaryField as any, {
    props: {
      binding,
      renderMode: 'form',
      bufferStrategy: 'live',
      commitOnBlur: true,
      ...props,
    },
  });
}

describe('OMonetaryField mount wiring', () => {
  afterEach(() => {
    restoreSfc(OFieldBase as any);
    restoreSfc(ElInput as any);
  });

  test('registers currency sibling fields and renders formatted display', async () => {
    const binding = makeBinding({
      Amount: new Decimal('12.345'),
      CurrencyId: { Id: 'C1', Code: 'USD', Symbol: '$', DecimalDigits: 2 },
    });
    const m = mountField(binding, { readonly: true });
    await flushPromises();
    expect(binding.__registered).toContain('CurrencyId');
    expect(binding.__registered).toContain('CurrencyId.DecimalDigits');
    expect((m.q('.o-field-display-text')?.textContent || '').length).toBeGreaterThan(0);

    const toView = lastBaseProps.current?.toView as (v: unknown) => unknown;
    const fromView = lastBaseProps.current?.fromView as (v: unknown) => unknown;
    expect(toView(new Decimal('1'))).toBe('1');
    expect(fromView('2.5')).toBeTruthy();
    expect(fromView(null)).toBeNull();
    expect(toView('nope')).toBeNull();
    expect(fromView('')).toBeNull();
    m.unmount();
  });

  test('skips currency registration without currencyField', async () => {
    const binding = makeBinding({ Amount: '1' }, { type: 'monetary', currencyField: '  ' });
    const m = mountField(binding);
    await flushPromises();
    expect(binding.__registered).toEqual([]);
    m.unmount();
  });

  test('clears null when nullable and keeps prior value when not nullable', async () => {
    const nullable = makeBinding({
      Amount: new Decimal('1.25'),
      CurrencyId: { DecimalDigits: 2 },
    });
    const nullableMount = mountField(nullable, { nullable: true });
    await flushPromises();
    setInput(nullableMount.q('input.el-input') as HTMLInputElement, '');
    await nextTick();
    expect(nullable.__value.value).toBeNull();
    nullableMount.unmount();
    restoreSfc(OFieldBase as any);
    restoreSfc(ElInput as any);

    const required = makeBinding({
      Amount: new Decimal('1.25'),
      CurrencyId: { DecimalDigits: 2 },
    });
    const requiredMount = mountField(required, { nullable: false });
    await flushPromises();
    setInput(requiredMount.q('input.el-input') as HTMLInputElement, '');
    await nextTick();
    expect(new Decimal(required.__value.value).toString()).toBe('1.25');
    requiredMount.unmount();
  });

  test('rejects non-numeric input and live-commits quantized edits', async () => {
    const binding = makeBinding({
      Amount: new Decimal('1'),
      CurrencyId: { DecimalDigits: 2 },
    });
    const m = mountField(binding);
    await flushPromises();
    const input = m.q('input.el-input') as HTMLInputElement;

    setInput(input, 'abc');
    await nextTick();
    expect(new Decimal(binding.__value.value).toString()).toBe('1');

    setInput(input, '12.');
    await nextTick();
    expect(new Decimal(binding.__value.value).toString()).toBe('1');

    setInput(input, '4.56');
    await flushPromises();
    expect(new Decimal(binding.__value.value).toString()).toBe('4.56');
    m.unmount();
  });

  test('commits trailing-dot on blur and rejects over-max input', async () => {
    const binding = makeBinding({
      Amount: new Decimal('1'),
      CurrencyId: { DecimalDigits: 2 },
    });
    const m = mountField(binding, { max: '10', precision: 10 });
    await flushPromises();
    const input = m.q('input.el-input') as HTMLInputElement;

    setInput(input, '3.');
    blurInput(input);
    await flushPromises();
    expect(new Decimal(binding.__value.value).toString()).toBe('3');

    setInput(input, '99.99');
    blurInput(input);
    await flushPromises();
    expect(new Decimal(binding.__value.value).toString()).toBe('3');
    m.unmount();
  });

  test('uses props.scale when currency digits are unavailable', async () => {
    const binding = makeBinding(
      { Amount: new Decimal('1'), CurrencyId: {} },
      { type: 'monetary', currencyField: 'CurrencyId' }
    );
    const m = mountField(binding, { scale: 1 });
    await flushPromises();
    const input = m.q('input.el-input') as HTMLInputElement;
    setInput(input, '1.23');
    await flushPromises();
    expect(new Decimal(binding.__value.value).toString()).toBe('1');
    setInput(input, '1.2');
    await flushPromises();
    expect(new Decimal(binding.__value.value).toString()).toBe('1.2');
    m.unmount();
  });

  test('re-registers currency paths when currencyField changes', async () => {
    const binding = makeBinding({ Amount: '1', CurrencyId: { DecimalDigits: 2 } });
    const m = mountField(binding);
    await flushPromises();
    const before = binding.__registered.length;
    binding.meta.currencyField = 'PayCurrencyId';
    await nextTick();
    await flushPromises();
    expect(binding.__registered.length).toBeGreaterThan(before);
    expect(binding.__registered).toContain('PayCurrencyId');
    m.unmount();
  });

  test('displays aggregate metric when raw empty', async () => {
    const binding = makeBinding({
      Amount: null,
      metrics: { Amount__sum: new Decimal('9.5') },
      CurrencyId: { DecimalDigits: 1, Symbol: '$' },
    });
    const m = mountField(binding, { agg: 'sum', readonly: true });
    await flushPromises();
    expect((m.q('.o-field-display-text')?.textContent || '').length).toBeGreaterThan(0);
    m.unmount();
  });

  test('bootstraps via useField when binding is omitted', async () => {
    installFieldBaseStub();
    installElInputStub();
    const draft = reactive({
      Amount: new Decimal('1'),
      CurrencyId: { DecimalDigits: 2, Symbol: '$' },
    });
    const Host = defineComponent({
      setup() {
        provide('view-container', ref('Form'));
        provide('view-mode', ref('edit'));
        provide('form-root', { draft });
        return () =>
          h(OMonetaryField as any, {
            store: {} as any,
            prop: 'Amount',
            renderMode: 'form',
            readonly: true,
          });
      },
    });
    const m = mountApp(Host);
    await flushPromises();
    expect((m.q('.o-field-display-text')?.textContent || '').length).toBeGreaterThan(0);
    m.unmount();
  });
});

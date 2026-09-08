// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Mount wiring for ODecimalField. Pure format/scale helpers live in odecimal_helpers.test.ts.
 */

import { computed, defineComponent, h, nextTick, provide, reactive, ref } from 'vue';
import { ElInput } from 'element-plus';
import Decimal from '@/core/utils/decimal';

import type { UseField } from '@/web/web/composables/useField';
import { flushPromises, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import ODecimalField from './ODecimalField.vue';
import OFieldBase from './OFieldBase.vue';

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

const lastBaseProps: { current: Record<string, unknown> | null } = { current: null };

function installFieldBaseStub(mode: 'slots' | 'rules' = 'slots') {
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
      if (mode === 'rules') {
        return () => h('div', { class: 'rules', 'data-count': String((p.rules as any[])?.length || 0) });
      }
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

function mountField(binding: any, props: Record<string, unknown> = {}, baseMode: 'slots' | 'rules' = 'slots') {
  lastBaseProps.current = null;
  installFieldBaseStub(baseMode);
  installElInputStub();
  return mountApp(ODecimalField as any, {
    props: {
      binding,
      renderMode: 'form',
      bufferStrategy: 'live',
      commitOnBlur: true,
      ...props,
    },
  });
}

describe('ODecimalField mount wiring', () => {
  afterEach(() => {
    restoreSfc(OFieldBase as any);
    restoreSfc(ElInput as any);
  });

  test('pads display when meta.scale or props.scale is declared (mount wiring)', async () => {
    const withMeta = makeBinding({ Amount: new Decimal('0.01') }, { type: 'decimal', scale: 4 });
    const metaMount = mountField(withMeta, { readonly: true });
    await flushPromises();
    expect(metaMount.q('.o-field-display-text')?.textContent).toBe('0.0100');
    metaMount.unmount();
    restoreSfc(OFieldBase as any);
    restoreSfc(ElInput as any);

    const withProps = makeBinding({ Amount: new Decimal('1.2') });
    const propsMount = mountField(withProps, { readonly: true, scale: 2 });
    await flushPromises();
    expect(propsMount.q('.o-field-display-text')?.textContent).toBe('1.20');
    propsMount.unmount();
  });

  test('resolves scaleField from sibling / metrics__max and registers the path', async () => {
    const binding = makeBinding(
      { Amount: new Decimal('1.2345'), AmountScale: 2 },
      { type: 'decimal', scaleField: 'AmountScale' }
    );
    const m = mountField(binding, { readonly: true });
    await flushPromises();
    expect(binding.__registered).toContain('AmountScale');
    expect(m.q('.o-field-display-text')?.textContent).toBe('1.23');

    binding.__value.value = new Decimal('9.999');
    binding.__recordRef.value = {
      Amount: binding.__value.value,
      metrics: { AmountScale__max: 1 },
    };
    await nextTick();
    await flushPromises();
    expect(m.q('.o-field-display-text')?.textContent).toBe('10.0');
    m.unmount();
  });

  test('live-commits edits using DB soft max 18 when scale is unset', async () => {
    const binding = makeBinding({ Amount: new Decimal('1') });
    const m = mountField(binding);
    await flushPromises();
    const input = m.q('input.el-input') as HTMLInputElement;
    setInput(input, '1.234567890123456789');
    await flushPromises();
    expect(new Decimal(binding.__value.value).toString()).toBe('1.234567890123456789');

    setInput(input, '1.2345678901234567890');
    await flushPromises();
    expect(new Decimal(binding.__value.value).toString()).toBe('1.234567890123456789');
    m.unmount();
  });

  test('quantizes edits to declared props.scale', async () => {
    const binding = makeBinding({ Amount: new Decimal('1') });
    const m = mountField(binding, { scale: 2 });
    await flushPromises();
    const input = m.q('input.el-input') as HTMLInputElement;
    setInput(input, '1.239');
    await flushPromises();
    expect(new Decimal(binding.__value.value).toString()).toBe('1');
    setInput(input, '1.23');
    await flushPromises();
    expect(new Decimal(binding.__value.value).toString()).toBe('1.23');
    m.unmount();
  });

  test('validates via internal rule using edit scale', async () => {
    const binding = makeBinding({ Amount: '1.239' }, { type: 'decimal', scale: 2 });
    const m = mountField(binding, {}, 'rules');
    await flushPromises();
    const rules = (lastBaseProps.current?.rules || []) as any[];
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
    m.unmount();
  });

  test('maps toView/fromView and displays aggregate metrics', async () => {
    const binding = makeBinding({
      Amount: null,
      metrics: { Amount__sum: new Decimal('9.5') },
    });
    const m = mountField(binding, { agg: 'sum', readonly: true });
    await flushPromises();
    expect(m.q('.o-field-display-text')?.textContent || '').toContain('9.5');

    const toView = lastBaseProps.current?.toView as (v: unknown) => unknown;
    const fromView = lastBaseProps.current?.fromView as (v: unknown) => unknown;
    expect(toView(new Decimal('1'))).toBe('1');
    expect(fromView('2.5')).toBeTruthy();
    expect(fromView(null)).toBeNull();
    expect(toView('nope')).toBeNull();
    m.unmount();
  });

  test('bootstraps via useField when binding is omitted', async () => {
    installFieldBaseStub('slots');
    installElInputStub();
    const draft = reactive({ Amount: new Decimal('1') });
    const Host = defineComponent({
      setup() {
        provide('view-container', ref('Form'));
        provide('view-mode', ref('edit'));
        provide('form-root', { draft });
        return () =>
          h(ODecimalField as any, {
            store: {} as any,
            prop: 'Amount',
            renderMode: 'form',
            readonly: true,
          });
      },
    });
    const m = mountApp(Host);
    await flushPromises();
    expect(m.q('.o-field-display-text')?.textContent).toBe('1');
    m.unmount();
  });

  test('falls back currentScale to 18 when getScale returns out of range', async () => {
    const binding = makeBinding({ Amount: new Decimal('1') });
    const m = mountField(binding, { scale: 99 });
    await flushPromises();
    const input = m.q('input.el-input') as HTMLInputElement;
    setInput(input, '1.5');
    await flushPromises();
    expect(new Decimal(binding.__value.value).toString()).toBe('1.5');
    m.unmount();
  });
});

// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Mount coverage for native input/checkbox cell paths (value/onInput/onChange/blur)
 * and toView/fromView/rules helpers on scalar Choy* fields.
 */

import { computed, h, ref } from 'vue';

import { flushPromises, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import FieldBase from './FieldBase.vue';
import ChoyBooleanField from './ChoyBooleanField.vue';
import ChoyVarcharField from './ChoyVarcharField.vue';
import ChoyTextField from './ChoyTextField.vue';
import ChoyIntField from './ChoyIntField.vue';
import ChoyBigintField from './ChoyBigintField.vue';
import ChoyNumberField from './ChoyNumberField.vue';

const lastBaseProps: { current: Record<string, unknown> | null } = { current: null };

function makeBinding(initial: unknown, prop = 'Value') {
  const value = ref(initial);
  return {
    env: { isForm: true, isEditMode: true, viewMode: 'edit', fieldPrefix: null },
    prop,
    meta: {} as any,
    fieldRef: () => value as any,
    fieldRefOf: () => value as any,
    recordRef: () => computed(() => ({})) as any,
    registerFields: () => {},
    store: undefined,
    asView: () => ({ fieldValue: () => value }) as any,
    __value: value,
  };
}

function installFieldBaseStub(mode: 'edit' | 'display' | 'both' | 'rules' = 'both') {
  stubSfc(FieldBase as any, {
    name: 'FieldBase',
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
      id: { type: String, default: undefined },
    },
    setup(p: any, { slots }: any) {
      lastBaseProps.current = p;
      if (mode === 'rules') {
        return () => h('div', { class: 'rules', 'data-count': String((p.rules as any[])?.length || 0) });
      }
      const fieldValue = () => (p.binding as any).fieldRef();
      const kids: any[] = [];
      if (mode !== 'display') kids.push(slots.edit?.({ fieldValue, inputName: 'n', inputId: 'i' }));
      if (mode !== 'edit') kids.push(slots.display?.({ fieldValue, inputName: 'n', inputId: 'i' }));
      return () => h('div', { class: 'base' }, kids);
    },
  });
}

function setInputValue(el: HTMLInputElement | HTMLTextAreaElement, value: string) {
  el.value = value;
  el.dispatchEvent(new Event('input', { bubbles: true }));
}

async function runRule(rules: any[], value: unknown): Promise<Error | undefined> {
  const rule = rules[rules.length - 1];
  return await new Promise(resolve => {
    rule.validator({}, value, (e?: Error) => resolve(e));
  });
}

describe('Choy native scalar field cells', () => {
  afterEach(() => {
    restoreSfc(FieldBase as any);
    lastBaseProps.current = null;
  });

  test('BooleanField edit toggles checked and clearable null', async () => {
    installFieldBaseStub('edit');
    const binding = makeBinding(true);
    const m = mountApp(ChoyBooleanField as any, {
      props: {
        binding,
        widget: 'checkbox',
        nullable: true,
        clearable: true,
        bufferStrategy: 'live',
        commitOnBlur: true,
        renderMode: 'form',
      },
    });
    await flushPromises();
    const input = m.q('input.choy-bool-input') as HTMLInputElement;
    expect(input).toBeTruthy();
    const change = new Event('change', { bubbles: true });
    Object.defineProperty(change, 'target', { value: { checked: false } });
    input.dispatchEvent(change);
    await flushPromises();
    expect(binding.__value.value).toBe(false);

    const clear = m.q('button.choy-clear-btn') as HTMLButtonElement;
    expect(clear).toBeTruthy();
    clear.click();
    await flushPromises();
    expect(binding.__value.value).toBeNull();
    m.unmount();
  });

  test('BooleanField toView/fromView and display switch', async () => {
    installFieldBaseStub('both');
    const binding = makeBinding(true);
    const m = mountApp(ChoyBooleanField as any, {
      props: {
        binding,
        widget: 'switch',
        nullAsFalse: true,
        bufferStrategy: 'live',
        renderMode: 'form',
      },
    });
    await flushPromises();
    expect(m.q('input.choy-bool-input')).toBeTruthy();
    const toView = lastBaseProps.current?.toView as (v: unknown) => unknown;
    const fromView = lastBaseProps.current?.fromView as (v: unknown) => unknown;
    expect(toView(true)).toBe(true);
    expect(toView(false)).toBe(false);
    expect(toView(null)).toBeNull();
    expect(toView(1)).toBe(true);
    expect(toView(0)).toBe(false);
    expect(toView(2)).toBeNull();
    expect(toView('yes')).toBe(true);
    expect(toView('no')).toBe(false);
    expect(toView('maybe')).toBeNull();
    expect(toView({})).toBeNull();
    expect(fromView(null)).toBe(false);
    expect(fromView(true)).toBe(true);
    m.unmount();

    restoreSfc(FieldBase as any);
    installFieldBaseStub('both');
    const nullableBinding = makeBinding(null);
    const nullableMount = mountApp(ChoyBooleanField as any, {
      props: {
        binding: nullableBinding,
        nullAsFalse: false,
        nullable: true,
        bufferStrategy: 'live',
        renderMode: 'form',
      },
    });
    await flushPromises();
    const fromViewNullable = lastBaseProps.current?.fromView as (v: unknown) => unknown;
    expect(fromViewNullable(null)).toBeNull();
    expect(fromViewNullable(true)).toBe(true);
    expect(fromViewNullable(false)).toBe(false);
    const bss = nullableMount.setupState() as any;
    expect(typeof bss.toBool).toBe('function');
    expect(bss.toBool(true)).toBe(true);
    expect(bss.toBool('true')).toBe(true);
    expect(bss.toBool(1)).toBe(true);
    expect(bss.toBool(0)).toBe(false);
    expect(bss.toBool('false')).toBe(false);
    nullableMount.unmount();
  });

  test('VarcharField edit/display, normalize, and rules', async () => {
    installFieldBaseStub('both');
    const binding = makeBinding('hi');
    const m = mountApp(ChoyVarcharField as any, {
      props: {
        binding,
        maxLength: 4,
        trimOnBlur: 'both',
        nullable: true,
        bufferStrategy: 'live',
        commitOnBlur: true,
        renderMode: 'form',
      },
    });
    await flushPromises();
    expect(m.q('.choy-field-display-text')?.textContent).toBe('hi');
    const input = m.q('input.choy-input') as HTMLInputElement;
    expect(input?.id).toBe('i');
    setInputValue(input, '  abcdXX  ');
    input.dispatchEvent(new Event('blur', { bubbles: true }));
    await flushPromises();
    expect(binding.__value.value).toBe('abcd');

    const toView = lastBaseProps.current?.toView as (v: unknown) => unknown;
    expect(toView(null)).toBeNull();
    expect(toView(12)).toBe('12');

    restoreSfc(FieldBase as any);
    installFieldBaseStub('rules');
    const rulesMount = mountApp(ChoyVarcharField as any, {
      props: { binding: makeBinding(null), maxLength: 2, nullable: false, renderMode: 'form' },
    });
    await flushPromises();
    const rules = (lastBaseProps.current?.rules || []) as any[];
    expect(await runRule(rules, null)).toBeTruthy();
    expect(await runRule(rules, 1)).toBeTruthy();
    expect(await runRule(rules, 'abc')).toBeTruthy();
    expect(await runRule(rules, 'ab')).toBeUndefined();
    rulesMount.unmount();
    m.unmount();
  });

  test('TextField edit/display, normalize, and rules', async () => {
    installFieldBaseStub('both');
    const binding = makeBinding('note');
    const m = mountApp(ChoyTextField as any, {
      props: {
        binding,
        maxLength: 5,
        trimOnBlur: 'both',
        nullable: true,
        bufferStrategy: 'live',
        commitOnBlur: true,
        renderMode: 'form',
      },
    });
    await flushPromises();
    expect(m.q('.choy-textfield-display')?.textContent).toBe('note');
    const ta = m.q('textarea.choy-textarea') as HTMLTextAreaElement;
    setInputValue(ta, '  helloXX  ');
    ta.dispatchEvent(new Event('blur', { bubbles: true }));
    await flushPromises();
    expect(binding.__value.value).toBe('hello');

    const toView = lastBaseProps.current?.toView as (v: unknown) => unknown;
    expect(toView(null)).toBeNull();
    expect(toView(9)).toBe('9');

    restoreSfc(FieldBase as any);
    installFieldBaseStub('rules');
    const rulesMount = mountApp(ChoyTextField as any, {
      props: { binding: makeBinding(null), maxLength: 2, nullable: false, renderMode: 'form' },
    });
    await flushPromises();
    const rules = (lastBaseProps.current?.rules || []) as any[];
    expect(await runRule(rules, null)).toBeTruthy();
    expect(await runRule(rules, 1)).toBeTruthy();
    expect(await runRule(rules, 'abc')).toBeTruthy();
    expect(await runRule(rules, 'ab')).toBeUndefined();
    rulesMount.unmount();
    m.unmount();
  });

  test('IntField edit/blur/clamp and rules', async () => {
    installFieldBaseStub('both');
    const binding = makeBinding(1);
    const m = mountApp(ChoyIntField as any, {
      props: {
        binding,
        min: 0,
        max: 10,
        nullable: true,
        bufferStrategy: 'live',
        commitOnBlur: true,
        renderMode: 'form',
      },
    });
    await flushPromises();
    expect(m.q('.choy-field-display-text')?.textContent).toBe('1');
    const input = m.q('input.choy-int-input') as HTMLInputElement;
    setInputValue(input, '7');
    await flushPromises();
    expect(binding.__value.value).toBe(7);
    input.dispatchEvent(new Event('blur', { bubbles: true }));
    await flushPromises();
    expect(binding.__value.value).toBe(7);
    setInputValue(input, '99');
    await flushPromises();
    expect(binding.__value.value).toBe(10);
    setInputValue(input, '-');
    input.dispatchEvent(new Event('blur', { bubbles: true }));
    await flushPromises();
    expect(binding.__value.value).toBeNull();

    const toView = lastBaseProps.current?.toView as (v: unknown) => unknown;
    expect(toView(null)).toBeNull();
    expect(toView('')).toBeNull();
    expect(toView('3')).toBe(3);
    expect(toView('x')).toBeNull();

    restoreSfc(FieldBase as any);
    installFieldBaseStub('rules');
    const rulesMount = mountApp(ChoyIntField as any, {
      props: { binding: makeBinding(null), min: 1, max: 5, nullable: false, renderMode: 'form' },
    });
    await flushPromises();
    const rules = (lastBaseProps.current?.rules || []) as any[];
    expect(await runRule(rules, null)).toBeTruthy();
    expect(await runRule(rules, 1.5)).toBeTruthy();
    expect(await runRule(rules, 0)).toBeTruthy();
    expect(await runRule(rules, 3)).toBeUndefined();
    rulesMount.unmount();
    m.unmount();
  });

  test('BigintField edit/blur/wireFormat and rules', async () => {
    installFieldBaseStub('both');
    const binding = makeBinding('10');
    const m = mountApp(ChoyBigintField as any, {
      props: {
        binding,
        min: 0,
        max: 100,
        wireFormat: 'string',
        nullable: true,
        bufferStrategy: 'live',
        commitOnBlur: true,
        renderMode: 'form',
      },
    });
    await flushPromises();
    expect(m.q('.choy-field-display-text')?.textContent).toBe('10');
    const input = m.q('input.choy-bigint-input') as HTMLInputElement;
    setInputValue(input, '42');
    await flushPromises();
    expect(binding.__value.value).toBe('42');
    input.dispatchEvent(new Event('blur', { bubbles: true }));
    await flushPromises();
    expect(binding.__value.value).toBe('42');
    setInputValue(input, '999');
    await flushPromises();
    expect(binding.__value.value).toBe('100');
    setInputValue(input, '-');
    input.dispatchEvent(new Event('blur', { bubbles: true }));
    await flushPromises();
    expect(binding.__value.value).toBeNull();

    const toView = lastBaseProps.current?.toView as (v: unknown) => unknown;
    const fromView = lastBaseProps.current?.fromView as (v: unknown) => unknown;
    expect(toView(null)).toBeNull();
    expect(toView(' 5 ')).toBe('5');
    expect(toView('nope')).toBeNull();
    expect(toView(7)).toBe('7');
    expect(toView(1.5)).toBeNull();
    expect(toView(7n)).toBe('7');
    expect(toView({})).toBeNull();
    expect(fromView(null)).toBeNull();
    expect(fromView('9')).toBe('9');

    restoreSfc(FieldBase as any);
    installFieldBaseStub('both');
    const numBinding = makeBinding(null);
    const numMount = mountApp(ChoyBigintField as any, {
      props: {
        binding: numBinding,
        wireFormat: 'number',
        bufferStrategy: 'live',
        commitOnBlur: true,
        renderMode: 'form',
      },
    });
    await flushPromises();
    const fromViewNum = lastBaseProps.current?.fromView as (v: unknown) => unknown;
    expect(fromViewNum('12')).toBe(12);
    expect(fromViewNum('9007199254740993')).toBeNull();
    const toViewNum = lastBaseProps.current?.toView as (v: unknown) => unknown;
    expect(toViewNum(3)).toBe('3');
    const biSs = numMount.setupState() as any;
    if (typeof biSs.toBigInt === 'function') {
      expect(biSs.toBigInt(5)).toBe(5n);
      expect(biSs.toBigInt(1.5)).toBeNull();
      expect(biSs.toBigInt('not-int')).toBeNull();
    }
    // Drive wireFormat=number clamp via live input beyond JS safe integer bounds.
    setInputValue(numMount.q('input.choy-bigint-input') as HTMLInputElement, '9007199254740993');
    await flushPromises();

    restoreSfc(FieldBase as any);
    installFieldBaseStub('rules');
    const rulesMount = mountApp(ChoyBigintField as any, {
      props: {
        binding: makeBinding(null),
        min: 1,
        max: 10,
        nullable: false,
        wireFormat: 'number',
        renderMode: 'form',
      },
    });
    await flushPromises();
    const rules = (lastBaseProps.current?.rules || []) as any[];
    expect(await runRule(rules, null)).toBeTruthy();
    expect(await runRule(rules, 'x')).toBeTruthy();
    expect(await runRule(rules, 1.5)).toBeTruthy();
    expect(await runRule(rules, 0)).toBeTruthy();
    expect(await runRule(rules, 5)).toBeUndefined();
    expect(await runRule(rules, 5n)).toBeUndefined();
    expect(await runRule(rules, '9007199254740993')).toBeTruthy();
    expect(await runRule(rules, Number.NaN)).toBeTruthy();
    rulesMount.unmount();
    numMount.unmount();
    m.unmount();
  });

  test('NumberField intermediate blur and rules', async () => {
    installFieldBaseStub('both');
    const binding = makeBinding(1);
    const m = mountApp(ChoyNumberField as any, {
      props: {
        binding,
        min: 0,
        max: 10,
        nullable: true,
        bufferStrategy: 'live',
        commitOnBlur: true,
        renderMode: 'form',
      },
    });
    await flushPromises();
    expect(m.q('.choy-field-display-text')?.textContent).toBe('1');
    const input = m.q('input.choy-number-input') as HTMLInputElement;
    setInputValue(input, '');
    await flushPromises();
    expect(binding.__value.value).toBeNull();
    input.dispatchEvent(new Event('blur', { bubbles: true }));
    await flushPromises();
    expect(binding.__value.value).toBeNull();
    setInputValue(input, '3.');
    input.dispatchEvent(new Event('blur', { bubbles: true }));
    await flushPromises();
    expect(binding.__value.value).toBe(3);
    setInputValue(input, '-');
    input.dispatchEvent(new Event('blur', { bubbles: true }));
    await flushPromises();
    expect(binding.__value.value).toBeNull();

    const toView = lastBaseProps.current?.toView as (v: unknown) => unknown;
    expect(toView('')).toBeNull();
    expect(toView(2.5)).toBe(2.5);
    expect(toView('x')).toBeNull();

    restoreSfc(FieldBase as any);
    installFieldBaseStub('rules');
    const rulesMount = mountApp(ChoyNumberField as any, {
      props: { binding: makeBinding(null), min: 1, max: 5, nullable: false, renderMode: 'form' },
    });
    await flushPromises();
    const rules = (lastBaseProps.current?.rules || []) as any[];
    expect(await runRule(rules, null)).toBeTruthy();
    expect(await runRule(rules, 'x')).toBeTruthy();
    expect(await runRule(rules, 0)).toBeTruthy();
    expect(await runRule(rules, 6)).toBeTruthy();
    expect(await runRule(rules, 3)).toBeUndefined();
    rulesMount.unmount();
    m.unmount();
  });
});

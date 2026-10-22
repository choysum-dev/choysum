// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Mount coverage for native input/checkbox cell paths (value/onInput/onChange/blur)
 * and toView/fromView/rules helpers on scalar Choy* fields.
 */

import { computed, h, nextTick, ref } from 'vue';

import { flushPromises, fnRecorder, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import { ChoyMessage } from '../../composables/useChoyMessage';
import ChoyFieldBase from './ChoyFieldBase.vue';
import ChoyBooleanField from './ChoyBooleanField.vue';
import ChoyVarcharField from './ChoyVarcharField.vue';
import ChoyCharField from './ChoyCharField.vue';
import ChoyTextField from './ChoyTextField.vue';
import ChoyIntField from './ChoyIntField.vue';
import ChoyBigintField from './ChoyBigintField.vue';
import ChoyNumberField from './ChoyNumberField.vue';

const lastBaseProps: { current: Record<string, unknown> | null } = { current: null };

function makeBinding(initial: unknown, prop = 'Value', meta: Record<string, unknown> = {}) {
  const value = ref(initial);
  return {
    env: { isForm: true, isEditMode: true, viewMode: 'edit', fieldPrefix: null },
    prop,
    meta: meta as any,
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
  stubSfc(ChoyFieldBase as any, {
    name: 'ChoyFieldBase',
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
      // Mirror FieldBase view transform so toView/fromView (e.g. char trimEnd) apply in stubs.
      const fieldValue = () => {
        const raw = (p.binding as any).fieldRef();
        return computed({
          get: () => {
            const v = raw.value;
            return typeof p.toView === 'function' ? p.toView(v) : v;
          },
          set: next => {
            raw.value = typeof p.fromView === 'function' ? p.fromView(next) : next;
          },
        });
      };
      return () => {
        const kids: any[] = [];
        if (mode !== 'display') kids.push(slots.edit?.({ fieldValue, inputName: 'n', inputId: 'i' }));
        if (mode !== 'edit') kids.push(slots.display?.({ fieldValue, inputName: 'n', inputId: 'i' }));
        return h('div', { class: 'base' }, kids);
      };
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
    restoreSfc(ChoyFieldBase as any);
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
    const input = m.q('.choy-bool-input') as HTMLElement;
    expect(input).toBeTruthy();
    input.click();
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
    expect(m.q('.choy-bool-input')).toBeTruthy();
    const displayRoot = m.q('[data-testid=choy-bool-display]') as HTMLElement | null;
    expect(displayRoot).toBeTruthy();
    // Minimal DOM has no descendant selectors; walk display children.
    const displayKids = Array.from(displayRoot?.childNodes || []) as Element[];
    expect(displayKids.some(el => String(el.className || '').includes('choy-bool-input'))).toBe(true);
    expect(displayKids.some(el => String(el.className || '').includes('choy-bool-display__text'))).toBe(false);
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

    restoreSfc(ChoyFieldBase as any);
    installFieldBaseStub('both');
    const nullableBinding = makeBinding(null);
    const nullableMount = mountApp(ChoyBooleanField as any, {
      props: {
        binding: nullableBinding,
        widget: 'switch',
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
    // Null + switch: display falls back to text (Switch has no indeterminate).
    expect(nullableMount.q('.choy-bool-display__text')).toBeTruthy();
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
    const display = m.q('.choy-field-display-text') as HTMLElement | null;
    expect(display?.textContent).toBe('hi');
    expect(display?.getAttribute('title')).toBe('hi');
    expect(m.q('.choy-varchar-word-limit')?.textContent || '').toContain('2 / 4');
    const input = m.q('input.choy-varchar-input') as HTMLInputElement;
    expect(input?.id).toBe('i');
    setInputValue(input, '  abcdXX  ');
    input.dispatchEvent(new Event('blur', { bubbles: true }));
    await flushPromises();
    expect(binding.__value.value).toBe('abcd');
    expect(m.q('.choy-varchar-word-limit')?.textContent || '').toContain('4 / 4');

    const toView = lastBaseProps.current?.toView as (v: unknown) => unknown;
    expect(toView(null)).toBeNull();
    expect(toView(12)).toBe('12');

    restoreSfc(ChoyFieldBase as any);
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

  test('VarcharField copyable display toasts success after clipboard write', async () => {
    installFieldBaseStub('display');
    const writeText = fnRecorder(async () => undefined);
    const g = globalThis as { navigator?: { clipboard?: { writeText?: unknown } } };
    const prevNavigator = g.navigator;
    g.navigator = { clipboard: { writeText } };
    const origSuccess = ChoyMessage.success;
    const success = fnRecorder(() => 0);
    ChoyMessage.success = success as typeof ChoyMessage.success;

    const m = mountApp(ChoyVarcharField as any, {
      props: {
        binding: makeBinding('SAMPLE'),
        copyable: true,
        renderMode: 'form',
      },
    });
    await flushPromises();
    const btn = m.q('.choy-varchar-copy-btn') as HTMLButtonElement | null;
    expect(btn).toBeTruthy();
    btn!.dispatchEvent(new Event('click', { bubbles: true }));
    await flushPromises();
    expect(writeText.calls.length).toBe(1);
    expect(writeText.calls[0]?.[0]).toBe('SAMPLE');
    expect(success.calls.length).toBe(1);
    expect(String(success.calls[0]?.[0] ?? '')).toContain('Copied');

    ChoyMessage.success = origSuccess;
    g.navigator = prevNavigator;
    m.unmount();
    restoreSfc(ChoyFieldBase as any);
  });

  test('char trimTrailingSpaces strips pads for word count and commits trimmed value', async () => {
    installFieldBaseStub('both');
    const padded = `Sample Char${' '.repeat(53)}`;
    const binding = makeBinding(padded, 'NameChar', { type: 'char', size: 64 });
    const m = mountApp(ChoyVarcharField as any, {
      props: {
        binding,
        maxLength: 64,
        trimTrailingSpaces: true,
        bufferStrategy: 'live',
        commitOnBlur: true,
        renderMode: 'form',
      },
    });
    await flushPromises();
    expect(m.q('.choy-varchar-word-limit')?.textContent || '').toContain('11 / 64');
    const input = m.q('input.choy-varchar-input') as HTMLInputElement;
    input.dispatchEvent(new Event('blur', { bubbles: true }));
    await flushPromises();
    expect(binding.__value.value).toBe('Sample Char');

    const wrap = mountApp(ChoyCharField as any, {
      props: {
        binding: makeBinding(`Hi${' '.repeat(10)}`, 'NameChar', { type: 'char', size: 64 }),
        maxLength: 64,
        renderMode: 'form',
      },
    });
    await flushPromises();
    expect(wrap.q('.choy-varchar-word-limit')?.textContent || '').toContain('2 / 64');
    wrap.unmount();

    const pwd = mountApp(ChoyVarcharField as any, {
      props: {
        binding: makeBinding('secret'),
        widget: 'password',
        renderMode: 'form',
      },
    });
    await flushPromises();
    const pwdInput = pwd.q('input.choy-varchar-input') as HTMLInputElement | null;
    expect(pwdInput?.getAttribute('type') || pwdInput?.type).toBe('password');
    pwd.unmount();
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
        autosize: { minRows: 3, maxRows: 6 },
      },
    });
    await flushPromises();
    expect(m.q('.choy-textfield-display')?.textContent).toBe('note');
    expect(m.q('.choy-textarea-word-limit')?.textContent || '').toContain('4 / 5');
    const ta = m.q('textarea.choy-textarea') as HTMLTextAreaElement;
    expect(ta?.getAttribute('rows') || String(ta?.rows)).toBe('3');
    setInputValue(ta, '  helloXX  ');
    ta.dispatchEvent(new Event('blur', { bubbles: true }));
    await flushPromises();
    expect(binding.__value.value).toBe('hello');
    expect(m.q('.choy-textarea-word-limit')?.textContent || '').toContain('5 / 5');

    const toView = lastBaseProps.current?.toView as (v: unknown) => unknown;
    expect(toView(null)).toBeNull();
    expect(toView(9)).toBe('9');

    restoreSfc(ChoyFieldBase as any);
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
    setInputValue(input, '7abc');
    await nextTick();
    expect(binding.__value.value).toBe(1);
    expect(input.value).toBe('1');
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

    restoreSfc(ChoyFieldBase as any);
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
    setInputValue(input, '42xyz');
    await nextTick();
    expect(binding.__value.value).toBe('10');
    expect(input.value).toBe('10');
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

    restoreSfc(ChoyFieldBase as any);
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

    restoreSfc(ChoyFieldBase as any);
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
    setInputValue(input, '1abc');
    await nextTick();
    expect(binding.__value.value).toBe(1);
    expect(input.value).toBe('1');
    setInputValue(input, '');
    await flushPromises();
    await nextTick();
    expect(binding.__value.value).toBeNull();
    expect((m.q('.choy-number-display--empty')?.textContent || '').trim()).toBe('');
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

    restoreSfc(ChoyFieldBase as any);
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

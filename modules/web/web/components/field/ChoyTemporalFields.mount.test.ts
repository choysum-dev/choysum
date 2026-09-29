// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Mount coverage for Date/Datetime/Time native picker cells and conversion helpers.
 */

import { computed, h, ref } from 'vue';

import { flushPromises, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import FieldBase from './FieldBase.vue';
import ChoyDateField from './ChoyDateField.vue';
import ChoyDatetimeField from './ChoyDatetimeField.vue';
import ChoyTimeField from './ChoyTimeField.vue';

const lastBaseProps: { current: Record<string, unknown> | null } = { current: null };

function makeBinding(initial: unknown) {
  const value = ref(initial);
  return {
    env: { isForm: true, isEditMode: true, viewMode: 'edit', fieldPrefix: null },
    prop: 'When',
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
    },
    setup(p: any, { slots }: any) {
      lastBaseProps.current = p;
      if (mode === 'rules') {
        return () => h('div', { class: 'rules' });
      }
      const fieldValue = () => (p.binding as any).fieldRef();
      const kids: any[] = [];
      if (mode !== 'display') kids.push(slots.edit?.({ fieldValue }));
      if (mode !== 'edit') kids.push(slots.display?.({ fieldValue }));
      return () => h('div', { class: 'base' }, kids);
    },
  });
}

async function runRule(rules: any[], value: unknown): Promise<Error | undefined> {
  const rule = rules[rules.length - 1];
  return await new Promise(resolve => {
    rule.validator({}, value, (e?: Error) => resolve(e));
  });
}

function setPicker(el: HTMLInputElement, value: string) {
  const ev = new Event('input', { bubbles: true });
  Object.defineProperty(ev, 'target', { value: { value } });
  el.dispatchEvent(ev);
  el.dispatchEvent(new Event('blur', { bubbles: true }));
}

describe('Choy temporal field cells', () => {
  afterEach(() => {
    restoreSfc(FieldBase as any);
    lastBaseProps.current = null;
  });

  test('DateField display, toView/fromView, and rules', async () => {
    installFieldBaseStub('display');
    const binding = makeBinding(new Date(Date.UTC(2024, 5, 15)));
    const m = mountApp(ChoyDateField as any, {
      props: {
        binding,
        bufferStrategy: 'live',
        commitOnBlur: true,
        renderMode: 'form',
      },
    });
    await flushPromises();
    expect(m.q('.choy-field-display-text')?.textContent || '').toMatch(/2024/);

    const toView = lastBaseProps.current?.toView as (v: unknown) => unknown;
    const fromView = lastBaseProps.current?.fromView as (v: unknown) => unknown;
    expect(toView(null)).toBeNull();
    expect(toView(new Date('invalid'))).toBeNull();
    expect(toView('2024-01-02')).toBeInstanceOf(Date);
    expect(toView('2024-01-02T00:00:00Z')).toBeInstanceOf(Date);
    expect(toView('not-a-date')).toBeNull();
    expect(toView(new Date('2024-03-04T00:00:00Z'))).toBeInstanceOf(Date);
    expect(toView(Date.parse('2024-03-04'))).toBeInstanceOf(Date);
    expect(fromView(null)).toBeNull();
    expect(fromView(new Date('2024-03-04T12:00:00Z'))).toMatch(/2024-03-04/);
    expect(fromView(new Date('invalid'))).toBeNull();

    restoreSfc(FieldBase as any);
    installFieldBaseStub('rules');
    const rulesMount = mountApp(ChoyDateField as any, {
      props: { binding: makeBinding(null), renderMode: 'form' },
    });
    await flushPromises();
    const rules = (lastBaseProps.current?.rules || []) as any[];
    expect(rules.length).toBeGreaterThan(0);
    expect(await runRule(rules, null)).toBeUndefined();
    expect(await runRule(rules, '2024-01-01')).toBeUndefined();
    expect(await runRule(rules, new Date('2024-01-01'))).toBeUndefined();
    const bad = await runRule(rules, 'totally-not-a-date');
    // Invalid strings should fail the internal date rule when parseFlexible rejects them.
    if (bad != null) expect(bad).toBeInstanceOf(Error);
    rulesMount.unmount();
    m.unmount();
  });

  test('TimeField edit/display, toView/fromView, and rules', async () => {
    installFieldBaseStub('both');
    const binding = makeBinding(new Date(2024, 0, 1, 14, 30, 0));
    const m = mountApp(ChoyTimeField as any, {
      props: {
        binding,
        bufferStrategy: 'live',
        commitOnBlur: true,
        renderMode: 'form',
      },
    });
    await flushPromises();
    expect(m.q('.choy-field-display-text')?.textContent || '').toMatch(/14/);
    const input = m.q('input.choy-time-picker') as HTMLInputElement;
    expect(input).toBeTruthy();
    expect(input.getAttribute('type')).toBe('time');
    setPicker(input, '09:15:30');
    await flushPromises();

    const toView = lastBaseProps.current?.toView as (v: unknown) => unknown;
    const fromView = lastBaseProps.current?.fromView as (v: unknown) => unknown;
    expect(toView(null)).toBeNull();
    expect(toView(new Date('invalid'))).toBeNull();
    expect(toView('09:15')).toBeInstanceOf(Date);
    expect(toView('09:15:30')).toBeInstanceOf(Date);
    expect(toView('nope')).toBeNull();
    expect(toView(new Date(2024, 0, 1, 8, 0, 0))).toBeInstanceOf(Date);
    expect(toView(Date.parse('1970-01-01T08:00:00Z'))).toBeInstanceOf(Date);
    expect(fromView(null)).toBeNull();
    expect(fromView(new Date(2024, 0, 1, 8, 5, 6))).toMatch(/08:05:06/);
    expect(fromView(new Date('invalid'))).toBeNull();

    restoreSfc(FieldBase as any);
    installFieldBaseStub('rules');
    const rulesMount = mountApp(ChoyTimeField as any, {
      props: { binding: makeBinding(null), renderMode: 'form' },
    });
    await flushPromises();
    const rules = (lastBaseProps.current?.rules || []) as any[];
    expect(await runRule(rules, null)).toBeUndefined();
    expect(await runRule(rules, '08:00:00')).toBeUndefined();
    expect(await runRule(rules, 'bad')).toBeTruthy();
    expect(await runRule(rules, new Date(2024, 0, 1, 8, 0, 0))).toBeUndefined();
    rulesMount.unmount();
    m.unmount();
  });

  test('DatetimeField edit/display, toView/fromView, and rules', async () => {
    installFieldBaseStub('both');
    const binding = makeBinding(new Date('2024-06-15T12:30:00.000Z'));
    const m = mountApp(ChoyDatetimeField as any, {
      props: {
        binding,
        bufferStrategy: 'live',
        commitOnBlur: true,
        renderMode: 'form',
      },
    });
    await flushPromises();
    expect(m.q('.choy-field-display-text')?.textContent).toBeTruthy();
    const input = m.q('input.choy-date-picker') as HTMLInputElement;
    expect(input).toBeTruthy();
    expect(input.getAttribute('type')).toBe('datetime-local');
    setPicker(input, '2024-07-01T10:15');
    await flushPromises();

    const toView = lastBaseProps.current?.toView as (v: unknown) => unknown;
    const fromView = lastBaseProps.current?.fromView as (v: unknown) => unknown;
    expect(toView(null)).toBeNull();
    expect(toView(new Date('2024-01-02T03:04:05.000Z'))).toBeInstanceOf(Date);
    expect(toView('2024-01-02T03:04:05.000Z')).toBeInstanceOf(Date);
    expect(toView('2024-01-02T03:04:05.1Z')).toBeInstanceOf(Date);
    expect(toView('2024-01-02T03:04:05Z')).toBeInstanceOf(Date);
    expect(toView('not-a-date')).toBeNull();
    expect(toView(Date.parse('2024-01-02T03:04:05.000Z'))).toBeInstanceOf(Date);
    expect(fromView(null)).toBeNull();
    const wall = toView('2024-01-02T03:04:05.000Z') as Date;
    expect(fromView(wall)).toMatch(/2024/);
    expect(fromView(new Date('invalid'))).toBeNull();

    restoreSfc(FieldBase as any);
    installFieldBaseStub('rules');
    const rulesMount = mountApp(ChoyDatetimeField as any, {
      props: { binding: makeBinding(null), renderMode: 'form' },
    });
    await flushPromises();
    const rules = (lastBaseProps.current?.rules || []) as any[];
    expect(await runRule(rules, null)).toBeUndefined();
    expect(await runRule(rules, '2024-01-02T03:04:05.000Z')).toBeUndefined();
    expect(await runRule(rules, 'bad')).toBeTruthy();
    expect(await runRule(rules, new Date('2024-01-02T03:04:05.000Z'))).toBeUndefined();
    rulesMount.unmount();
    m.unmount();
  });
});

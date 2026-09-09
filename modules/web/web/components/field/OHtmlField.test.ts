// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Mount wiring for OHtmlField (TipTap editor stubbed via FE unit package stubs).
 * Sanitize / plaintext helpers stay in ohtml_helpers.test.ts.
 */

import { computed, h, nextTick, reactive, ref } from 'vue';
import * as TipTapVue3 from '@tiptap/vue-3';

import type { UseField } from '@/web/web/composables/useField';
import { flushPromises, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import OFieldBase from './OFieldBase.vue';
import OHtmlField from './OHtmlField.vue';

type FeStubEditor = {
  getHTML: () => string;
  isActive: (name: string) => boolean;
  getAttributes: (name: string) => Record<string, unknown>;
  __feStubSetHTML: (html: string, emitUpdate?: boolean) => void;
};

function lastStubEditor(): FeStubEditor {
  const ed = (TipTapVue3 as { __feStubLastEditor?: FeStubEditor | null }).__feStubLastEditor;
  if (!ed) throw new Error('expected TipTap FE stub editor');
  return ed;
}

function makeBinding(
  record: Record<string, unknown>,
  opts?: { isForm?: boolean }
): UseField & { __value: any; __recordRef: any } {
  const value = ref(record.Body ?? null);
  const recordRef = ref(record);
  return {
    env: {
      isForm: opts?.isForm !== false,
      isEditMode: true,
      viewMode: 'edit',
      fieldPrefix: null,
    },
    prop: 'Body',
    meta: reactive({ type: 'html' }) as any,
    fieldRef: () => value as any,
    fieldRefOf: () => value as any,
    recordRef: () => computed(() => recordRef.value) as any,
    registerFields: () => {},
    store: undefined,
    asView: () => ({ fieldValue: () => value }) as any,
    __value: value,
    __recordRef: recordRef,
  } as any;
}

const lastBaseProps: { current: Record<string, unknown> | null } = { current: null };

function installFieldBaseStub(mode: 'both' | 'edit' | 'display' = 'both') {
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
      const kids: any[] = [];
      if (mode !== 'edit') kids.push(slots.display?.({ fieldValue, record }));
      if (mode !== 'display') kids.push(slots.edit?.({ fieldValue, record }));
      return () => h('div', { class: 'base' }, kids);
    },
  });
}

function mountField(binding: any, props: Record<string, unknown> = {}, mode: 'both' | 'edit' | 'display' = 'both') {
  lastBaseProps.current = null;
  installFieldBaseStub(mode);
  return mountApp(OHtmlField as any, {
    props: {
      binding,
      renderMode: 'form',
      ...props,
    },
  });
}

describe('OHtmlField mount wiring', () => {
  const originalPrompt = (window as any).prompt;

  afterEach(() => {
    (window as any).prompt = originalPrompt;
    restoreSfc(OFieldBase as any);
  });

  test('edit mode renders toolbar and TipTap editor surface', async () => {
    const binding = makeBinding({ Body: '<p>hello</p>' });
    const m = mountField(binding, {}, 'edit');
    await flushPromises();
    expect(m.q('.o-htmlfield-toolbar')).toBeTruthy();
    expect(m.q('.fe-stub-editor-content') || m.q('.o-htmlfield-editor')).toBeTruthy();
    expect(lastStubEditor().getHTML()).toBe('<p>hello</p>');
    m.unmount();
  });

  test('form display renders HTML text; table mode uses plaintext ellipsis', async () => {
    const formBinding = makeBinding({ Body: '<p>Hi <strong>there</strong></p>' });
    const formMount = mountField(formBinding, { renderMode: 'form' }, 'display');
    await flushPromises();
    const display = formMount.q('.o-htmlfield-display');
    expect(display).toBeTruthy();
    // Minimal DOM projects v-html via textContent; assert visible text + container class.
    expect((display?.textContent || '').replace(/\s+/g, ' ')).toContain('Hi');
    expect((display?.textContent || '').replace(/\s+/g, ' ')).toContain('there');
    formMount.unmount();
    restoreSfc(OFieldBase as any);

    const tableBinding = makeBinding({ Body: '<p>Plain <em>me</em></p>' });
    const tableMount = mountField(tableBinding, { renderMode: 'table' }, 'display');
    await flushPromises();
    const plain = tableMount.q('.o-htmlfield-plaintext');
    expect(plain).toBeTruthy();
    expect((plain?.textContent || '').replace(/\s+/g, ' ').trim()).toContain('Plain');
    tableMount.unmount();
  });

  test('commit bridge pushes editor updates into the field model', async () => {
    const binding = makeBinding({ Body: '<p>a</p>' });
    const m = mountField(binding, {}, 'edit');
    await flushPromises();
    lastStubEditor().__feStubSetHTML('<p>edited</p>', true);
    await nextTick();
    await flushPromises();
    expect(binding.__value.value).toBe('<p>edited</p>');
    m.unmount();
  });

  test('commit bridge normalizes blank editor HTML to null', async () => {
    const binding = makeBinding({ Body: '<p>x</p>' });
    const m = mountField(binding, {}, 'edit');
    await flushPromises();
    lastStubEditor().__feStubSetHTML('<p></p>', true);
    await nextTick();
    await flushPromises();
    expect(binding.__value.value).toBeNull();
    m.unmount();
  });

  test('store value changes flow into the editor', async () => {
    const binding = makeBinding({ Body: '<p>one</p>' });
    const m = mountField(binding, {}, 'edit');
    await flushPromises();
    binding.__value.value = '<p>two</p>';
    await nextTick();
    await flushPromises();
    expect(lastStubEditor().getHTML()).toBe('<p>two</p>');
    m.unmount();
  });

  test('toolbar buttons toggle marks and link uses prompt', async () => {
    const binding = makeBinding({ Body: '<p>x</p>' });
    const m = mountField(binding, {}, 'edit');
    await flushPromises();
    const buttons = Array.from(m.el.querySelectorAll('.o-htmlfield-btn')) as HTMLButtonElement[];
    expect(buttons.length).toBeGreaterThanOrEqual(5);
    buttons[0].click(); // bold
    buttons[1].click(); // italic
    const ed = lastStubEditor();
    expect(ed.isActive('bold')).toBe(true);
    expect(ed.isActive('italic')).toBe(true);

    (window as any).prompt = () => 'https://example.com';
    buttons[4].click(); // link
    expect(ed.isActive('link')).toBe(true);
    expect(ed.getAttributes('link').href).toBe('https://example.com');

    (window as any).prompt = () => '';
    buttons[4].click(); // unset via empty prompt while active
    expect(ed.isActive('link')).toBe(false);
    m.unmount();
  });

  test('toView/fromView and string validation rule are wired through OFieldBase', async () => {
    const binding = makeBinding({ Body: null });
    const m = mountField(binding, { required: true }, 'edit');
    await flushPromises();
    const toView = lastBaseProps.current?.toView as (v: unknown) => unknown;
    const fromView = lastBaseProps.current?.fromView as (v: unknown) => unknown;
    expect(toView(null)).toBeNull();
    expect(toView('<p>z</p>')).toBe('<p>z</p>');
    expect(fromView('<p>z</p>')).toBe('<p>z</p>');

    const rules = lastBaseProps.current?.rules as Array<{ validator?: Function }>;
    expect(Array.isArray(rules) && rules.length >= 1).toBe(true);
    const validator = rules[rules.length - 1].validator!;
    let err: Error | undefined;
    validator({}, 123, (e?: Error) => {
      err = e;
    });
    expect(err).toBeTruthy();
    err = undefined;
    validator({}, '<p>ok</p>', (e?: Error) => {
      err = e;
    });
    expect(err).toBeUndefined();
    m.unmount();
  });
});

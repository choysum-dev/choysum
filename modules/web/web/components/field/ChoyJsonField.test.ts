// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, h, nextTick, reactive, ref } from 'vue';
import VueJsonPretty from 'vue-json-pretty';

import type { UseField } from '@/web/web/composables/useField';
import { flushPromises, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import FieldBase from './FieldBase.vue';
import JsonobjectField from './ChoyJsonField.vue';
/* el-stubs-for-typecheck */
const ElInput: any = { name: "ElInput" };


function makeBinding(
  record: Record<string, unknown>,
  env: Record<string, unknown> = { isForm: true, isEditMode: false, viewMode: 'display', fieldPrefix: null }
): UseField & { __value: any } {
  const value = ref(record.Payload ?? null);
  const recordRef = ref(record);
  return {
    env,
    prop: 'Payload',
    meta: reactive({ type: 'jsonobject' }) as any,
    fieldRef: () => value as any,
    fieldRefOf: () => value as any,
    recordRef: () => computed(() => recordRef.value) as any,
    registerFields: () => undefined,
    store: undefined,
    asView: () => ({ fieldValue: () => value }) as any,
    __value: value,
  } as any;
}

function installFieldBaseStub() {
  stubSfc(FieldBase as any, {
    name: 'FieldBase',
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
    setup(p: any, { slots }: any) {
      const fieldValue = () => (p.binding as any).fieldRef();
      const record = () => (p.binding as any).recordRef();
      return () =>
        h('div', { class: 'field-base-stub' }, [
          slots.edit?.({ fieldValue, record }),
          slots.display?.({ fieldValue, record }),
        ]);
    },
  });
}

function installPrettyStub() {
  stubSfc(VueJsonPretty as any, {
    name: 'VueJsonPrettyStub',
    props: {
      data: { type: [Object, Array], default: null },
      deep: { type: Number, default: undefined },
      showLength: { type: Boolean, default: false },
      showLine: { type: Boolean, default: false },
      collapsedOnClickBrackets: { type: Boolean, default: false },
    },
    setup(p: any) {
      return () =>
        h('div', { class: 'vue-json-pretty-stub', 'data-deep': String(p.deep ?? '') }, JSON.stringify(p.data));
    },
  });
}

function installElInputStub() {
  stubSfc(ElInput as any, {
    name: 'ElInput',
    props: {
      modelValue: { type: [String, Number], default: '' },
      type: String,
      placeholder: String,
      autosize: [Boolean, Object],
    },
    emits: ['update:modelValue', 'blur'],
    setup(props: any, { emit }: any) {
      return () =>
        h('textarea', {
          class: 'choy-json-input',
          value: props.modelValue,
          placeholder: props.placeholder,
          onInput: (e: Event) => emit('update:modelValue', (e.target as HTMLTextAreaElement).value),
          onBlur: () => emit('blur'),
        });
    },
  });
}

describe('JsonobjectField', () => {
  afterEach(() => {
    restoreSfc(FieldBase as any);
    restoreSfc(VueJsonPretty as any);
    restoreSfc(ElInput as any);
  });

  test('form display uses vue-json-pretty for object values', async () => {
    installFieldBaseStub();
    installPrettyStub();
    installElInputStub();
    const binding = makeBinding({ Payload: { b: 2, a: 1 } });
    const m = mountApp(JsonobjectField as any, {
      props: { binding, renderMode: 'form' },
    });
    await nextTick();
    const pretty = m.q('.vue-json-pretty-stub');
    expect(pretty).toBeTruthy();
    expect(pretty?.textContent || '').toContain('"a":1');
    expect(pretty?.getAttribute('data-deep')).toBe('3');
    expect(m.q('.choy-json-display')).toBeFalsy();
    m.unmount();
  });

  test('table/inline display uses compact plaintext', async () => {
    installFieldBaseStub();
    installPrettyStub();
    installElInputStub();
    const binding = makeBinding({ Payload: { hello: 'world' } });
    for (const renderMode of ['table', 'inline'] as const) {
      const m = mountApp(JsonobjectField as any, {
        props: { binding, renderMode },
      });
      await nextTick();
      expect(m.q('.vue-json-pretty-stub')).toBeFalsy();
      expect(m.q('.choy-json-display')?.textContent || '').toContain('hello');
      m.unmount();
    }
  });

  test('auto renderMode follows isForm for compact vs pretty', async () => {
    installFieldBaseStub();
    installPrettyStub();
    installElInputStub();

    const listBinding = makeBinding({ Payload: { k: 1 } }, { isForm: false });
    const listMount = mountApp(JsonobjectField as any, {
      props: { binding: listBinding, renderMode: 'auto' },
    });
    await nextTick();
    expect(listMount.q('.vue-json-pretty-stub')).toBeFalsy();
    expect(listMount.q('.choy-json-display')?.textContent || '').toContain('"k"');
    listMount.unmount();

    const formBinding = makeBinding({ Payload: { k: 1 } }, { isForm: true });
    const formMount = mountApp(JsonobjectField as any, {
      props: { binding: formBinding, renderMode: 'auto' },
    });
    await nextTick();
    expect(formMount.q('.vue-json-pretty-stub')).toBeTruthy();
    formMount.unmount();

    const noEnvBinding = makeBinding({ Payload: { k: 1 } });
    (noEnvBinding as any).env = undefined;
    const noEnvMount = mountApp(JsonobjectField as any, {
      props: { binding: noEnvBinding, renderMode: 'auto' },
    });
    await nextTick();
    expect(noEnvMount.q('.vue-json-pretty-stub')).toBeTruthy();
    noEnvMount.unmount();
  });

  test('null form display stays empty without pretty tree', async () => {
    installFieldBaseStub();
    installPrettyStub();
    installElInputStub();
    const binding = makeBinding({ Payload: null });
    const m = mountApp(JsonobjectField as any, {
      props: { binding, renderMode: 'form' },
    });
    await nextTick();
    expect(m.q('.vue-json-pretty-stub')).toBeFalsy();
    expect(m.q('.choy-json-display--empty')).toBeTruthy();
    m.unmount();
  });

  test('edit mode still mounts textarea cell', async () => {
    installFieldBaseStub();
    installPrettyStub();
    installElInputStub();
    const binding = makeBinding(
      { Payload: { x: 1 } },
      { isForm: true, isEditMode: true, viewMode: 'edit', fieldPrefix: null }
    );
    const m = mountApp(JsonobjectField as any, {
      props: { binding, renderMode: 'form' },
    });
    await nextTick();
    await flushPromises();
    expect(m.q('.choy-json-input')).toBeTruthy();
    m.unmount();
  });

  test('edit textarea commits valid JSON, rejects invalid, and clears nullable', async () => {
    installFieldBaseStub();
    installPrettyStub();
    installElInputStub();
    const binding = makeBinding(
      { Payload: { a: 1 } },
      { isForm: true, isEditMode: true, viewMode: 'edit', fieldPrefix: null }
    );
    const m = mountApp(JsonobjectField as any, {
      props: {
        binding,
        renderMode: 'form',
        nullable: true,
        allowArray: false,
        bufferStrategy: 'blur',
        commitOnBlur: true,
      },
    });
    await nextTick();
    await flushPromises();
    const ta = m.q('.choy-json-input') as HTMLTextAreaElement;
    expect(ta).toBeTruthy();

    ta.value = '{"z":2,"a":1}';
    ta.dispatchEvent(new Event('input', { bubbles: true }));
    ta.dispatchEvent(new Event('blur', { bubbles: true }));
    await flushPromises();
    expect(binding.__value.value).toEqual({ a: 1, z: 2 });

    ta.value = '{bad';
    ta.dispatchEvent(new Event('input', { bubbles: true }));
    ta.dispatchEvent(new Event('blur', { bubbles: true }));
    await flushPromises();
    expect(m.q('.choy-json-err')).toBeTruthy();

    ta.value = '[1]';
    ta.dispatchEvent(new Event('input', { bubbles: true }));
    ta.dispatchEvent(new Event('blur', { bubbles: true }));
    await flushPromises();
    expect(m.q('.choy-json-err')?.textContent || '').toMatch(/array|Array/i);

    ta.value = '   ';
    ta.dispatchEvent(new Event('input', { bubbles: true }));
    ta.dispatchEvent(new Event('blur', { bubbles: true }));
    await flushPromises();
    expect(binding.__value.value).toBeNull();
    m.unmount();
  });

  test('edit non-nullable empty and toView/fromView/rules helpers', async () => {
    installFieldBaseStub();
    installPrettyStub();
    installElInputStub();
    const binding = makeBinding(
      { Payload: { a: 1 } },
      { isForm: true, isEditMode: true, viewMode: 'edit', fieldPrefix: null }
    );
    const m = mountApp(JsonobjectField as any, {
      props: {
        binding,
        renderMode: 'form',
        nullable: false,
        allowArray: true,
        bufferStrategy: 'blur',
        commitOnBlur: true,
      },
    });
    await nextTick();
    await flushPromises();
    const ta = m.q('.choy-json-input') as HTMLTextAreaElement;
    ta.value = '';
    ta.dispatchEvent(new Event('input', { bubbles: true }));
    ta.dispatchEvent(new Event('blur', { bubbles: true }));
    await flushPromises();
    expect(m.q('.choy-json-err')).toBeTruthy();

    ta.value = '[1,2]';
    ta.dispatchEvent(new Event('input', { bubbles: true }));
    ta.dispatchEvent(new Event('blur', { bubbles: true }));
    await flushPromises();
    expect(binding.__value.value).toEqual([1, 2]);
    m.unmount();
    restoreSfc(FieldBase as any);

    const captured: { current: any } = { current: null };
    stubSfc(FieldBase as any, {
      name: 'FieldBase',
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
      setup(p: any) {
        captured.current = p;
        return () => h('div', { class: 'rules-only' });
      },
    });
    installPrettyStub();
    const rulesMount = mountApp(JsonobjectField as any, {
      props: {
        binding: makeBinding({ Payload: null }),
        renderMode: 'form',
        nullable: false,
        allowArray: false,
      },
    });
    await flushPromises();
    const toV = captured.current.toView as (v: unknown) => unknown;
    const fromV = captured.current.fromView as (v: unknown) => unknown;
    expect(toV(null)).toBeNull();
    expect(toV({ a: 1 })).toEqual({ a: 1 });
    expect(toV('{"b":2}')).toEqual({ b: 2 });
    expect(toV('"x"')).toBeNull();
    expect(toV('{bad')).toBeNull();
    expect(fromV({ c: 3 })).toEqual({ c: 3 });

    const rules = (captured.current.rules || []) as any[];
    const rule = rules[rules.length - 1];
    const run = (value: unknown) =>
      new Promise<Error | undefined>(resolve => {
        rule.validator({}, value, (e?: Error) => resolve(e));
      });
    expect(await run(null)).toBeTruthy();
    expect(await run('{"a":1}')).toBeUndefined();
    expect(await run(1)).toBeTruthy();
    expect(await run([1])).toBeTruthy();
    expect(await run({ a: 1 })).toBeUndefined();
    rulesMount.unmount();
  });
  test('setupState covers stableStringify, normalizeIncoming, and jsonEquals edges', async () => {
    installFieldBaseStub();
    installPrettyStub();
    installElInputStub();
    const binding = makeBinding(
      { Payload: { a: 1 } },
      { isForm: true, isEditMode: true, viewMode: 'edit', fieldPrefix: null }
    );
    const m = mountApp(JsonobjectField as any, {
      props: { binding, renderMode: 'form', allowArray: true, nullable: true },
    });
    await nextTick();
    await flushPromises();
    const ss = m.setupState() as any;
    expect(typeof ss.stableStringify).toBe('function');
    expect(typeof ss.normalizeIncoming).toBe('function');
    expect(typeof ss.jsonEquals).toBe('function');

    expect(ss.stableStringify(null)).toBe('');
    expect(ss.stableStringify([1, 2])).toContain('1');
    expect(ss.stableStringify({ b: 2, a: 1 })).toContain('"a"');
    const circular: any = {};
    circular.self = circular;
    expect(ss.stableStringify(circular)).toBe('');

    expect(ss.normalizeIncoming(null)).toBeNull();
    expect(ss.normalizeIncoming({ x: 1 })).toEqual({ x: 1 });
    expect(ss.normalizeIncoming('null')).toBeNull();
    expect(ss.normalizeIncoming('"hi"')).toBeNull();
    expect(ss.normalizeIncoming('{bad')).toBeNull();
    expect(ss.normalizeIncoming(12)).toBeNull();
    expect(ss.normalizeIncoming('{"k":1}')).toEqual({ k: 1 });

    expect(ss.jsonEquals(1, 1)).toBe(true);
    expect(ss.jsonEquals({ a: 1 }, { a: 1 })).toBe(true);
    expect(ss.jsonEquals(circular, circular)).toBe(true);
    const circular2: any = {};
    circular2.self = circular2;
    expect(ss.jsonEquals(circular, circular2)).toBe(false);
    m.unmount();
  });

});

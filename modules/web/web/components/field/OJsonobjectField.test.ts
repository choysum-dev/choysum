// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, h, nextTick, reactive, ref } from 'vue';
import { ElInput } from 'element-plus';
import VueJsonPretty from 'vue-json-pretty';

import type { UseField } from '@/web/web/composables/useField';
import { flushPromises, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import OFieldBase from './OFieldBase.vue';
import OJsonobjectField from './OJsonobjectField.vue';

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
  stubSfc(OFieldBase as any, {
    name: 'OFieldBase',
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
          class: 'o-json-input',
          value: props.modelValue,
          placeholder: props.placeholder,
          onInput: (e: Event) => emit('update:modelValue', (e.target as HTMLTextAreaElement).value),
          onBlur: () => emit('blur'),
        });
    },
  });
}

describe('OJsonobjectField', () => {
  afterEach(() => {
    restoreSfc(OFieldBase as any);
    restoreSfc(VueJsonPretty as any);
    restoreSfc(ElInput as any);
  });

  test('form display uses vue-json-pretty for object values', async () => {
    installFieldBaseStub();
    installPrettyStub();
    installElInputStub();
    const binding = makeBinding({ Payload: { b: 2, a: 1 } });
    const m = mountApp(OJsonobjectField as any, {
      props: { binding, renderMode: 'form' },
    });
    await nextTick();
    const pretty = m.q('.vue-json-pretty-stub');
    expect(pretty).toBeTruthy();
    expect(pretty?.textContent || '').toContain('"a":1');
    expect(pretty?.getAttribute('data-deep')).toBe('3');
    expect(m.q('.o-json-display')).toBeFalsy();
    m.unmount();
  });

  test('table/inline display uses compact plaintext', async () => {
    installFieldBaseStub();
    installPrettyStub();
    installElInputStub();
    const binding = makeBinding({ Payload: { hello: 'world' } });
    for (const renderMode of ['table', 'inline'] as const) {
      const m = mountApp(OJsonobjectField as any, {
        props: { binding, renderMode },
      });
      await nextTick();
      expect(m.q('.vue-json-pretty-stub')).toBeFalsy();
      expect(m.q('.o-json-display')?.textContent || '').toContain('hello');
      m.unmount();
    }
  });

  test('auto renderMode follows isForm for compact vs pretty', async () => {
    installFieldBaseStub();
    installPrettyStub();
    installElInputStub();

    const listBinding = makeBinding({ Payload: { k: 1 } }, { isForm: false });
    const listMount = mountApp(OJsonobjectField as any, {
      props: { binding: listBinding, renderMode: 'auto' },
    });
    await nextTick();
    expect(listMount.q('.vue-json-pretty-stub')).toBeFalsy();
    expect(listMount.q('.o-json-display')?.textContent || '').toContain('"k"');
    listMount.unmount();

    const formBinding = makeBinding({ Payload: { k: 1 } }, { isForm: true });
    const formMount = mountApp(OJsonobjectField as any, {
      props: { binding: formBinding, renderMode: 'auto' },
    });
    await nextTick();
    expect(formMount.q('.vue-json-pretty-stub')).toBeTruthy();
    formMount.unmount();

    const noEnvBinding = makeBinding({ Payload: { k: 1 } });
    (noEnvBinding as any).env = undefined;
    const noEnvMount = mountApp(OJsonobjectField as any, {
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
    const m = mountApp(OJsonobjectField as any, {
      props: { binding, renderMode: 'form' },
    });
    await nextTick();
    expect(m.q('.vue-json-pretty-stub')).toBeFalsy();
    expect(m.q('.o-json-display--empty')).toBeTruthy();
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
    const m = mountApp(OJsonobjectField as any, {
      props: { binding, renderMode: 'form' },
    });
    await nextTick();
    await flushPromises();
    expect(m.q('.o-json-input')).toBeTruthy();
    m.unmount();
  });
});

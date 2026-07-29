// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount } from '@vue/test-utils';
import { computed, defineComponent, h, nextTick, reactive, ref } from 'vue';
import { describe, expect, it, vi } from 'vitest';
import type { UseField } from '@/web/web/composables/useField';
import OJsonobjectField from './OJsonobjectField.vue';

vi.mock('@/web/web/i18n', async () => {
  const actual = await vi.importActual<typeof import('@/web/web/i18n')>('@/web/web/i18n');
  return {
    ...actual,
    createTranslate: () => ({
      _t: (msg: string, ...args: unknown[]) => (args.length ? `${msg}:${args.join(',')}` : msg),
    }),
  };
});

vi.mock('element-plus', async () => {
  const actual = await vi.importActual<typeof import('element-plus')>('element-plus');
  return {
    ...actual,
    ElInput: defineComponent({
      name: 'ElInputStub',
      props: {
        modelValue: { type: [String, Number], default: '' },
        type: String,
        placeholder: String,
        autosize: [Boolean, Object],
      },
      emits: ['update:modelValue', 'blur'],
      setup(props, { emit }) {
        return () =>
          h('textarea', {
            class: 'o-json-input',
            value: props.modelValue,
            placeholder: props.placeholder,
            onInput: (e: Event) => emit('update:modelValue', (e.target as HTMLTextAreaElement).value),
            onBlur: () => emit('blur'),
          });
      },
    }),
  };
});

vi.mock('vue-json-pretty', () => ({
  default: defineComponent({
    name: 'VueJsonPrettyStub',
    props: {
      data: { type: [Object, Array], default: null },
      deep: { type: Number, default: undefined },
      showLength: { type: Boolean, default: false },
      showLine: { type: Boolean, default: false },
      collapsedOnClickBrackets: { type: Boolean, default: false },
    },
    setup(p) {
      return () =>
        h('div', { class: 'vue-json-pretty-stub', 'data-deep': String(p.deep ?? '') }, JSON.stringify(p.data));
    },
  }),
}));

vi.mock('vue-json-pretty/lib/styles.css', () => ({}));

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
    const record = () => (p.binding as any).recordRef();
    return () =>
      h('div', { class: 'field-base-stub' }, [
        slots.edit?.({ fieldValue, record }),
        slots.display?.({ fieldValue, record }),
      ]);
  },
});

describe('OJsonobjectField', () => {
  it('form display uses vue-json-pretty for object values', async () => {
    const binding = makeBinding({ Payload: { b: 2, a: 1 } });
    const wrapper = mount(OJsonobjectField as any, {
      props: { binding, renderMode: 'form' },
      global: { stubs: { OFieldBase: fieldBaseStub } },
    });
    await nextTick();
    const pretty = wrapper.find('.vue-json-pretty-stub');
    expect(pretty.exists()).toBe(true);
    expect(pretty.text()).toContain('"a":1');
    expect(pretty.attributes('data-deep')).toBe('3');
    expect(wrapper.find('.o-json-display').exists()).toBe(false);
    wrapper.unmount();
  });

  it('table/inline display uses compact plaintext', async () => {
    const binding = makeBinding({ Payload: { hello: 'world' } });
    for (const renderMode of ['table', 'inline'] as const) {
      const wrapper = mount(OJsonobjectField as any, {
        props: { binding, renderMode },
        global: { stubs: { OFieldBase: fieldBaseStub } },
      });
      await nextTick();
      expect(wrapper.find('.vue-json-pretty-stub').exists()).toBe(false);
      expect(wrapper.find('.o-json-display').text()).toContain('hello');
      wrapper.unmount();
    }
  });

  it('auto renderMode follows isForm for compact vs pretty', async () => {
    const listBinding = makeBinding({ Payload: { k: 1 } }, { isForm: false });
    const listWrapper = mount(OJsonobjectField as any, {
      props: { binding: listBinding, renderMode: 'auto' },
      global: { stubs: { OFieldBase: fieldBaseStub } },
    });
    await nextTick();
    expect(listWrapper.find('.vue-json-pretty-stub').exists()).toBe(false);
    expect(listWrapper.find('.o-json-display').text()).toContain('"k"');
    listWrapper.unmount();

    const formBinding = makeBinding({ Payload: { k: 1 } }, { isForm: true });
    const formWrapper = mount(OJsonobjectField as any, {
      props: { binding: formBinding, renderMode: 'auto' },
      global: { stubs: { OFieldBase: fieldBaseStub } },
    });
    await nextTick();
    expect(formWrapper.find('.vue-json-pretty-stub').exists()).toBe(true);
    formWrapper.unmount();

    const noEnvBinding = makeBinding({ Payload: { k: 1 } });
    (noEnvBinding as any).env = undefined;
    const noEnvWrapper = mount(OJsonobjectField as any, {
      props: { binding: noEnvBinding, renderMode: 'auto' },
      global: { stubs: { OFieldBase: fieldBaseStub } },
    });
    await nextTick();
    // env missing → not table-like → pretty path when value present
    expect(noEnvWrapper.find('.vue-json-pretty-stub').exists()).toBe(true);
    noEnvWrapper.unmount();
  });

  it('null form display stays empty without pretty tree', async () => {
    const binding = makeBinding({ Payload: null });
    const wrapper = mount(OJsonobjectField as any, {
      props: { binding, renderMode: 'form' },
      global: { stubs: { OFieldBase: fieldBaseStub } },
    });
    await nextTick();
    expect(wrapper.find('.vue-json-pretty-stub').exists()).toBe(false);
    expect(wrapper.find('.o-json-display--empty').exists()).toBe(true);
    wrapper.unmount();
  });

  it('edit mode still mounts textarea cell', async () => {
    const binding = makeBinding(
      { Payload: { x: 1 } },
      { isForm: true, isEditMode: true, viewMode: 'edit', fieldPrefix: null }
    );
    const wrapper = mount(OJsonobjectField as any, {
      props: { binding, renderMode: 'form' },
      global: { stubs: { OFieldBase: fieldBaseStub } },
    });
    await nextTick();
    expect(wrapper.find('.o-json-input').exists()).toBe(true);
    wrapper.unmount();
  });
});

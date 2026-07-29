// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount, flushPromises } from '@vue/test-utils';
import { computed, defineComponent, h, nextTick, reactive, ref } from 'vue';import { describe, expect, it, vi } from 'vitest';
import type { UseField } from '@/web/web/composables/useField';
import OHtmlField from './OHtmlField.vue';

vi.mock('@/web/web/i18n', async () => {
  const actual = await vi.importActual<typeof import('@/web/web/i18n')>('@/web/web/i18n');
  return {
    ...actual,
    createTranslate: () => ({
      _t: (msg: string, ...args: unknown[]) => (args.length ? `${msg}:${args.join(',')}` : msg),
    }),
  };
});

vi.mock('dompurify', () => ({
  default: {
    addHook: () => undefined,
    sanitize: (html: string) => String(html),
  },
}));

const editorState = vi.hoisted(() => {
  // eslint-disable-next-line @typescript-eslint/no-require-imports
  const { shallowRef } = require('vue') as typeof import('vue');
  let html = '';
  let emitUpdateOnSetContent = false;
  const active: Record<string, boolean> = {};
  const attrs: Record<string, any> = {};
  const listeners: Record<string, Array<() => void>> = {};
  const editor = {
    isActive: (name: string) => !!active[name],
    getHTML: () => html,
    getAttributes: (name: string) => attrs[name] || {},
    commands: {
      setContent: (next: string) => {
        html = next;
        if (emitUpdateOnSetContent) {
          for (const cb of listeners.update || []) cb();
        }
      },
    },
    chain: () => ({
      focus: () => ({
        toggleBold: () => ({ run: () => undefined }),
        toggleItalic: () => ({ run: () => undefined }),
        toggleBulletList: () => ({ run: () => undefined }),
        toggleOrderedList: () => ({ run: () => undefined }),
        unsetLink: () => ({
          run: () => {
            active.link = false;
          },
        }),
        extendMarkRange: () => ({
          setLink: ({ href }: { href: string }) => ({
            run: () => {
              active.link = true;
              attrs.link = { href };
            },
          }),
        }),
      }),
    }),
    on: (event: string, cb: () => void) => {
      listeners[event] = listeners[event] || [];
      listeners[event].push(cb);
    },
    off: (event: string, cb: () => void) => {
      listeners[event] = (listeners[event] || []).filter(fn => fn !== cb);
    },
    destroy: () => undefined,
    __setHtml(next: string) {
      html = next;
    },
    __setActive(name: string, v: boolean) {
      active[name] = v;
    },
    __setAttr(name: string, value: any) {
      attrs[name] = value;
    },
    __emit(event: string) {
      for (const cb of listeners[event] || []) cb();
    },
  };
  const editorRef = shallowRef<typeof editor | undefined>(editor);
  return {
    editor,
    editorRef,
    get presentRef() {
      return {
        get value() {
          return editorRef.value != null;
        },
        set value(v: boolean) {
          editorRef.value = v ? editor : undefined;
        },
      };
    },
    set emitUpdateOnSetContent(v: boolean) {
      emitUpdateOnSetContent = v;
    },
    reset: () => {
      html = '';
      emitUpdateOnSetContent = false;
      editorRef.value = editor;
      for (const k of Object.keys(active)) delete active[k];
      for (const k of Object.keys(attrs)) delete attrs[k];
      for (const k of Object.keys(listeners)) delete listeners[k];
    },
  };
});

vi.mock('@tiptap/vue-3', () => ({
  useEditor: () => editorState.editorRef,
  EditorContent: defineComponent({
    name: 'EditorContentStub',
    setup() {
      return () => h('div', { class: 'editor-content-stub' });
    },
  }),
}));

vi.mock('@tiptap/starter-kit', () => ({ default: {} }));
vi.mock('@tiptap/extension-link', () => ({
  default: { configure: () => ({}) },
}));

const useFieldMock = vi.hoisted(() => ({
  impl: null as null | ((...args: any[]) => any),
}));

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
  env: Record<string, unknown> = { isForm: true, isEditMode: true, viewMode: 'edit', fieldPrefix: null }
): UseField & { __value: any } {
  const value = ref(record.Terms ?? null);
  const recordRef = ref(record);
  return {
    env,
    prop: 'Terms',
    meta: reactive({ type: 'html' }) as any,
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
  setup(p, { slots, expose }) {
    const fieldValue = () => (p.binding as any).fieldRef();
    const record = () => (p.binding as any).recordRef();
    expose({
      toView: p.toView,
      fromView: p.fromView,
      rules: p.rules,
    });
    return () =>
      h('div', { class: 'field-base-stub' }, [
        slots.edit?.({ fieldValue, record }),
        slots.display?.({ fieldValue, record }),
      ]);
  },
});

describe('OHtmlField', () => {
  it('renders editor and sanitized display html', async () => {
    editorState.reset();
    const binding = makeBinding({ Terms: '<p>Hello</p>' });
    const wrapper = mount(OHtmlField as any, {
      props: { binding, label: 'Terms' },
      global: { stubs: { OFieldBase: fieldBaseStub } },
    });
    await nextTick();
    expect(wrapper.find('.o-htmlfield-edit').exists()).toBe(true);
    expect(wrapper.find('.o-htmlfield-display').exists()).toBe(true);
    expect(wrapper.find('.o-htmlfield-display').html()).toContain('Hello');
    wrapper.unmount();
  });

  it('uses plaintext projection for table/inline and form/auto modes', async () => {
    editorState.reset();
    const binding = makeBinding({ Terms: '<p>Hello <strong>world</strong></p>' });
    for (const renderMode of ['table', 'inline'] as const) {
      const wrapper = mount(OHtmlField as any, {
        props: { binding, renderMode },
        global: { stubs: { OFieldBase: fieldBaseStub } },
      });
      await nextTick();
      expect(wrapper.find('.o-htmlfield-plaintext').text()).toContain('Hello world');
      wrapper.unmount();
    }

    const formWrapper = mount(OHtmlField as any, {
      props: { binding, renderMode: 'form' },
      global: { stubs: { OFieldBase: fieldBaseStub } },
    });
    await nextTick();
    expect(formWrapper.find('.o-htmlfield-display').exists()).toBe(true);
    formWrapper.unmount();

    const listBinding = makeBinding({ Terms: '<p>x</p>' }, { isForm: false });
    const autoWrapper = mount(OHtmlField as any, {
      props: { binding: listBinding, renderMode: 'auto' },
      global: { stubs: { OFieldBase: fieldBaseStub } },
    });
    await nextTick();
    expect(autoWrapper.find('.o-htmlfield-plaintext').exists()).toBe(true);
    autoWrapper.unmount();
  });

  it('wires toolbar actions and toggleLink prompt paths', async () => {
    editorState.reset();
    const binding = makeBinding({ Terms: '<p>Hi</p>' });
    const wrapper = mount(OHtmlField as any, {
      props: { binding },
      global: { stubs: { OFieldBase: fieldBaseStub } },
    });
    await nextTick();
    const buttons = wrapper.findAll('.o-htmlfield-btn');
    expect(buttons.length).toBe(5);
    for (const btn of buttons.slice(0, 4)) {
      await btn.trigger('click');
    }

    const promptFn = vi.fn();
    (window as any).prompt = promptFn;
    promptFn.mockReturnValueOnce(null);
    await buttons[4].trigger('click');

    promptFn.mockReturnValueOnce('   ');
    await buttons[4].trigger('click');

    editorState.editor.__setAttr('link', { href: 'https://prev.example' });
    promptFn.mockReturnValueOnce('https://example.com');
    await buttons[4].trigger('click');
    expect(editorState.editor.isActive('link')).toBe(true);
    expect(promptFn).toHaveBeenCalledWith(expect.any(String), 'https://prev.example');

    editorState.editor.__setActive('link', true);
    await buttons[4].trigger('click');
    expect(editorState.editor.isActive('link')).toBe(false);

    delete (window as any).prompt;
    wrapper.unmount();
  });

  it('syncs editor updates into the binding and skips no-op setContent', async () => {
    editorState.reset();
    // Simulate TipTap emitting update while applying store → editor.
    editorState.emitUpdateOnSetContent = true;

    const binding = makeBinding({ Terms: '<p>Hi</p>' });
    const wrapper = mount(OHtmlField as any, {
      props: { binding },
      global: { stubs: { OFieldBase: fieldBaseStub } },
    });
    await nextTick();
    await flushPromises();

    // Same HTML as editor → setEditorHtml no-op path.
    binding.__value.value = '<p>Hi</p>';
    await nextTick();

    // Null / empty store values clear the editor.
    binding.__value.value = null;
    await nextTick();
    expect(editorState.editor.getHTML()).toBe('');
    binding.__value.value = '';
    await nextTick();

    binding.__value.value = '<p>Hi</p>';
    await nextTick();

    editorState.editor.__setHtml('<p>Edited</p>');
    editorState.editor.__emit('update');
    await nextTick();
    expect(binding.__value.value).toBe('<p>Edited</p>');

    // Same cleaned value → skip model write.
    editorState.editor.__emit('update');
    await nextTick();
    expect(binding.__value.value).toBe('<p>Edited</p>');

    editorState.editor.__setHtml('<p></p>');
    editorState.editor.__emit('update');
    await nextTick();
    expect(binding.__value.value).toBeNull();

    wrapper.unmount();
  });

  it('handles missing editor, late editor mount, and validates rules', async () => {
    editorState.reset();
    editorState.presentRef.value = false;
    const binding = makeBinding({ Terms: '<p>Hi</p>' });
    const wrapper = mount(OHtmlField as any, {
      props: { binding, rules: [{ type: 'string', message: 'x' } as any] },
      global: { stubs: { OFieldBase: fieldBaseStub } },
    });
    await nextTick();
    expect(wrapper.find('.o-htmlfield-toolbar').exists()).toBe(false);

    // TipTap-style late editor creation.
    editorState.presentRef.value = true;
    await nextTick();
    await flushPromises();
    expect(wrapper.find('.o-htmlfield-toolbar').exists()).toBe(true);
    expect(editorState.editor.getHTML()).toBe('<p>Hi</p>');

    // Drop editor again → cleanup off() + destroy on unmount with no editor.
    editorState.presentRef.value = false;
    await nextTick();

    const base = wrapper.findComponent({ name: 'OFieldBaseStub' });
    const rules = (base.props() as any).rules as any[];
    const validator = rules[rules.length - 1].validator as (r: unknown, v: unknown, cb: (e?: Error) => void) => void;
    await new Promise<void>(resolve => validator({}, null, () => resolve()));
    await new Promise<void>(resolve => validator({}, '', () => resolve()));
    await new Promise<void>((resolve, reject) =>
      validator({}, 1, err => (err ? resolve() : reject(new Error('expected error'))))
    );
    await new Promise<void>(resolve => validator({}, 'ok', () => resolve()));

    const toView = (base.props() as any).toView as (v: any) => any;
    const fromView = (base.props() as any).fromView as (v: any) => any;
    expect(toView(null)).toBeNull();
    expect(toView(12)).toBe('12');
    expect(fromView('abc')).toBe('abc');

    wrapper.unmount();
    editorState.presentRef.value = true;
  });

  it('uses useField when binding is omitted', async () => {
    editorState.reset();
    const value = ref('<p>via-store</p>');
    useFieldMock.impl = () =>
      ({
        env: { isForm: true },
        prop: 'Terms',
        meta: reactive({ type: 'html' }),
        fieldRef: () => value,
        fieldRefOf: () => value,
        recordRef: () => computed(() => ({ Terms: value.value })),
        registerFields: () => undefined,
        store: undefined,
        asView: () => ({ fieldValue: () => value }),
      }) as any;
    try {
      const wrapper = mount(OHtmlField as any, {
        props: { store: {} as any, prop: 'Terms' },
        global: { stubs: { OFieldBase: fieldBaseStub } },
      });
      await nextTick();
      expect(wrapper.find('.o-htmlfield-display').html()).toContain('via-store');
      wrapper.unmount();
    } finally {
      useFieldMock.impl = null;
    }
  });
});

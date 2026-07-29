// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount } from '@vue/test-utils';
import { computed, defineComponent, h, nextTick, reactive, ref } from 'vue';
import { describe, expect, it, vi } from 'vitest';
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
    sanitize: (html: string) => String(html).replace(/<script[\s\S]*?<\/script>/gi, ''),
  },
}));

const editorState = vi.hoisted(() => {
  let html = '';
  const listeners: Record<string, Array<() => void>> = {};
  const editor = {
    isActive: () => false,
    getHTML: () => html,
    getAttributes: () => ({}),
    commands: {
      setContent: (next: string) => {
        html = next;
      },
    },
    chain: () => ({
      focus: () => ({
        toggleBold: () => ({ run: () => undefined }),
        toggleItalic: () => ({ run: () => undefined }),
        toggleBulletList: () => ({ run: () => undefined }),
        toggleOrderedList: () => ({ run: () => undefined }),
        unsetLink: () => ({ run: () => undefined }),
        extendMarkRange: () => ({
          setLink: () => ({ run: () => undefined }),
        }),
      }),
    }),
    on: (event: string, cb: () => void) => {
      listeners[event] = listeners[event] || [];
      listeners[event].push(cb);
    },
    destroy: () => undefined,
    __setHtml(next: string) {
      html = next;
    },
    __emit(event: string) {
      for (const cb of listeners[event] || []) cb();
    },
  };
  return { editor, reset: () => { html = ''; } };
});

vi.mock('@tiptap/vue-3', () => ({
  useEditor: () => computed(() => editorState.editor),
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

function makeBinding(record: Record<string, unknown>): UseField & { __value: any } {
  const value = ref(record.Terms ?? null);
  const recordRef = ref(record);
  return {
    env: { isForm: true, isEditMode: true, viewMode: 'edit', fieldPrefix: null },
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

describe('OHtmlField', () => {
  it('renders editor and sanitized display html', async () => {
    editorState.reset();
    const binding = makeBinding({ Terms: '<p>Hello</p>' });
    const wrapper = mount(OHtmlField as any, {
      props: { binding, label: 'Terms' },
      global: {
        stubs: { OFieldBase: fieldBaseStub },
      },
    });
    await nextTick();
    expect(wrapper.find('.o-htmlfield-edit').exists()).toBe(true);
    expect(wrapper.find('.o-htmlfield-display').exists()).toBe(true);
    expect(wrapper.find('.o-htmlfield-display').html()).toContain('Hello');
    wrapper.unmount();
  });

  it('uses plaintext projection in table renderMode', async () => {
    editorState.reset();
    const binding = makeBinding({ Terms: '<p>Hello <strong>world</strong></p>' });
    const wrapper = mount(OHtmlField as any, {
      props: { binding, renderMode: 'table' },
      global: {
        stubs: { OFieldBase: fieldBaseStub },
      },
    });
    await nextTick();
    expect(wrapper.find('.o-htmlfield-plaintext').text()).toContain('Hello world');
    wrapper.unmount();
  });
});

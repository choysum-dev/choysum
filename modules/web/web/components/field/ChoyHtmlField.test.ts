// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, defineComponent, h, ref } from 'vue';
import type { UseField } from '@/web/web/composables/useField';
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import { createLocalFormStore } from '@/web/web/stores/localFormStore';
import * as tiptapVue from '@tiptap/vue-3';
import ChoyHtmlField from './ChoyHtmlField.vue';

function makeBinding(value: string | null, isEditMode = false): UseField & { __value: { value: string | null } } {
  const current = ref(value);
  return {
    env: {
      isForm: true,
      fieldPresentation: 'form',
      isEditMode,
      viewMode: isEditMode ? 'edit' : 'display',
      fieldPrefix: null,
    },
    prop: 'Body',
    meta: { string: 'Body', type: 'html', help: 'Body help' },
    fieldRef: () => current,
    fieldRefOf: () => current,
    recordRef: () => computed(() => ({ Id: '1' })),
    registerFields: () => {},
    store: {},
    asView: (spec: { toView?: (raw: unknown) => unknown; fromView?: (v: unknown) => unknown }) => {
      spec.toView?.(null);
      spec.toView?.('<p>x</p>');
      spec.fromView?.(null);
      spec.fromView?.('<p>x</p>');
      return { fieldValue: () => current, fieldValueOfRow: () => current };
    },
    __value: current,
  } as unknown as UseField & { __value: { value: string | null } };
}

function toolbarButtons(root: { q: (sel: string) => Element | null }): HTMLElement[] {
  const toolbar = root.q('[role="toolbar"]');
  if (!toolbar) return [];
  const out: HTMLElement[] = [];
  for (let node = toolbar.firstChild; node; node = node.nextSibling) {
    if ((node as Element).tagName === 'BUTTON') out.push(node as HTMLElement);
  }
  return out;
}

describe('ChoyHtmlField', () => {
  test('display mode renders sanitized html from the store binding', async () => {
    const m = mountApp(ChoyHtmlField as any, {
      props: {
        binding: makeBinding('<p>Hello</p><script>alert(1)</script>'),
        label: 'Body',
        renderMode: 'form',
      },
    });
    await flushPromises();
    expect(m.q('[data-anchor="choy.field-base"]')).toBeTruthy();
    expect(m.q('[data-slot="form-item"]')).toBeFalsy();
    const prose = m.q('.prose-choy');
    expect(prose?.innerHTML || '').toContain('Hello');
    expect(prose?.innerHTML || '').not.toContain('<script');
    m.unmount();
  });

  test('plaintext display strips tags', async () => {
    const m = mountApp(ChoyHtmlField as any, {
      props: {
        binding: makeBinding('<p>Plain <b>text</b></p>'),
        plaintext: true,
        renderMode: 'form',
      },
    });
    await flushPromises();
    expect(m.q('.choy-field-display-text')?.textContent || '').toContain('Plain');
    expect(m.q('.choy-field-display-text')?.textContent || '').not.toContain('<b>');
    m.unmount();
  });

  test('store and prop resolve a binding without an explicit one', async () => {
    const store = createLocalFormStore({
      fields: [{ name: 'Body', label: 'Body', type: 'varchar' }],
      initialValues: { Body: '<p>Stored</p>' },
    });
    const m = mountApp(ChoyHtmlField as any, {
      props: { store, prop: 'Body', renderMode: 'form' },
    });
    await flushPromises();
    // store+prop mounts without throwing; value display needs a form draft host.
    expect(m.q('[data-anchor="choy.field-base"]')).toBeTruthy();
    expect(m.text()).toContain('Body');
    expect(store.getField('Body')).toBe('<p>Stored</p>');
    m.unmount();
  });

  test('edit mode writes toolbar changes and link prompts back to the binding', async () => {
    const binding = makeBinding('<p>Hi</p>', true);
    const prompts: Array<string | null> = ['https://example.com', 'javascript:alert(1)', '', null];
    const prevPrompt = window.prompt;
    window.prompt = () => (prompts.length ? prompts.shift()! : null);
    const m = mountApp(ChoyHtmlField as any, {
      props: {
        binding,
        renderMode: 'form',
        id: 'body-html',
        required: true,
        showInlineError: true,
      },
    });
    try {
      await flushPromises();
      const buttons = toolbarButtons(m);
      expect(buttons.length).toBe(5);
      expect(m.q('.choy-html-field__edit')?.getAttribute('id')).toBeFalsy();
      const stubDom = (tiptapVue as { __feStubLastEditor?: { view?: { dom?: HTMLElement } } }).__feStubLastEditor
        ?.view?.dom;
      expect(stubDom?.getAttribute('id') || (stubDom as HTMLElement | undefined)?.id).toBe('body-html');
      expect(stubDom?.getAttribute('aria-required')).toBe('true');
      for (const button of buttons) {
        button.click();
        await flushPromises();
      }
      expect(buttons[0]?.className || '').toContain('bg-muted');
      expect(binding.__value.value || '').toContain('Hi');
      for (const button of toolbarButtons(m)) {
        button.click();
        await flushPromises();
      }
      for (let i = 0; i < 3; i++) {
        toolbarButtons(m)[4]?.click();
        await flushPromises();
      }

      const stub = tiptapVue as {
        __feStubEditorRef?: { value: unknown };
        __feStubLastEditor?: { __feStubSetHTML?: (html: string, emit?: boolean) => void };
      };
      const editorRef = stub.__feStubEditorRef;
      expect(editorRef).toBeTruthy();
      editorRef!.value = null;
      binding.__value.value = '<p>ping</p>';
      await flushPromises();
      expect(m.q('[role="toolbar"]')).toBeFalsy();
      editorRef!.value = stub.__feStubLastEditor;
      binding.__value.value = '<p>x</p>';
      await flushPromises();
      stub.__feStubLastEditor?.__feStubSetHTML?.('<p></p>', false);
      binding.__value.value = null;
      await flushPromises();
      binding.__value.value = '';
      await flushPromises();
    } finally {
      m.unmount();
      window.prompt = prevPrompt;
    }
  });

  test('tiptap stub binds view.dom to the mounted contenteditable', async () => {
    let created = false;
    const remountKey = ref(0);
    const Probe = defineComponent({
      name: 'TipTapStubProbe',
      setup() {
        const { useEditor, EditorContent } = tiptapVue as {
          useEditor: (opts: Record<string, unknown>) => { value: any };
          EditorContent: any;
        };
        const editor = useEditor({
          content: '',
          onCreate: () => {
            created = true;
          },
        });
        const ed = editor.value;
        ed.view.dom.setAttribute('data-probe', '1');
        expect(ed.view.dom.getAttribute('data-probe')).toBe('1');
        ed.view.dom.removeAttribute('data-probe');
        expect(ed.view.dom.getAttribute('data-probe')).toBe(null);
        const noop = () => undefined;
        ed.on('probe', noop);
        ed.off('probe', noop);
        ed.off('missing', noop);
        return () => h(EditorContent, { editor: ed, key: remountKey.value });
      },
    });
    const m = mountApp(Probe as any);
    await flushPromises();
    expect(created).toBe(true);
    const stub = tiptapVue as { __feStubLastEditor?: { view?: { dom?: HTMLElement } } };
    const live = stub.__feStubLastEditor?.view?.dom;
    expect(live).toBeTruthy();
    expect((live as HTMLElement).className || '').toContain('fe-stub-editor-content');
    live!.setAttribute('aria-invalid', 'true');
    expect(live!.getAttribute('aria-invalid')).toBe('true');
    // Remount with a new vnode so the stub copies attrs from the previous real DOM node.
    remountKey.value = 1;
    await flushPromises();
    expect(stub.__feStubLastEditor?.view?.dom?.getAttribute('aria-invalid')).toBe('true');
    m.unmount();
  });
});

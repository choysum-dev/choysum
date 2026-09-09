// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h, onMounted } from 'vue';
import { createPinia, setActivePinia } from 'pinia';
import { ElButton, ElDialog, ElForm, ElFormItem, ElInput, ElMessage } from 'element-plus';

import { useI18nStore } from '@/web/web/stores/i18nStore';
import { registerStoreFactory } from '@/web/web/stores/registry';
import { flushPromises, fnRecorder, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import OFieldTranslationsDialog from './OFieldTranslationsDialog.vue';

const DEFAULT_LANGS = [
  { Code: 'en_US', Name: 'English (US)' },
  { Code: 'zh_CN', Name: 'Chinese (Simplified)' },
];

const origSuccess = ElMessage.success;
const origError = ElMessage.error;

function setInput(el: HTMLInputElement, value: string) {
  el.value = value;
  el.dispatchEvent(new Event('input', { bubbles: true }));
}

function inputByLabel(el: HTMLElement, label: string): HTMLInputElement {
  const item = Array.from(el.querySelectorAll('.item')).find(n => n.getAttribute('data-label') === label);
  if (!item) throw new Error('missing ' + label);
  return item.querySelector('input') as HTMLInputElement;
}

function labelsOf(el: HTMLElement): string[] {
  return Array.from(el.querySelectorAll('.item')).map(n => n.getAttribute('data-label') || '');
}

describe('OFieldTranslationsDialog', () => {
  let pinia: ReturnType<typeof createPinia>;
  const getActiveLanguages = fnRecorder(async () => DEFAULT_LANGS.map(r => ({ ...r })));
  const messageSuccess = fnRecorder();
  const messageError = fnRecorder();

  function installStubs() {
    stubSfc(ElDialog as any, {
      name: 'ElDialog',
      props: ['modelValue', 'title'],
      emits: ['opened', 'closed', 'update:modelValue'],
      setup(props: any, { emit, slots }: any) {
        onMounted(() => {
          if (props.modelValue) emit('opened');
        });
        return () =>
          h('div', { class: 'dialog', 'data-title': props.title }, [
            h('button', {
              type: 'button',
              class: 'dialog-emit-closed',
              onClick: () => emit('closed'),
            }),
            slots.default?.(),
            slots.footer?.(),
          ]);
      },
    });
    stubSfc(ElForm as any, {
      name: 'ElForm',
      setup(_: any, { slots }: any) {
        return () => h('form', {}, slots.default?.());
      },
    });
    stubSfc(ElFormItem as any, {
      name: 'ElFormItem',
      props: ['label'],
      setup(props: any, { slots }: any) {
        return () => h('div', { class: 'item', 'data-label': props.label }, slots.default?.());
      },
    });
    stubSfc(ElInput as any, {
      name: 'ElInput',
      props: {
        modelValue: { type: [String, Number, null] as any, default: null },
        maxlength: { type: [String, Number], default: undefined },
        showWordLimit: { type: Boolean, default: false },
      },
      emits: ['update:modelValue'],
      setup(props: any, { emit }: any) {
        return () =>
          h('input', {
            class: 'input',
            value: props.modelValue ?? '',
            'data-maxlength': props.maxlength,
            onInput: (e: Event) => emit('update:modelValue', (e.target as HTMLInputElement).value),
          });
      },
    });
    stubSfc(ElButton as any, {
      name: 'ElButton',
      props: ['type', 'loading', 'nativeType'],
      emits: ['click'],
      setup(props: any, { emit, slots }: any) {
        return () =>
          h(
            'button',
            {
              type: 'button',
              class: 'btn',
              'data-test': props.type === 'primary' ? 'save' : 'cancel',
              onClick: () => emit('click'),
            },
            slots.default?.()
          );
      },
    });
  }

  beforeEach(() => {
    pinia = createPinia();
    setActivePinia(pinia);
    getActiveLanguages.mockReset();
    getActiveLanguages.mockImplementation(async () => DEFAULT_LANGS.map(r => ({ ...r })));
    messageSuccess.mockClear();
    messageError.mockClear();
    (ElMessage as any).success = messageSuccess;
    (ElMessage as any).error = messageError;
    registerStoreFactory('base.Language', () => ({ GetActiveLanguages: getActiveLanguages }));
    // terminologyLang is computed from locale; leave default en_US unless a test sets draftLang.
    void useI18nStore();
    installStubs();
  });

  afterEach(() => {
    (ElMessage as any).success = origSuccess;
    (ElMessage as any).error = origError;
    restoreSfc(ElDialog as any);
    restoreSfc(ElForm as any);
    restoreSfc(ElFormItem as any);
    restoreSfc(ElInput as any);
    restoreSfc(ElButton as any);
  });

  function mountDialog(props: Record<string, unknown>, on?: Record<string, (...args: any[]) => void>) {
    return mountApp(OFieldTranslationsDialog as any, {
      props: {
        modelValue: true,
        recordId: 'lang-1',
        fieldName: 'Name',
        fieldLabel: 'Name',
        ...props,
      },
      on,
      plugins: [pinia],
    });
  }

  function fieldStore(overrides: Record<string, unknown> = {}) {
    return {
      GetFieldTranslations: fnRecorder(async () => ({ en_US: 'Hello', zh_CN: '你好' })),
      UpdateFieldTranslations: fnRecorder(async () => true),
      Browse: fnRecorder(async () => ({ Name: '您好' })),
      ...overrides,
    };
  }

  test('loads active languages and saves UpdateFieldTranslations patch', async () => {
    const store = fieldStore();
    const onSaved = fnRecorder();
    const m = mountDialog({ store }, { onSaved });
    await flushPromises();

    expect(store.GetFieldTranslations.calls[0]).toEqual(['lang-1', 'Name']);
    const labels = labelsOf(m.el);
    expect(labels).toContain('English (US)');
    expect(labels).toContain('Chinese (Simplified)');
    expect(labels.some(l => l.includes('(zh_CN)'))).toBeFalsy();

    setInput(inputByLabel(m.el, 'Chinese (Simplified)'), '您好');
    m.click('[data-test="save"]');
    await flushPromises();

    expect(store.UpdateFieldTranslations.calls[0]).toEqual(['lang-1', 'Name', { zh_CN: '您好' }]);
    expect(store.Browse.calls.length).toBe(1);
    expect(onSaved.calls[0]?.[0]).toBe('您好');
    m.unmount();
  });

  test('clears non-en_US translation with false delete sentinel', async () => {
    const store = fieldStore({
      Browse: fnRecorder(async () => ({ Name: 'Hello' })),
    });
    const onSaved = fnRecorder();
    const m = mountDialog({ store }, { onSaved });
    await flushPromises();

    setInput(inputByLabel(m.el, 'Chinese (Simplified)'), '');
    m.click('[data-test="save"]');
    await flushPromises();

    expect(store.UpdateFieldTranslations.calls[0]).toEqual(['lang-1', 'Name', { zh_CN: false }]);
    expect(onSaved.calls[0]?.[0]).toBe('Hello');
    m.unmount();
  });

  test('overlays draftValue onto draftLang then saves dirty', async () => {
    const store = fieldStore({
      GetFieldTranslations: fnRecorder(async () => ({
        en_US: 'English (US)',
        zh_CN: '英语（美国）',
      })),
      Browse: fnRecorder(async () => ({ Name: '英语（美国）111' })),
    });
    const m = mountDialog({
      store,
      draftValue: '英语（美国）111',
      draftLang: 'zh_CN',
    });
    await flushPromises();

    expect(inputByLabel(m.el, 'English (US)').value).toBe('English (US)');
    expect(inputByLabel(m.el, 'Chinese (Simplified)').value).toBe('英语（美国）111');

    m.click('[data-test="save"]');
    await flushPromises();

    expect(store.UpdateFieldTranslations.calls[0]).toEqual([
      'lang-1',
      'Name',
      { zh_CN: '英语（美国）111' },
    ]);
    m.unmount();
  });

  test('GetFieldTranslations failure leaves empty rows', async () => {
    const store = fieldStore({
      GetFieldTranslations: fnRecorder(async () => {
        throw new Error('load boom');
      }),
    });
    const m = mountDialog({ store });
    await flushPromises();

    expect(messageError.calls.length).toBe(1);
    expect(m.qa('.input').length).toBe(0);
    m.unmount();
  });

  test('injects en_US when active languages omit it', async () => {
    getActiveLanguages.mockImplementation(async () => [{ Code: 'zh_CN', Name: '' }]);
    const store = fieldStore({
      GetFieldTranslations: fnRecorder(async () => ({ en_US: 'Hello', zh_CN: '你好' })),
      Browse: fnRecorder(async () => ({})),
    });
    const m = mountDialog({ store, fieldLabel: undefined });
    await flushPromises();

    const labels = labelsOf(m.el);
    expect(labels[0]).toBe('English (US)');
    expect(labels).toContain('zh_CN');
    expect(m.q('.dialog')?.getAttribute('data-title') || '').toMatch(/Translate/);
    m.unmount();
  });

  test('clears en_US with empty string', async () => {
    const store = fieldStore({
      Browse: fnRecorder(async () => ({ Name: '' })),
    });
    const m = mountDialog({ store });
    await flushPromises();

    setInput(inputByLabel(m.el, 'English (US)'), '');
    m.click('[data-test="save"]');
    await flushPromises();

    expect(store.UpdateFieldTranslations.calls[0]).toEqual(['lang-1', 'Name', { en_US: '' }]);
    m.unmount();
  });

  test('cancel closes via update:modelValue false', async () => {
    const store = fieldStore();
    const onUpdateModelValue = fnRecorder();
    const m = mountDialog({ store }, { 'onUpdate:modelValue': onUpdateModelValue });
    await flushPromises();

    m.click('[data-test="cancel"]');
    await flushPromises();

    expect(onUpdateModelValue.calls.some(args => args[0] === false)).toBeTruthy();
    m.unmount();
  });

  test('reflects maxLength on inputs', async () => {
    const store = fieldStore();
    const m = mountDialog({ store, maxLength: 40 });
    await flushPromises();

    expect(m.q('.input')?.getAttribute('data-maxlength')).toBe('40');
    m.unmount();
  });
});

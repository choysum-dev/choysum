// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h, nextTick } from 'vue';
import { createPinia, setActivePinia } from 'pinia';

import { ChoyMessage } from '../../composables/useChoyMessage';
import { replaceStoreFactory } from '@/web/web/stores/registry';
import { flushPromises, fnRecorder, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import { ChoyDialog, ChoyDialogContent, ChoyDialogTitle } from '@/web/web/components/layout/choyDialog';
import ChoyButton from '@/web/web/components/layout/ChoyButton.vue';
import PropertyLabelTranslationsDialog from './ChoyPropertyLabelTranslationsDialog.vue';

const DEFAULT_LANGS = [
  { Code: 'en_US', Name: 'English (US)' },
  { Code: 'zh_CN', Name: 'Chinese (Simplified)' },
];

const INPUT_SEL = '[data-testid="choy-property-label-translation-input"]';

const origSuccess = ChoyMessage.success;
const origError = ChoyMessage.error;

function setInput(el: HTMLInputElement, value: string) {
  el.value = value;
  el.dispatchEvent(new Event('input', { bubbles: true }));
}

function labelsOf(el: HTMLElement): string[] {
  return Array.from(el.querySelectorAll('label')).map(n => (n.textContent || '').trim());
}

describe('ChoyPropertyLabelTranslationsDialog', () => {
  let pinia: ReturnType<typeof createPinia>;
  let restoreLanguageFactory: (() => void) | undefined;
  const getActiveLanguages = fnRecorder(async () => DEFAULT_LANGS.map(r => ({ ...r })));
  const messageSuccess = fnRecorder();
  const messageError = fnRecorder();

  function installStubs() {
    stubSfc(ChoyDialog as any, {
      name: 'Dialog',
      props: { open: { type: Boolean, default: false } },
      emits: ['update:open'],
      setup(props: any, { slots }: any) {
        return () =>
          h('div', { class: 'dialog', 'data-open': props.open ? '1' : '0' }, [
            props.open ? slots.default?.() : null,
          ]);
      },
    });
    stubSfc(ChoyDialogContent as any, {
      name: 'DialogContent',
      setup(_: any, { slots, attrs }: any) {
        return () => h('div', { class: ['dialog-content', attrs.class] }, slots.default?.());
      },
    });
    stubSfc(ChoyDialogTitle as any, {
      name: 'DialogTitle',
      setup(_: any, { slots }: any) {
        return () => h('div', { class: 'dialog-title' }, slots.default?.());
      },
    });
    stubSfc(ChoyButton as any, {
      name: 'ChoyButton',
      props: {
        type: { type: String, default: 'button' },
        variant: { type: String, default: 'default' },
        disabled: { type: Boolean, default: false },
      },
      emits: ['click'],
      setup(props: any, { emit, slots, attrs }: any) {
        return () =>
          h(
            'button',
            {
              ...attrs,
              type: props.type || 'button',
              class: 'btn',
              disabled: props.disabled || undefined,
              'data-test': props.variant === 'outline' ? 'cancel' : 'save',
              onClick: (e: Event) => emit('click', e),
            },
            slots.default?.(),
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
    ChoyMessage.success = messageSuccess as typeof ChoyMessage.success;
    ChoyMessage.error = messageError as typeof ChoyMessage.error;
    restoreLanguageFactory = replaceStoreFactory('base.Language', () => ({
      GetActiveLanguages: getActiveLanguages,
    }));
    installStubs();
  });

  afterEach(() => {
    restoreLanguageFactory?.();
    restoreLanguageFactory = undefined;
    ChoyMessage.success = origSuccess;
    ChoyMessage.error = origError;
    restoreSfc(ChoyDialog as any);
    restoreSfc(ChoyDialogContent as any);
    restoreSfc(ChoyDialogTitle as any);
    restoreSfc(ChoyButton as any);
  });

  function mountDialog(props: Record<string, unknown> = {}, on?: Record<string, (...args: any[]) => void>) {
    return mountApp(PropertyLabelTranslationsDialog as any, {
      props: {
        modelValue: true,
        translations: { en_US: 'Hello', zh_CN: '你好' },
        ...props,
      },
      on,
      plugins: [pinia],
    });
  }

  test('loadRows with GetActiveLanguages success ensures en_US present', async () => {
    const m = mountDialog();
    await flushPromises();
    const labels = labelsOf(m.el);
    expect(labels[0]).toBe('English (US)');
    expect(labels).toContain('Chinese (Simplified)');
    const inputs = m.qa(INPUT_SEL) as HTMLInputElement[];
    expect(inputs.length).toBe(2);
    expect(inputs[0]?.value).toBe('Hello');
    expect(inputs[1]?.value).toBe('你好');
    m.unmount();
  });

  test('GetActiveLanguages throws → still shows en_US base', async () => {
    getActiveLanguages.mockImplementation(async () => {
      throw new Error('langs boom');
    });
    const m = mountDialog({ translations: { en_US: 'Base' } });
    await flushPromises();
    const labels = labelsOf(m.el);
    expect(labels).toEqual(['English (US)']);
    expect((m.q(INPUT_SEL) as HTMLInputElement)?.value).toBe('Base');
    expect(messageError.calls.length).toBe(0);
    m.unmount();
  });

  test('injects en_US when active languages omit it and includes extra map keys', async () => {
    getActiveLanguages.mockImplementation(async () => [
      { Code: 'zh_CN', Name: '' },
      { Code: '', Name: 'skip' },
    ]);
    const m = mountDialog({
      translations: { en_US: 'Hello', zh_CN: '你好', fr_FR: 'Bonjour' },
    });
    await flushPromises();
    const labels = labelsOf(m.el);
    expect(labels[0]).toBe('English (US)');
    expect(labels).toContain('zh_CN');
    expect(labels).toContain('fr_FR');
    m.unmount();
  });

  test('save keeps en_US even empty; drops empty non-base langs; emit saved; close', async () => {
    const onSaved = fnRecorder();
    const onUpdate = fnRecorder();
    const m = mountDialog(
      { translations: { en_US: 'Hello', zh_CN: '你好' } },
      { onSaved, 'onUpdate:modelValue': onUpdate },
    );
    await flushPromises();
    const inputs = m.qa(INPUT_SEL) as HTMLInputElement[];
    setInput(inputs[0]!, '');
    setInput(inputs[1]!, '');
    m.click('[data-testid="choy-property-label-translations-save"]');
    await nextTick();
    expect(onSaved.calls[0]?.[0]).toEqual({ en_US: '' });
    expect(onUpdate.calls.some(args => args[0] === false)).toBeTruthy();
    m.unmount();
  });

  test('save keeps non-empty non-base langs', async () => {
    const onSaved = fnRecorder();
    const m = mountDialog(
      { translations: { en_US: 'Hello', zh_CN: '你好' } },
      { onSaved },
    );
    await flushPromises();
    const inputs = m.qa(INPUT_SEL) as HTMLInputElement[];
    setInput(inputs[1]!, '您好');
    m.click('[data-testid="choy-property-label-translations-save"]');
    await nextTick();
    expect(onSaved.calls[0]?.[0]).toEqual({ en_US: 'Hello', zh_CN: '您好' });
    m.unmount();
  });

  test('Cancel closes; closed emit', async () => {
    const onUpdate = fnRecorder();
    const onClosed = fnRecorder();
    const m = mountApp(PropertyLabelTranslationsDialog as any, {
      reactiveProps: true,
      props: {
        modelValue: true,
        translations: { en_US: 'x' },
        'onUpdate:modelValue': (v: boolean) => {
          (m.props as any).modelValue = v;
        },
        onClosed,
      },
      plugins: [pinia],
    });
    await flushPromises();
    m.click('[data-test="cancel"]');
    await nextTick();
    await flushPromises();
    expect(onUpdate.calls.some(args => args[0] === false) || (m.props as any).modelValue === false).toBeTruthy();
    expect(onClosed.calls.length).toBe(1);
    m.unmount();
  });

  test('title with and without labelTitle', async () => {
    const withTitle = mountDialog({ labelTitle: 'tax_id' });
    await flushPromises();
    expect((withTitle.q('.dialog-title')?.textContent || '').trim()).toMatch(/Translate:/);
    expect((withTitle.q('.dialog-title')?.textContent || '').trim()).toMatch(/tax_id/);
    withTitle.unmount();

    const bare = mountDialog({ labelTitle: '', translations: null });
    await flushPromises();
    expect((bare.q('.dialog-title')?.textContent || '').trim()).toMatch(/Translate label/);
    bare.unmount();
  });

  test('loadRows outer catch path clears rows and shows error', async () => {
    // Spreading a Proxy that throws hits the outer try/catch in loadRows.
    const badTranslations = new Proxy(
      {},
      {
        ownKeys() {
          throw new Error('spread boom');
        },
        getOwnPropertyDescriptor() {
          throw new Error('spread boom');
        },
      },
    );
    const m = mountDialog({ translations: badTranslations as any });
    await flushPromises();
    expect(messageError.calls.length).toBe(1);
    expect(m.qa(INPUT_SEL).length).toBe(0);
    m.unmount();
  });

  test('sorts non-base languages by label', async () => {
    getActiveLanguages.mockImplementation(async () => [
      { Code: 'de_DE', Name: 'German' },
      { Code: 'fr_FR', Name: 'French' },
      { Code: 'en_US', Name: 'English (US)' },
    ]);
    const m = mountDialog({ translations: {} });
    await flushPromises();
    const labels = labelsOf(m.el);
    expect(labels[0]).toBe('English (US)');
    expect(labels.indexOf('French')).toBeLessThan(labels.indexOf('German'));
    m.unmount();
  });

  test('save after load failure still emits empty en_US base key', async () => {
    const badTranslations = new Proxy(
      {},
      {
        ownKeys() {
          throw new Error('spread boom');
        },
        getOwnPropertyDescriptor() {
          throw new Error('spread boom');
        },
      },
    );
    const onSaved = fnRecorder();
    const m = mountDialog({ translations: badTranslations as any }, { onSaved });
    await flushPromises();
    expect(m.qa(INPUT_SEL).length).toBe(0);
    m.click('[data-testid="choy-property-label-translations-save"]');
    await nextTick();
    expect(onSaved.calls[0]?.[0]).toEqual({ en_US: '' });
    m.unmount();
  });
});

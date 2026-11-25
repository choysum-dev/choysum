// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h, nextTick, provide, reactive } from 'vue';
import { createPinia, setActivePinia } from 'pinia';

import { ChoyMessage } from '../../composables/useChoyMessage';
import { replaceStoreFactory } from '@/web/web/stores/registry';
import { flushPromises, fnRecorder, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import { ChoyDialog, ChoyDialogContent, ChoyDialogTitle } from '@/web/web/components/layout/choyDialog';
import ChoyButton from '@/web/web/components/button/ChoyButton.vue';
import ChoyFormView from '@/web/web/components/view/ChoyFormView.vue';
import type { UiModelStore } from '@/web/web/stores/uiModelStore';
import PropertyLabelTranslationsDialog from './ChoyPropertyLabelTranslationsDialog.vue';

const DEFAULT_LANGS = [
  { Code: 'en_US', Name: 'English (US)' },
  { Code: 'zh_CN', Name: 'Chinese (Simplified)' },
];

const INPUT_SEL = 'input.choy-varchar-input';

const origSuccess = ChoyMessage.success;
const origError = ChoyMessage.error;

type MountedDialog = ReturnType<typeof mountApp>;

function unwrapRef<T>(v: T | { value: T } | undefined): T | undefined {
  if (v == null) return undefined;
  if (typeof v === 'object' && v !== null && 'value' in (v as object)) {
    return (v as { value: T }).value;
  }
  return v as T;
}

function dialogStore(m: MountedDialog): UiModelStore | null | undefined {
  const ss = m.setupState() as { formStore?: UiModelStore | { value: UiModelStore | null } };
  return unwrapRef(ss.formStore) ?? null;
}

function dialogFieldCodes(m: MountedDialog): string[] {
  const ss = m.setupState() as { fieldCodes?: string[] | { value: string[] } };
  return unwrapRef(ss.fieldCodes) ?? [];
}

function labelsOf(m: MountedDialog): string[] {
  const store = dialogStore(m);
  return dialogFieldCodes(m).map(code => {
    const meta = store?.getFieldMeta(code) as { string?: string } | undefined;
    return String(meta?.string ?? code);
  });
}

function fieldValueByCode(m: MountedDialog, code: string): string {
  return String(dialogStore(m)?.getField(code) ?? '');
}

function setFieldByCode(m: MountedDialog, code: string, value: string): void {
  dialogStore(m)?.setField(code, value);
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
    stubSfc(ChoyFormView as any, {
      name: 'ChoyFormView',
      props: {
        store: Object,
        initialValues: Object,
        viewMode: String,
        embedded: Boolean,
        showHeader: Boolean,
        showActions: Boolean,
        submitHandler: Function,
        cancelHandler: Function,
      },
      setup(props: any, { slots, attrs, expose }: any) {
        const formData = reactive({ ...(props.initialValues || {}) });
        const syncStoreField = (path: string, value: unknown) => {
          (formData as any)[path] = value;
          const store = props.store as UiModelStore | null | undefined;
          store?.setField?.(path, value);
        };
        provide('form-root', {
          get mode() {
            return 'create';
          },
          get draft() {
            return formData;
          },
          setField(path: string, value: unknown) {
            syncStoreField(path, value);
          },
          getField(path: string) {
            return (formData as any)[path];
          },
          onChange() {},
        });
        expose({
          submit: () => {
            const store = props.store as UiModelStore | null | undefined;
            const merged = store?.getValues ? { ...formData, ...store.getValues() } : { ...formData };
            return props.submitHandler?.({
              formData: merged,
              mode: 'create',
              data: merged,
              defaultSubmit: async () => null,
            });
          },
          cancel: async () => {
            await props.cancelHandler?.();
          },
        });
        return () => h('div', { ...attrs, class: ['choy-form-view-stub', attrs.class] }, slots.default?.({ formData }));
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
    restoreSfc(ChoyFormView as any);
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
    const labels = labelsOf(m);
    expect(labels[0]).toBe('English (US)');
    expect(labels).toContain('Chinese (Simplified)');
    expect(dialogFieldCodes(m).length).toBe(2);
    expect(fieldValueByCode(m, 'en_US')).toBe('Hello');
    expect(fieldValueByCode(m, 'zh_CN')).toBe('你好');
    m.unmount();
  });

  test('GetActiveLanguages throws → still shows en_US base', async () => {
    getActiveLanguages.mockImplementation(async () => {
      throw new Error('langs boom');
    });
    const m = mountDialog({ translations: { en_US: 'Base' } });
    await flushPromises();
    const labels = labelsOf(m);
    expect(labels).toEqual(['English (US)']);
    expect(fieldValueByCode(m, 'en_US')).toBe('Base');
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
    const labels = labelsOf(m);
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
    setFieldByCode(m, 'en_US', '');
    setFieldByCode(m, 'zh_CN', '');
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
    setFieldByCode(m, 'zh_CN', '您好');
    m.click('[data-testid="choy-property-label-translations-save"]');
    await nextTick();
    expect(onSaved.calls[0]?.[0]).toEqual({ en_US: 'Hello', zh_CN: '您好' });
    m.unmount();
  });

  test('ignores stale loadRows from a previous opening', async () => {
    let releaseFirst!: () => void;
    const firstGate = new Promise<void>(resolve => {
      releaseFirst = resolve;
    });
    let calls = 0;
    getActiveLanguages.mockImplementation(async () => {
      calls += 1;
      if (calls === 1) await firstGate;
      return DEFAULT_LANGS.map(r => ({ ...r }));
    });

    const m = mountApp(PropertyLabelTranslationsDialog as any, {
      reactiveProps: true,
      props: {
        modelValue: true,
        translations: { en_US: 'First' },
        'onUpdate:modelValue': (v: boolean) => {
          (m.props as any).modelValue = v;
        },
      },
      plugins: [pinia],
    });
    await flushPromises();
    expect(m.qa(INPUT_SEL).length).toBe(0);

    (m.props as any).modelValue = false;
    await nextTick();
    await flushPromises();

    (m.props as any).translations = { en_US: 'Second', zh_CN: '第二' };
    (m.props as any).modelValue = true;
    await flushPromises();
    expect(fieldValueByCode(m, 'en_US')).toBe('Second');

    releaseFirst();
    await flushPromises();
    expect(fieldValueByCode(m, 'en_US')).toBe('Second');
    expect(fieldValueByCode(m, 'zh_CN')).toBe('第二');
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
    const labels = labelsOf(m);
    expect(labels[0]).toBe('English (US)');
    expect(labels.indexOf('French')).toBeLessThan(labels.indexOf('German'));
    m.unmount();
  });

  test('save after load failure closes without wiping translations', async () => {
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
    const onUpdate = fnRecorder();
    const m = mountDialog(
      { translations: badTranslations as any },
      { onSaved, 'onUpdate:modelValue': onUpdate },
    );
    await flushPromises();
    expect(m.qa(INPUT_SEL).length).toBe(0);
    const saveBtn = m.q('[data-testid="choy-property-label-translations-save"]') as HTMLButtonElement | null;
    expect(saveBtn?.disabled).toBe(true);
    // Disabled button may not fire DOM click; exercise the empty-rows guard directly.
    const ss = m.setupState() as { handleSave?: () => void };
    ss.handleSave?.();
    await nextTick();
    expect(onSaved.calls.length).toBe(0);
    expect(onUpdate.calls.some(args => args[0] === false)).toBeTruthy();
    m.unmount();
  });

  test('save seeds empty en_US when rows omit the base language', async () => {
    const onSaved = fnRecorder();
    const m = mountDialog({ translations: { zh_CN: '你好' } }, { onSaved });
    await flushPromises();
    const ss = m.setupState() as {
      fieldCodes?: string[] | { value: string[] };
      handleSave?: () => void;
    };
    const codes = Array.isArray((ss.fieldCodes as { value?: unknown } | undefined)?.value)
      ? (ss.fieldCodes as { value: string[] }).value
      : (ss.fieldCodes as string[]);
    const baseIdx = codes.findIndex(c => c === 'en_US');
    expect(baseIdx).toBeGreaterThanOrEqual(0);
    codes.splice(baseIdx, 1);
    expect(codes.some(c => c === 'en_US')).toBe(false);
    ss.handleSave?.();
    await nextTick();
    expect(onSaved.calls[0]?.[0]).toEqual({ zh_CN: '你好', en_US: '' });
    m.unmount();
  });
});

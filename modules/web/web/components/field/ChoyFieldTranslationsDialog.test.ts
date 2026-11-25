// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h, provide, reactive } from 'vue';
import { ChoyMessage } from '../../composables/useChoyMessage';
import { createPinia, setActivePinia } from 'pinia';

import { useI18nStore } from '@/web/web/stores/i18nStore';
import { replaceStoreFactory } from '@/web/web/stores/registry';
import { flushPromises, fnRecorder, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import { ChoyDialog, ChoyDialogContent, ChoyDialogTitle } from '@/web/web/components/layout/choyDialog';
import ChoyButton from '@/web/web/components/button/ChoyButton.vue';
import ChoyFormView from '@/web/web/components/view/ChoyFormView.vue';
import type { UiModelStore } from '@/web/web/stores/uiModelStore';
import FieldTranslationsDialog from './ChoyFieldTranslationsDialog.vue';

const DEFAULT_LANGS = [
  { Code: 'en_US', Name: 'English (US)' },
  { Code: 'zh_CN', Name: 'Chinese (Simplified)' },
];

const INPUT_SEL = 'input.choy-varchar-input';

const origSuccess = ChoyMessage.success;
const origError = ChoyMessage.error;

type MountedDialog = ReturnType<typeof mountApp>;

function unwrapRef<T>(v: T | { value: T } | undefined): T | undefined {
  // Object(v) === v excludes null (typeof null === 'object').
  if (typeof v !== 'object' || Object(v) !== v) return v as T | undefined;
  if ('value' in v) return (v as { value: T }).value;
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
  const ss = m.setupState() as { rowMeta?: Map<string, { label: string }> | { value: Map<string, { label: string }> } };
  const meta = unwrapRef(ss.rowMeta);
  return dialogFieldCodes(m).map(code => meta?.get(code)?.label ?? code);
}

function setFieldByLabel(m: MountedDialog, label: string, value: string): void {
  const labels = labelsOf(m);
  const idx = labels.findIndex(l => l === label);
  if (idx < 0) throw new Error('missing ' + label);
  const code = dialogFieldCodes(m)[idx];
  dialogStore(m)?.setField(code, value);
}

function fieldValueByLabel(m: MountedDialog, label: string): string {
  const labels = labelsOf(m);
  const idx = labels.findIndex(l => l === label);
  if (idx < 0) throw new Error('missing ' + label);
  const code = dialogFieldCodes(m)[idx];
  return String(dialogStore(m)?.getField(code) ?? '');
}

describe('FieldTranslationsDialog', () => {
  let pinia: ReturnType<typeof createPinia>;
  let restoreLanguageFactory: (() => void) | undefined;
  const getActiveLanguages = fnRecorder(async () => DEFAULT_LANGS.map(r => ({ ...r })));
  const messageSuccess = fnRecorder();
  const messageError = fnRecorder();

  function installStubs() {
    stubSfc(ChoyDialog as any, {
      name: 'Dialog',
      props: {
        open: { type: Boolean, default: false },
      },
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
      setup(props: any, { emit, slots }: any) {
        return () =>
          h(
            'button',
            {
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
    // Store lang defaults to en_US unless a test sets draftLang.
    void useI18nStore();
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
    return mountApp(FieldTranslationsDialog as any, {
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
    const labels = labelsOf(m);
    expect(labels).toContain('English (US)');
    expect(labels).toContain('Chinese (Simplified)');
    expect(labels.some(l => l.includes('(zh_CN)'))).toBeFalsy();

    setFieldByLabel(m, 'Chinese (Simplified)', '您好');
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

    setFieldByLabel(m, 'Chinese (Simplified)', '');
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

    expect(fieldValueByLabel(m, 'English (US)')).toBe('English (US)');
    expect(fieldValueByLabel(m, 'Chinese (Simplified)')).toBe('英语（美国）111');

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
    expect(m.qa(INPUT_SEL).length).toBe(0);
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

    const labels = labelsOf(m);
    expect(labels[0]).toBe('English (US)');
    expect(labels).toContain('zh_CN');
    expect((m.q('.dialog-title')?.textContent || '').trim()).toMatch(/Translate/);
    m.unmount();
  });

  test('clears en_US with empty string', async () => {
    const store = fieldStore({
      Browse: fnRecorder(async () => ({ Name: '' })),
    });
    const m = mountDialog({ store });
    await flushPromises();

    setFieldByLabel(m, 'English (US)', '');
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

    const meta = dialogStore(m)?.getFieldMeta('en_US') as { size?: number } | undefined;
    expect(meta?.size).toBe(40);
    m.unmount();
  });
  test('resolveDraftLang catch and save error via setupState', async () => {
    getActiveLanguages.mockImplementation(async () => [
      { Code: 'fr_FR', Name: 'French' },
      { Code: 'de_DE', Name: 'German' },
    ]);
    const store = fieldStore({
      GetFieldTranslations: fnRecorder(async () => ({ fr_FR: 'Bonjour', de_DE: 'Hallo', en_US: 'Hello' })),
      UpdateFieldTranslations: fnRecorder(async () => {
        throw new Error('save boom');
      }),
    });
    const m = mountDialog({ store, draftLang: '' });
    await flushPromises();
    // fr_FR vs de_DE ordering hits label.localeCompare (neither side is en_US).
    const labels = labelsOf(m);
    expect(labels.indexOf('French')).toBeGreaterThan(0);
    expect(labels.indexOf('German')).toBeGreaterThan(0);
    const ss = m.setupState() as any;
    expect(typeof ss.resolveDraftLang).toBe('function');
    // Happy path: empty draftLang reads lang from i18n store.
    expect(typeof ss.resolveDraftLang()).toBe('string');
    // Catch path: useI18nStore throws without an active pinia.
    setActivePinia(undefined as any);
    expect(ss.resolveDraftLang()).toBe('');
    setActivePinia(pinia);
    setFieldByLabel(m, 'French', 'Salut');
    m.click('[data-test="save"]');
    await flushPromises();
    expect(messageError.calls.length).toBeGreaterThanOrEqual(1);
    m.unmount();
  });

});

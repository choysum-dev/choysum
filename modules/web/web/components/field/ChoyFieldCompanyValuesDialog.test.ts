// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h } from 'vue';
import { ChoyMessage } from '../../composables/useChoyMessage';
import { createPinia, setActivePinia } from 'pinia';

import { useAuthStore } from '@/auth/web/stores/auth';
import { replaceStoreFactory } from '@/web/web/stores/registry';
import { flushPromises, fnRecorder, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import { ChoyDialog, ChoyDialogContent, ChoyDialogTitle } from '@/web/web/components/layout/choyDialog';
import ChoyButton from '@/web/web/components/layout/ChoyButton.vue';
import FieldCompanyValuesDialog from './ChoyFieldCompanyValuesDialog.vue';

const DEFAULT_COMPANY_ROWS = [
  { Id: 'comp_main', DisplayName: 'Main Company' },
  { Id: 'comp_eu', DisplayName: 'EU Company' },
];

const DEFAULT_AUTH_METADATA = {
  allowedCompanyIds: ['comp_main', 'comp_eu'],
  enabledCompanyIds: ['comp_main'],
  activeCompanyId: 'comp_main',
};

const ROW_SEL = '.choy-field-company-values-dialog__row';
const INPUT_SEL = 'input.choy-field-company-values-dialog__input';

const origSuccess = ChoyMessage.success;
const origError = ChoyMessage.error;

function setInput(el: HTMLInputElement, value: string) {
  el.value = value;
  el.dispatchEvent(new Event('input', { bubbles: true }));
}

function inputByLabel(el: HTMLElement, label: string): HTMLInputElement {
  const row = Array.from(el.querySelectorAll(ROW_SEL)).find(n => {
    const lab = n.querySelector('.choy-field-company-values-dialog__label');
    return (lab?.textContent || '').trim() === label;
  });
  if (!row) throw new Error('missing ' + label);
  return row.querySelector(INPUT_SEL) as HTMLInputElement;
}

function labelsOf(el: HTMLElement): string[] {
  return Array.from(el.querySelectorAll(ROW_SEL)).map(n => {
    const lab = n.querySelector('.choy-field-company-values-dialog__label');
    return (lab?.textContent || '').trim();
  });
}

describe('FieldCompanyValuesDialog', () => {
  let pinia: ReturnType<typeof createPinia>;
  let restoreCompanyFactory: (() => void) | undefined;
  const companySearch = fnRecorder(async () => DEFAULT_COMPANY_ROWS.map(r => ({ ...r })));
  const messageSuccess = fnRecorder();
  const messageError = fnRecorder();

  function seedAuth(meta: Record<string, unknown> = DEFAULT_AUTH_METADATA) {
    useAuthStore().identity = { metadata: { ...meta } } as any;
  }

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
    companySearch.mockReset();
    companySearch.mockImplementation(async () => DEFAULT_COMPANY_ROWS.map(r => ({ ...r })));
    messageSuccess.mockClear();
    messageError.mockClear();
    ChoyMessage.success = messageSuccess as typeof ChoyMessage.success;
    ChoyMessage.error = messageError as typeof ChoyMessage.error;
    restoreCompanyFactory = replaceStoreFactory('base.Company', () => ({ Search: companySearch }));
    seedAuth();
    installStubs();
  });

  afterEach(() => {
    restoreCompanyFactory?.();
    restoreCompanyFactory = undefined;
    ChoyMessage.success = origSuccess;
    ChoyMessage.error = origError;
    restoreSfc(ChoyDialog as any);
    restoreSfc(ChoyDialogContent as any);
    restoreSfc(ChoyDialogTitle as any);
    restoreSfc(ChoyButton as any);
  });

  function mountDialog(props: Record<string, unknown> = {}, on?: Record<string, (...args: any[]) => void>) {
    return mountApp(FieldCompanyValuesDialog as any, {
      props: {
        modelValue: true,
        recordId: 'prod-1',
        fieldName: 'Cost',
        fieldLabel: 'Cost',
        ...props,
      },
      on,
      plugins: [pinia],
    });
  }

  function fieldStore(overrides: Record<string, unknown> = {}) {
    return {
      GetFieldCompanyValues: fnRecorder(async () => ({ comp_main: '12.5', comp_eu: '11.0' })),
      UpdateFieldCompanyValues: fnRecorder(async () => true),
      Browse: fnRecorder(async () => ({ Cost: '11.5' })),
      ...overrides,
    };
  }

  test('loads allowed companies and saves number-coerced patch', async () => {
    const store = fieldStore();
    const onSaved = fnRecorder();
    const m = mountDialog({ store, fieldType: 'number' }, { onSaved });
    await flushPromises();

    expect(store.GetFieldCompanyValues.calls[0]).toEqual(['prod-1', 'Cost']);
    const labels = labelsOf(m.el);
    expect(labels).toContain('Main Company');
    expect(labels).toContain('EU Company');

    setInput(inputByLabel(m.el, 'EU Company'), '11.5');
    m.click('[data-test="save"]');
    await flushPromises();

    expect(store.UpdateFieldCompanyValues.calls[0]).toEqual(['prod-1', 'Cost', { comp_eu: 11.5 }]);
    expect(store.Browse.calls.length).toBe(1);
    expect(onSaved.calls[0]?.[0]).toBe('11.5');
    m.unmount();
  });

  test('clears any company value with false delete sentinel', async () => {
    const store = fieldStore({
      Browse: fnRecorder(async () => ({ Cost: '12.5' })),
    });
    const onSaved = fnRecorder();
    const m = mountDialog({ store }, { onSaved });
    await flushPromises();

    setInput(inputByLabel(m.el, 'Main Company'), '');
    m.click('[data-test="save"]');
    await flushPromises();

    expect(store.UpdateFieldCompanyValues.calls[0]).toEqual(['prod-1', 'Cost', { comp_main: false }]);
    expect(onSaved.calls[0]?.[0]).toBe('12.5');
    m.unmount();
  });

  test('overlays draftValue onto draftCompanyId then saves dirty', async () => {
    const store = fieldStore({
      Browse: fnRecorder(async () => ({ Cost: '99' })),
    });
    const m = mountDialog({
      store,
      draftValue: '99',
      draftCompanyId: 'comp_main',
    });
    await flushPromises();

    expect(inputByLabel(m.el, 'Main Company').value).toBe('99');
    m.click('[data-test="save"]');
    await flushPromises();

    expect(store.UpdateFieldCompanyValues.calls[0]).toEqual(['prod-1', 'Cost', { comp_main: '99' }]);
    m.unmount();
  });

  test('GetFieldCompanyValues failure leaves empty rows', async () => {
    const store = fieldStore({
      GetFieldCompanyValues: fnRecorder(async () => {
        throw new Error('load boom');
      }),
    });
    const m = mountDialog({ store });
    await flushPromises();

    expect(messageError.calls.length).toBe(1);
    expect(m.qa(INPUT_SEL).length).toBe(0);
    m.unmount();
  });

  test('empty allowlist falls back to enabled ∪ map keys', async () => {
    seedAuth({
      allowedCompanyIds: [],
      enabledCompanyIds: ['comp_main'],
      activeCompanyId: 'comp_main',
    });
    const store = fieldStore({
      GetFieldCompanyValues: fnRecorder(async () => ({ comp_main: '1', comp_eu: '2' })),
    });
    const m = mountDialog({ store, fieldLabel: undefined });
    await flushPromises();

    const labels = labelsOf(m.el);
    expect(labels).toContain('Main Company');
    expect(labels).toContain('EU Company');
    expect(labels.length).toBe(2);
    expect((m.q('.dialog-title')?.textContent || '').trim()).toMatch(/Company values/);
    m.unmount();
  });

  test('coerces boolean true and false tokens in the patch', async () => {
    seedAuth({
      allowedCompanyIds: ['comp_main', 'comp_eu'],
      enabledCompanyIds: ['comp_main', 'comp_eu'],
      activeCompanyId: 'comp_main',
    });
    companySearch.mockImplementation(async () => [
      { Id: 'comp_main', DisplayName: 'Main' },
      { Id: 'comp_eu', DisplayName: 'EU' },
    ]);
    const store = fieldStore({
      GetFieldCompanyValues: fnRecorder(async () => ({ comp_main: true })),
      Browse: fnRecorder(async () => ({ Flag: false })),
    });
    const m = mountDialog({
      store,
      fieldName: 'Flag',
      fieldLabel: 'Flag',
      fieldType: 'boolean',
    });
    await flushPromises();

    setInput(inputByLabel(m.el, 'EU'), 'yes');
    setInput(inputByLabel(m.el, 'Main'), '0');
    m.click('[data-test="save"]');
    await flushPromises();

    expect(store.UpdateFieldCompanyValues.calls[0]).toEqual([
      'prod-1',
      'Flag',
      { comp_eu: true, comp_main: false },
    ]);
    m.unmount();
  });

  test('save error keeps dialog open without saved emit', async () => {
    const store = fieldStore({
      UpdateFieldCompanyValues: fnRecorder(async () => {
        throw new Error('save boom');
      }),
    });
    const onSaved = fnRecorder();
    const onUpdateModelValue = fnRecorder();
    const m = mountDialog({ store }, { onSaved, 'onUpdate:modelValue': onUpdateModelValue });
    await flushPromises();

    setInput(inputByLabel(m.el, 'EU Company'), '9');
    m.click('[data-test="save"]');
    await flushPromises();

    expect(messageError.calls.length).toBe(1);
    expect(onSaved.calls.length).toBe(0);
    expect(onUpdateModelValue.calls.some(args => args[0] === false)).toBeFalsy();
    m.unmount();
  });

  test('no dirty still Browses, closes, and emits saved', async () => {
    const store = fieldStore({
      Browse: fnRecorder(async () => ({ Cost: '12.5' })),
    });
    const onSaved = fnRecorder();
    const onUpdateModelValue = fnRecorder();
    const m = mountDialog({ store }, { onSaved, 'onUpdate:modelValue': onUpdateModelValue });
    await flushPromises();

    m.click('[data-test="save"]');
    await flushPromises();

    expect(store.UpdateFieldCompanyValues.calls.length).toBe(0);
    expect(store.Browse.calls.length).toBe(1);
    expect(onSaved.calls.length).toBe(1);
    expect(onUpdateModelValue.calls.some(args => args[0] === false)).toBeTruthy();
    m.unmount();
  });
  test('formatCaughtError and coercePatchValue edges via setupState', async () => {
    const store = fieldStore();
    const m = mountDialog({ store, fieldType: 'boolean', draftCompanyId: '' });
    await flushPromises();
    const ss = m.setupState() as any;
    expect(typeof ss.formatCaughtError).toBe('function');
    expect(typeof ss.coercePatchValue).toBe('function');
    expect(ss.formatCaughtError(null, 'fb')).toBe('fb');
    expect(ss.formatCaughtError('  boom  ', 'fb')).toBe('boom');
    expect(ss.formatCaughtError('   ', 'fb')).toBe('fb');
    expect(ss.formatCaughtError({ message: '  m  ' }, 'fb')).toBe('m');
    expect(ss.formatCaughtError({ message: '  ' }, 'fb')).toBe('fb');
    expect(ss.formatCaughtError(12, 'fb')).toBe('fb');
    expect(ss.coercePatchValue('')).toBe(true);
    expect(ss.coercePatchValue('yes')).toBe(true);
    expect(ss.coercePatchValue('no')).toBe(false);
    expect(typeof ss.resolveDraftCompanyId).toBe('function');
    expect(ss.resolveDraftCompanyId()).toBe('comp_main');
    m.unmount();

    const intMount = mountDialog({ store: fieldStore(), fieldType: 'integer' });
    await flushPromises();
    const iss = intMount.setupState() as any;
    expect(iss.coercePatchValue('42')).toBe(42);
    expect(iss.coercePatchValue('nope')).toBe('nope');
    intMount.unmount();

    seedAuth({
      allowedCompanyIds: ['comp_main'],
      enabledCompanyIds: ['comp_main'],
      activeCompanyId: 'comp_main',
    });
    const draftMount = mountDialog({ store: fieldStore(), draftCompanyId: '  ' });
    await flushPromises();
    const dss = draftMount.setupState() as any;
    expect(dss.resolveDraftCompanyId()).toBe('comp_main');
    draftMount.unmount();

    // readAuthCompanyMeta catch when Pinia/auth is unavailable.
    const catchMount = mountDialog({ store: fieldStore() });
    await flushPromises();
    const css = catchMount.setupState() as any;
    expect(typeof css.readAuthCompanyMeta).toBe('function');
    setActivePinia(undefined as any);
    const meta = css.readAuthCompanyMeta();
    expect(meta.activeCompanyId).toBe('');
    expect(meta.allowedCompanyIds).toEqual([]);
    setActivePinia(pinia);
    catchMount.unmount();
  });

});

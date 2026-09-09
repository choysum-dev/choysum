// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h, onMounted } from 'vue';
import { createPinia, setActivePinia } from 'pinia';
import { ElButton, ElDialog, ElForm, ElFormItem, ElInput, ElMessage } from 'element-plus';

import { useAuthStore } from '@/auth/web/stores/auth';
import { replaceStoreFactory } from '@/web/web/stores/registry';
import { flushPromises, fnRecorder, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import OFieldCompanyValuesDialog from './OFieldCompanyValuesDialog.vue';

const DEFAULT_COMPANY_ROWS = [
  { Id: 'comp_main', DisplayName: 'Main Company' },
  { Id: 'comp_eu', DisplayName: 'EU Company' },
];

const DEFAULT_AUTH_METADATA = {
  allowedCompanyIds: ['comp_main', 'comp_eu'],
  enabledCompanyIds: ['comp_main'],
  activeCompanyId: 'comp_main',
};

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

describe('OFieldCompanyValuesDialog', () => {
  let pinia: ReturnType<typeof createPinia>;
  let restoreCompanyFactory: (() => void) | undefined;
  const companySearch = fnRecorder(async () => DEFAULT_COMPANY_ROWS.map(r => ({ ...r })));
  const messageSuccess = fnRecorder();
  const messageError = fnRecorder();

  function seedAuth(meta: Record<string, unknown> = DEFAULT_AUTH_METADATA) {
    useAuthStore().identity = { metadata: { ...meta } } as any;
  }

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
    companySearch.mockReset();
    companySearch.mockImplementation(async () => DEFAULT_COMPANY_ROWS.map(r => ({ ...r })));
    messageSuccess.mockClear();
    messageError.mockClear();
    (ElMessage as any).success = messageSuccess;
    (ElMessage as any).error = messageError;
    restoreCompanyFactory = replaceStoreFactory('base.Company', () => ({ Search: companySearch }));
    seedAuth();
    installStubs();
  });

  afterEach(() => {
    restoreCompanyFactory?.();
    restoreCompanyFactory = undefined;
    (ElMessage as any).success = origSuccess;
    (ElMessage as any).error = origError;
    restoreSfc(ElDialog as any);
    restoreSfc(ElForm as any);
    restoreSfc(ElFormItem as any);
    restoreSfc(ElInput as any);
    restoreSfc(ElButton as any);
  });

  function mountDialog(props: Record<string, unknown>, on?: Record<string, (...args: any[]) => void>) {
    return mountApp(OFieldCompanyValuesDialog as any, {
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
    expect(m.qa('.input').length).toBe(0);
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
    expect(m.q('.dialog')?.getAttribute('data-title') || '').toMatch(/Company values/);
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
});

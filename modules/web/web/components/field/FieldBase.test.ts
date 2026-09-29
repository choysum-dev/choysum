// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Mount wiring for FieldBase label/help/translate/company-values and list-editing-row-id gate.
 * Dialog child SFCs are stubbed; their own suites remain smoke until store-heavy mounts are green.
 */

import { computed, defineComponent, h, nextTick, ref } from 'vue';

import { createTermReference } from '@/core/service/i18n';
import type { UseField } from '@/web/web/composables/useField';
import { flushPromises, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import ChoyButton from '@/web/web/components/layout/ChoyButton.vue';
import ChoyTableColumn from '@/web/web/components/table/ChoyTableColumn.vue';
import FieldBase from './FieldBase.vue';
import FieldCompanyValuesDialog from './ChoyFieldCompanyValuesDialog.vue';
import FieldTranslationsDialog from './ChoyFieldTranslationsDialog.vue';

function makeBinding(
  meta?: {
    string?: string;
    stringText?: ReturnType<typeof createTermReference>;
    help?: string;
    helpText?: ReturnType<typeof createTermReference>;
    translate?: boolean;
    companyDependent?: boolean;
    isReadonly?: boolean;
    type?: string;
  },
  opts?: { recordId?: string | null; isEditMode?: boolean; fieldPrefix?: string | null; store?: any }
): UseField & { __value: any } {
  const value = ref('x');
  const recordId = opts && 'recordId' in opts ? opts.recordId : '1';
  const record = ref(recordId ? { Id: recordId } : {});
  return {
    env: {
      isForm: true,
      isEditMode: opts?.isEditMode ?? true,
      viewMode: opts?.isEditMode === false ? 'readonly' : 'edit',
      fieldPrefix: opts?.fieldPrefix ?? null,
    },
    prop: 'AccessTokenId',
    meta: meta as any,
    fieldRef: () => value as any,
    fieldRefOf: () => value as any,
    recordRef: () => computed(() => record.value) as any,
    registerFields: () => {},
    store: ('store' in (opts || {}) ? opts?.store : {}) as any,
    asView: () => ({ fieldValue: () => value }) as any,
    __value: value,
  } as any;
}

const EditStub = defineComponent({
  name: 'EditStub',
  setup() {
    return () => h('div', { class: 'edit-stub' }, 'edit');
  },
});

const DisplayStub = defineComponent({
  name: 'DisplayStub',
  setup() {
    return () => h('div', { class: 'display-stub' }, 'display');
  },
});

const dialogSfcs = [FieldTranslationsDialog, FieldCompanyValuesDialog];

function restoreAll() {
  restoreSfc(ChoyButton as any);
  for (const Comp of dialogSfcs) restoreSfc(Comp as any);
  restoreSfc(ChoyTableColumn as any);
}

function installUiStubs() {
  stubSfc(ChoyButton as any, {
    name: 'ChoyButton',
    inheritAttrs: false,
    props: {
      type: { type: String, default: 'button' },
      variant: { type: String, default: 'default' },
      size: { type: String, default: 'default' },
      disabled: { type: Boolean, default: false },
      class: { type: [String, Object, Array], default: undefined },
      as: { type: String, default: 'button' },
    },
    emits: ['click'],
    setup(p: any, { slots, emit, attrs }: any) {
      return () =>
        h(
          'button',
          {
            type: p.type || 'button',
            class: p.class || attrs.class,
            'aria-label': attrs['aria-label'],
            disabled: p.disabled || undefined,
            onClick: (e: Event) => emit('click', e),
          },
          slots.default?.()
        );
    },
  });
}

function installDialogStubs(opts?: {
  onTranslateSaved?: (v: string | null) => void;
  onCompanySaved?: (v: unknown) => void;
}) {
  stubSfc(FieldTranslationsDialog as any, {
    name: 'FieldTranslationsDialog',
    props: ['modelValue', 'store', 'recordId', 'fieldName', 'fieldLabel', 'draftValue', 'maxLength'],
    emits: ['update:modelValue', 'saved'],
    setup(p: any, { emit }: any) {
      return () =>
        h('div', {
          class: 'translations-dialog',
          'data-open': String(!!p.modelValue),
          'data-field': String(p.fieldName || ''),
          onClick: () => {
            opts?.onTranslateSaved?.(null);
            emit('saved', 'translated-value');
            emit('update:modelValue', false);
          },
        });
    },
  });
  stubSfc(FieldCompanyValuesDialog as any, {
    name: 'FieldCompanyValuesDialog',
    props: ['modelValue', 'store', 'recordId', 'fieldName', 'fieldLabel', 'fieldType', 'draftValue', 'maxLength'],
    emits: ['update:modelValue', 'saved'],
    setup(p: any, { emit }: any) {
      return () =>
        h('div', {
          class: 'company-values-dialog',
          'data-open': String(!!p.modelValue),
          'data-type': String(p.fieldType || ''),
          onClick: () => {
            opts?.onCompanySaved?.(null);
            emit('saved', 'company-value');
            emit('update:modelValue', false);
          },
        });
    },
  });
}

function mountBase(
  props: Record<string, unknown>,
  opts?: {
    provide?: Record<string, unknown>;
    slots?: Record<string, (...args: any[]) => any>;
    dialog?: {
      onTranslateSaved?: (v: string | null) => void;
      onCompanySaved?: (v: unknown) => void;
    };
  }
) {
  installUiStubs();
  installDialogStubs(opts?.dialog);
  return mountApp(FieldBase as any, {
    props,
    provide: opts?.provide,
    slots: opts?.slots || {
      edit: () => h(EditStub),
      display: () => h(DisplayStub),
    },
  });
}

describe('FieldBase label and help', () => {
  afterEach(() => {
    restoreAll();
  });

  test('omits props.label and shows meta string fallback', () => {
    const m = mountBase({
      binding: makeBinding({ string: 'Access Token ID' }),
      renderMode: 'form',
    });
    expect(m.q('.choy-field-base__label-text')?.textContent).toContain('Access Token ID');
    m.unmount();
  });

  test('lets explicit label override metadata', () => {
    const m = mountBase({
      binding: makeBinding({ string: 'Access Token ID' }),
      label: 'Custom Name',
      renderMode: 'form',
    });
    expect(m.q('.choy-field-base__label-text')?.textContent).toContain('Custom Name');
    m.unmount();
  });

  test('prefers FieldsGet overlay translated string via store helpers', () => {
    const binding = makeBinding({ string: 'Access Token ID' });
    binding.store = {
      getFieldMeta: (name: string) =>
        name === 'AccessTokenId'
          ? ({ type: 'varchar', typeAnnotation: 'string', id: '1', string: '访问令牌 ID' } as any)
          : undefined,
      getFieldsGetTranslatedString: (name: string) => (name === 'AccessTokenId' ? '访问令牌 ID' : undefined),
    } as any;
    const m = mountBase({ binding, renderMode: 'form' });
    expect(m.q('.choy-field-base__label-text')?.textContent).toContain('访问令牌 ID');
    m.unmount();
  });

  test('renders form label help tip when meta.help is present', () => {
    const m = mountBase({
      binding: makeBinding({ string: 'Code', help: 'Short unique code used in references' }),
      renderMode: 'form',
    });
    const help = m.q('.choy-field-base__help-btn') as HTMLElement | null;
    expect(help).toBeTruthy();
    expect(help!.getAttribute('title')).toBe('Short unique code used in references');
    m.unmount();
  });

  test('omits help tip when help is blank', () => {
    const m = mountBase({
      binding: makeBinding({ string: 'Code', help: '   ' }),
      renderMode: 'form',
    });
    expect(m.q('.choy-field-base__help-btn')).toBeFalsy();
    m.unmount();
  });

  test('renders inline help tip beside the control', () => {
    const m = mountBase({
      binding: makeBinding({ string: 'Code', help: 'Inline help text' }),
      renderMode: 'inline',
    });
    expect(m.q('.choy-field-base__inline-wrap')).toBeTruthy();
    expect(m.q('.choy-field-base__help-btn')?.getAttribute('title')).toBe('Inline help text');
    m.unmount();
  });
});

describe('FieldBase FieldsGet readonly', () => {
  afterEach(() => {
    restoreAll();
  });

  test('honors FieldsGet isReadonly overlay and shows display slot', async () => {
    const binding = makeBinding({ string: 'Token' });
    binding.store = {
      getFieldMeta: () => ({ type: 'varchar', isReadonly: true } as any),
      ensureFieldsGet: async () => ({}),
    } as any;
    const m = mountBase({ binding, renderMode: 'form' });
    await flushPromises();
    expect(m.q('.display-stub')).toBeTruthy();
    expect(m.q('.edit-stub')).toBeFalsy();
    m.unmount();
  });

  test('honors static binding.meta.isReadonly without FieldsGet overlay', () => {
    const m = mountBase({
      binding: makeBinding({ string: 'Token', isReadonly: true }),
      renderMode: 'form',
    });
    expect(m.q('.display-stub')).toBeTruthy();
    expect(m.q('.edit-stub')).toBeFalsy();
    m.unmount();
  });
});

describe('FieldBase translate and company values actions', () => {
  afterEach(() => {
    restoreAll();
  });

  test('shows translate icon and applies saved value to the field binding', async () => {
    const binding = makeBinding({ string: 'Name', translate: true, type: 'char' });
    const m = mountBase({ binding, renderMode: 'form' });
    expect(m.q('.choy-field-base__translate-btn')).toBeTruthy();
    expect(m.q('.translations-dialog')?.getAttribute('data-open')).toBe('false');

    m.click('.choy-field-base__translate-btn');
    await nextTick();
    expect(m.q('.translations-dialog')?.getAttribute('data-open')).toBe('true');

    m.click('.translations-dialog');
    await flushPromises();
    expect(binding.__value.value).toBe('translated-value');
    m.unmount();
  });

  test('hides translate icon when binding.store is missing', () => {
    const m = mountBase({
      binding: makeBinding({ string: 'Name', translate: true }, { store: undefined }),
      renderMode: 'form',
    });
    expect(m.q('.choy-field-base__translate-btn')).toBeFalsy();
    m.unmount();
  });

  test('hides translate icon when record has no Id', () => {
    const m = mountBase({
      binding: makeBinding({ string: 'Name', translate: true }, { recordId: null }),
      renderMode: 'form',
    });
    expect(m.q('.choy-field-base__translate-btn')).toBeFalsy();
    m.unmount();
  });

  test('shows company-values icon and applies saved value', async () => {
    const binding = makeBinding({ string: 'Name', companyDependent: true, type: 'char' });
    const m = mountBase({ binding, renderMode: 'form' });
    expect(m.q('.choy-field-base__company-values-btn')).toBeTruthy();

    m.click('.choy-field-base__company-values-btn');
    await nextTick();
    expect(m.q('.company-values-dialog')?.getAttribute('data-open')).toBe('true');

    m.click('.company-values-dialog');
    await flushPromises();
    expect(binding.__value.value).toBe('company-value');
    m.unmount();
  });

  test('hides company-values icon when meta.companyDependent is missing', () => {
    const m = mountBase({
      binding: makeBinding({ string: 'Name' }),
      renderMode: 'form',
    });
    expect(m.q('.choy-field-base__company-values-btn')).toBeFalsy();
    m.unmount();
  });
});

describe('FieldBase list-editing-row-id gate', () => {
  afterEach(() => {
    restoreAll();
  });

  function installOvColumnStub(row: Record<string, unknown>) {
    stubSfc(ChoyTableColumn as any, {
      name: 'ChoyTableColumn',
      props: ['prop', 'label', 'vColumnProps'],
      setup(_p: any, { slots }: any) {
        return () => h('div', { class: 'ov-column-stub' }, slots.default?.({ row, $index: 0 }));
      },
    });
  }

  function mountTableCell(row: Record<string, unknown>, editingId: string | null, fieldPrefix?: string | null) {
    const editing = ref(editingId);
    installUiStubs();
    installDialogStubs();
    installOvColumnStub(row);
    return mountApp(FieldBase as any, {
      props: {
        binding: makeBinding({ string: 'Name' }, { fieldPrefix: fieldPrefix ?? null }),
        renderMode: 'table',
      },
      provide: { 'list-editing-row-id': editing },
      slots: {
        edit: () => h(EditStub),
        display: () => h(DisplayStub),
      },
    });
  }

  test('shows edit slot only for the row matching list-editing-row-id', () => {
    const m = mountTableCell({ Id: 'row-1' }, 'row-1');
    expect(m.q('.edit-stub')).toBeTruthy();
    expect(m.q('.display-stub')).toBeFalsy();
    m.unmount();
  });

  test('shows display when row id differs under active list-editing-row-id', () => {
    const m = mountTableCell({ Id: 'row-2' }, 'row-1');
    expect(m.q('.display-stub')).toBeTruthy();
    expect(m.q('.edit-stub')).toBeFalsy();
    m.unmount();
  });

  test('allows all rows in edit mode when list-editing-row-id is not injected', () => {
    installUiStubs();
    installDialogStubs();
    installOvColumnStub({ Id: 'row-9' });
    const m = mountApp(FieldBase as any, {
      props: {
        binding: makeBinding({ string: 'Name' }),
        renderMode: 'table',
      },
      slots: {
        edit: () => h(EditStub),
        display: () => h(DisplayStub),
      },
    });
    expect(m.q('.edit-stub')).toBeTruthy();
    m.unmount();
  });

  test('skips list-editing-row-id gate for nested relation tables with fieldPrefix', () => {
    const m = mountTableCell({ Id: 'child-1' }, 'other-row', 'Lines');
    expect(m.q('.edit-stub')).toBeTruthy();
    m.unmount();
  });

  test('evaluates client rules and registers with form validators', async () => {
    installUiStubs();
    installDialogStubs();
    const { FIELD_CLIENT_VALIDATORS_KEY } = await import('@/web/web/composables/fieldClientValidation');
    const validators = new Map<string, () => Promise<string>>();
    const binding = makeBinding({ string: 'Name' });
    binding.__value.value = '';
    const m = mountApp(FieldBase as any, {
      props: {
        binding,
        renderMode: 'form',
        rules: [
          {
            validator: (_r: unknown, v: unknown, cb: (e?: Error) => void) => {
              if (!v) cb(new Error('need value'));
              else cb();
            },
          },
        ],
      },
      provide: { [FIELD_CLIENT_VALIDATORS_KEY as any]: validators },
      slots: {
        edit: () => h(EditStub),
        display: () => h(DisplayStub),
      },
    });
    await flushPromises();
    expect(m.q('.choy-field-base__error')?.textContent).toContain('need value');
    expect(validators.size).toBe(1);
    const validate = [...validators.values()][0];
    expect(await validate()).toBe('need value');
    binding.__value.value = 'ok';
    await flushPromises();
    expect(await validate()).toBe('');
    expect(m.q('.choy-field-base__error')).toBeFalsy();
    m.unmount();
    expect(validators.size).toBe(0);
  });

  test('client validators stay isolated across same-prop instances', async () => {
    installUiStubs();
    installDialogStubs();
    const { FIELD_CLIENT_VALIDATORS_KEY } = await import('@/web/web/composables/fieldClientValidation');
    const validators = new Map<string, () => Promise<string>>();
    const bindingA = makeBinding({ string: 'Name' });
    bindingA.prop = 'Name';
    bindingA.__value.value = '';
    const bindingB = makeBinding({ string: 'Name' });
    bindingB.prop = 'Name';
    bindingB.__value.value = 'keep';
    const rules = [
      {
        validator: (_r: unknown, v: unknown, cb: (e?: Error) => void) => {
          if (!v) cb(new Error('need value'));
          else cb();
        },
      },
    ];
    const mountOne = (binding: any) =>
      mountApp(FieldBase as any, {
        props: { binding, renderMode: 'form', rules },
        provide: { [FIELD_CLIENT_VALIDATORS_KEY as any]: validators },
        slots: {
          edit: () => h(EditStub),
          display: () => h(DisplayStub),
        },
      });
    const a = mountOne(bindingA);
    const b = mountOne(bindingB);
    await flushPromises();
    expect(validators.size).toBe(2);
    a.unmount();
    expect(validators.size).toBe(1);
    const remaining = [...validators.values()][0];
    expect(await remaining()).toBe('');
    b.unmount();
    expect(validators.size).toBe(0);
  });
});

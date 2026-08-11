// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount } from '@vue/test-utils';
import { computed, defineComponent, h, nextTick, ref } from 'vue';
import { describe, expect, it, vi } from 'vitest';

import { createTermReference } from '@/core/service/i18n';
import type { UseField } from '@/web/web/composables/useField';
import OFieldBase from './OFieldBase.vue';

function makeBinding(
  meta?: {
    string?: string;
    stringText?: ReturnType<typeof createTermReference>;
    help?: string;
    helpText?: ReturnType<typeof createTermReference>;
    translate?: boolean;
    companyDependent?: boolean;
  },
  opts?: { recordId?: string | null; isEditMode?: boolean; fieldPrefix?: string | null },
): UseField {
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
    // Dialogs require an Object store prop; a minimal stub avoids Vue prop warnings.
    store: {} as any,
    asView: () => ({ fieldValue: () => value }) as any,
  } as UseField;
}

const EditStub = defineComponent({
  name: 'EditStub',
  setup(_, { slots }) {
    return () => h('div', { class: 'edit-stub' }, slots.default?.({} as any));
  },
});

const fieldBaseStubs = {
  'el-form-item': {
    props: ['label'],
    template:
      '<div class="form-item" :data-label="label"><span class="form-item-label"><slot name="label" /></span><slot /></div>',
  },
  // Inline error icon; Element Plus is not registered in this unit suite.
  'el-icon': { template: '<i class="el-icon-stub"><slot /></i>' },
  'el-tooltip': {
    props: ['content'],
    template: '<div class="tooltip-stub" :data-content="content"><slot /></div>',
  },
  'el-button': {
    template: '<button class="btn-stub" v-bind="$attrs"><slot /></button>',
  },
  OFieldTranslationsDialog: true,
  OFieldCompanyValuesDialog: true,
};

describe('OFieldBase label resolution', () => {
  const stringText = createTermReference('auth', 'Access Token ID', {
    scope: 'auth.model.Session.fields',
  });

  it('omits props.label and shows meta string / stringText fallback', () => {
    const wrapper = mount(OFieldBase, {
      props: {
        binding: makeBinding({ string: 'Access Token ID', stringText }),
        renderMode: 'form',
      },
      slots: {
        edit: () => h(EditStub),
      },
      global: { stubs: fieldBaseStubs },
    });
    expect(wrapper.get('.form-item-label').text()).toContain('Access Token ID');
  });

  it('lets explicit label override metadata', () => {
    const wrapper = mount(OFieldBase, {
      props: {
        binding: makeBinding({ string: 'Access Token ID', stringText }),
        label: 'Custom Name',
        renderMode: 'form',
      },
      slots: {
        edit: () => h(EditStub),
      },
      global: { stubs: fieldBaseStubs },
    });
    expect(wrapper.get('.form-item-label').text()).toContain('Custom Name');
  });

  it('prefers FieldsGet overlay translated string via store helpers', () => {
    const binding = makeBinding({ string: 'Access Token ID', stringText });
    binding.store = {
      getFieldMeta: (name: string) =>
        name === 'AccessTokenId'
          ? ({ type: 'varchar', typeAnnotation: 'string', id: '1', string: '访问令牌 ID' } as any)
          : undefined,
      getFieldsGetTranslatedString: (name: string) => (name === 'AccessTokenId' ? '访问令牌 ID' : undefined),
    } as any;

    const wrapper = mount(OFieldBase, {
      props: {
        binding,
        renderMode: 'form',
      },
      slots: {
        edit: () => h(EditStub),
      },
      global: { stubs: fieldBaseStubs },
    });
    expect(wrapper.get('.form-item-label').text()).toContain('访问令牌 ID');
  });
});

describe('OFieldBase field help tip', () => {
  it('renders form label help tip when meta.help is present', () => {
    const wrapper = mount(OFieldBase, {
      props: {
        binding: makeBinding({
          string: 'Code',
          help: 'Short unique code used in references',
        }),
        renderMode: 'form',
      },
      slots: {
        edit: () => h(EditStub),
      },
      global: { stubs: fieldBaseStubs },
    });
    const tip = wrapper.get('.o-field-base__help-icon').element.closest('.tooltip-stub') as HTMLElement | null;
    expect(tip?.getAttribute('data-content')).toBe('Short unique code used in references');
  });

  it('omits help tip when help is blank', () => {
    const wrapper = mount(OFieldBase, {
      props: {
        binding: makeBinding({ string: 'Code', help: '   ' }),
        renderMode: 'form',
      },
      slots: {
        edit: () => h(EditStub),
      },
      global: { stubs: fieldBaseStubs },
    });
    expect(wrapper.find('.o-field-base__help-icon').exists()).toBe(false);
  });

  it('does not render help tip in table mode', () => {
    const wrapper = mount(OFieldBase, {
      props: {
        binding: makeBinding({
          string: 'Code',
          help: 'Short unique code used in references',
        }),
        renderMode: 'table',
      },
      slots: {
        display: () => h('span', 'x'),
      },
      global: {
        stubs: {
          ...fieldBaseStubs,
          OVColumn: {
            template: '<div class="ov-column"><slot :row="{ Id: 1 }" :$index="0" /></div>',
          },
        },
      },
    });
    expect(wrapper.find('.o-field-base__help-icon').exists()).toBe(false);
  });

  it('prefers FieldsGet translated help via store helpers', () => {
    const binding = makeBinding({
      string: 'Code',
      help: 'Short unique code used in references',
      helpText: createTermReference('base', 'Short unique code used in references', {
        scope: 'base.model.Company.fields',
      }),
    });
    binding.store = {
      getFieldMeta: (name: string) =>
        name === 'AccessTokenId'
          ? ({
              type: 'varchar',
              typeAnnotation: 'string',
              id: '1',
              string: 'Code',
              help: '用于引用的短唯一编码',
            } as any)
          : undefined,
      getFieldsGetTranslatedHelp: (name: string) =>
        name === 'AccessTokenId' ? '用于引用的短唯一编码' : undefined,
    } as any;

    const wrapper = mount(OFieldBase, {
      props: {
        binding,
        renderMode: 'form',
      },
      slots: {
        edit: () => h(EditStub),
      },
      global: { stubs: fieldBaseStubs },
    });
    const tip = wrapper.get('.o-field-base__help-icon').element.closest('.tooltip-stub') as HTMLElement | null;
    expect(tip?.getAttribute('data-content')).toBe('用于引用的短唯一编码');
    expect(wrapper.get('.o-field-base__help-btn').attributes('aria-label')).toBe(
      'Code: 用于引用的短唯一编码'
    );
  });

  it('renders inline help tip beside the control', () => {
    const wrapper = mount(OFieldBase, {
      props: {
        binding: makeBinding({
          string: 'Code',
          help: 'Short unique code used in references',
        }),
        renderMode: 'inline',
      },
      slots: {
        edit: () => h(EditStub),
      },
      global: { stubs: fieldBaseStubs },
    });
    expect(wrapper.find('.o-field-base__inline-wrap--has-help').exists()).toBe(true);
    const tip = wrapper.get('.o-field-base__help-icon').element.closest('.tooltip-stub') as HTMLElement | null;
    expect(tip?.getAttribute('data-content')).toBe('Short unique code used in references');
    expect(wrapper.get('.o-field-base__help-btn').attributes('aria-label')).toContain(
      'Short unique code used in references'
    );
  });

  it('renders inline help tip in readonly display slot', () => {
    const wrapper = mount(OFieldBase, {
      props: {
        binding: makeBinding(
          {
            string: 'Code',
            help: 'Units per base currency',
          },
          { isEditMode: false }
        ),
        renderMode: 'inline',
      },
      slots: {
        display: () => h('span', { class: 'display-stub' }, 'x'),
      },
      global: { stubs: fieldBaseStubs },
    });
    expect(wrapper.find('.display-stub').exists()).toBe(true);
    expect(wrapper.find('.o-field-base__help-btn').exists()).toBe(true);
    expect(wrapper.get('.o-field-base__help-btn').attributes('aria-label')).toBe(
      'Code: Units per base currency'
    );
  });

  it('uses help-only accessible label when field label is blank', () => {
    const binding = makeBinding({ help: 'Standalone help text' });
    binding.prop = '';
    binding.meta = { help: 'Standalone help text' } as any;
    const wrapper = mount(OFieldBase, {
      props: {
        binding,
        renderMode: 'form',
      },
      slots: {
        edit: () => h(EditStub),
      },
      global: { stubs: fieldBaseStubs },
    });
    expect(wrapper.get('.o-field-base__help-btn').attributes('aria-label')).toBe('Standalone help text');
  });

  it('renders inline wrap without help tip when help is absent', () => {
    const wrapper = mount(OFieldBase, {
      props: {
        binding: makeBinding({ string: 'Code' }),
        renderMode: 'inline',
      },
      slots: {
        edit: () => h(EditStub),
      },
      global: { stubs: fieldBaseStubs },
    });
    expect(wrapper.find('.o-field-base__inline-wrap').exists()).toBe(true);
    expect(wrapper.find('.o-field-base__inline-wrap--has-help').exists()).toBe(false);
    expect(wrapper.find('.o-field-base__help-btn').exists()).toBe(false);
  });
});

describe('OFieldBase FieldsGet readonly', () => {
  it('honors FieldsGet isReadonly overlay and shows display slot (T5.3)', async () => {
    const ensureFieldsGet = vi.fn(async () => ({}));
    const binding = makeBinding({ string: 'Code' });
    binding.meta = { type: 'varchar', typeAnnotation: 'string', id: '1', string: 'Code' } as any;
    binding.store = {
      getFieldMeta: (name: string) =>
        name === 'AccessTokenId'
          ? ({ type: 'varchar', typeAnnotation: 'string', id: '1', string: 'Code', isReadonly: true } as any)
          : undefined,
      getFieldsGetTranslatedString: () => undefined,
      ensureFieldsGet,
    } as any;

    const wrapper = mount(OFieldBase, {
      props: {
        binding,
        renderMode: 'form',
      },
      slots: {
        edit: () => h('div', { class: 'edit-slot' }, 'edit'),
        display: () => h('div', { class: 'display-slot' }, 'display'),
      },
      global: {
        stubs: {
          ...fieldBaseStubs,
          'el-form-item': {
            props: ['label'],
            template: '<div class="form-item"><slot /></div>',
          },
        },
      },
    });

    expect(ensureFieldsGet).toHaveBeenCalled();
    expect(wrapper.find('.edit-slot').exists()).toBe(false);
    expect(wrapper.find('.display-slot').exists()).toBe(true);
  });

  it('honors static binding.meta.isReadonly without FieldsGet overlay (PR-P2-F2)', () => {
    const binding = makeBinding({ string: 'External Id' });
    binding.meta = {
      type: 'varchar',
      typeAnnotation: 'string',
      id: '1',
      string: 'External Id',
      isReadonly: true,
    } as any;

    const wrapper = mount(OFieldBase, {
      props: {
        binding,
        renderMode: 'form',
      },
      slots: {
        edit: () => h('div', { class: 'edit-slot' }, 'edit'),
        display: () => h('div', { class: 'display-slot' }, 'display'),
      },
      global: {
        stubs: {
          ...fieldBaseStubs,
          'el-form-item': {
            props: ['label'],
            template: '<div class="form-item"><slot /></div>',
          },
        },
      },
    });

    expect(wrapper.find('.edit-slot').exists()).toBe(false);
    expect(wrapper.find('.display-slot').exists()).toBe(true);
  });
});

describe('OFieldBase translate action', () => {
  it('shows translate icon in form edit when meta.translate and record Id exist', () => {
    const wrapper = mount(OFieldBase, {
      props: {
        binding: makeBinding({ string: 'Name', translate: true }),
        renderMode: 'form',
      },
      slots: {
        edit: () => h(EditStub),
      },
      global: { stubs: fieldBaseStubs },
    });
    const btn = wrapper.find('.o-field-base__translate-btn');
    expect(btn.exists()).toBe(true);
    expect(btn.attributes('aria-label')).toContain('Translate');
  });

  it('hides translate icon when binding.store is missing', () => {
    const binding = makeBinding({ string: 'Name', translate: true });
    (binding as any).store = undefined;
    const wrapper = mount(OFieldBase, {
      props: {
        binding,
        renderMode: 'form',
      },
      slots: {
        edit: () => h(EditStub),
      },
      global: { stubs: fieldBaseStubs },
    });
    expect(wrapper.find('.o-field-base__translate-btn').exists()).toBe(false);
  });

  it('hides translate icon when meta.translate is missing', () => {
    const wrapper = mount(OFieldBase, {
      props: {
        binding: makeBinding({ string: 'Name' }),
        renderMode: 'form',
      },
      slots: {
        edit: () => h(EditStub),
      },
      global: { stubs: fieldBaseStubs },
    });
    expect(wrapper.find('.o-field-base__translate-btn').exists()).toBe(false);
  });

  it('hides translate icon when record has no Id', () => {
    const wrapper = mount(OFieldBase, {
      props: {
        binding: makeBinding({ string: 'Name', translate: true }, { recordId: null }),
        renderMode: 'form',
      },
      slots: {
        edit: () => h(EditStub),
      },
      global: { stubs: fieldBaseStubs },
    });
    expect(wrapper.find('.o-field-base__translate-btn').exists()).toBe(false);
  });

  it('hides translate icon when not in edit mode', () => {
    const wrapper = mount(OFieldBase, {
      props: {
        binding: makeBinding({ string: 'Name', translate: true }, { isEditMode: false }),
        renderMode: 'form',
      },
      slots: {
        edit: () => h(EditStub),
        display: () => h('div', { class: 'display-slot' }, 'display'),
      },
      global: { stubs: fieldBaseStubs },
    });
    expect(wrapper.find('.o-field-base__translate-btn').exists()).toBe(false);
  });

  it('opens translation dialog and applies saved value to the field binding', async () => {
    const binding = makeBinding({ string: 'Name', translate: true });
    binding.prop = 'PartnerId.Name';
    binding.meta = { string: 'Name', translate: true, size: 80 } as any;
    const value = binding.fieldRef() as { value: string };
    value.value = 'old';

    const wrapper = mount(OFieldBase, {
      props: {
        binding,
        renderMode: 'form',
      },
      slots: {
        edit: () => h(EditStub),
      },
      global: {
        stubs: {
          ...fieldBaseStubs,
          OFieldTranslationsDialog: {
            name: 'OFieldTranslationsDialog',
            props: ['modelValue', 'fieldName', 'maxLength', 'draftValue'],
            emits: ['update:modelValue', 'saved'],
            template:
              '<div class="dialog-stub" :data-open="modelValue" :data-field="fieldName" :data-max="maxLength" :data-draft="draftValue"><button class="emit-saved" @click="$emit(\'saved\', \'新值\')" /></div>',
          },
        },
      },
    });

    expect(wrapper.find('.dialog-stub').attributes('data-open')).toBe('false');
    await wrapper.find('.o-field-base__translate-btn').trigger('click');
    await nextTick();
    const dialog = wrapper.find('.dialog-stub');
    expect(dialog.attributes('data-open')).toBe('true');
    expect(dialog.attributes('data-field')).toBe('Name');
    expect(dialog.attributes('data-max')).toBe('80');
    expect(dialog.attributes('data-draft')).toBe('old');
    await dialog.find('.emit-saved').trigger('click');
    expect(value.value).toBe('新值');
  });

  it('shows translate in preserveModeSlot edit wrap and applies null saved value', async () => {
    const binding = makeBinding({ string: 'Name', translate: true });
    binding.meta = { string: 'Name', translate: true, size: 0 } as any;
    const value = binding.fieldRef() as { value: string | null };
    value.value = 'draft';

    const wrapper = mount(OFieldBase, {
      props: {
        binding,
        renderMode: 'form',
        preserveModeSlot: true,
      },
      slots: {
        edit: () => h(EditStub),
        display: () => h('div', { class: 'display-slot' }, 'display'),
      },
      global: {
        stubs: {
          ...fieldBaseStubs,
          OFieldTranslationsDialog: {
            name: 'OFieldTranslationsDialog',
            props: ['modelValue', 'maxLength', 'draftValue'],
            emits: ['update:modelValue', 'saved'],
            template:
              '<div class="dialog-stub" :data-open="modelValue" :data-max="String(maxLength)" :data-draft="draftValue"><button class="emit-saved" @click="$emit(\'saved\', null)" /></div>',
          },
        },
      },
    });

    expect(wrapper.find('.o-field-base__translate-btn').exists()).toBe(true);
    expect(wrapper.find('.dialog-stub').attributes('data-max')).toBe('undefined');
    await wrapper.find('.o-field-base__translate-btn').trigger('click');
    await nextTick();
    await wrapper.find('.emit-saved').trigger('click');
    expect(value.value).toBeNull();
  });

  it('honors FieldsGet translate overlay and falls back to Translate field aria', async () => {
    const binding = makeBinding({ string: '' });
    binding.prop = 'Name';
    binding.meta = { string: '', translate: false } as any;
    binding.store = {
      getFieldMeta: (name: string) =>
        name === 'Name'
          ? ({ type: 'varchar', typeAnnotation: 'string', id: '1', string: '', translate: true, size: 12.5 } as any)
          : undefined,
      getFieldsGetTranslatedString: () => undefined,
    } as any;

    const wrapper = mount(OFieldBase, {
      props: {
        binding,
        renderMode: 'form',
      },
      slots: {
        edit: () => h(EditStub),
      },
      global: {
        stubs: {
          ...fieldBaseStubs,
          OFieldTranslationsDialog: true,
        },
      },
    });

    const btn = wrapper.find('.o-field-base__translate-btn');
    expect(btn.exists()).toBe(true);
    expect(btn.attributes('aria-label')).toMatch(/Translate/);
  });

  it('swallows translation draft write failures when fieldRef assignment throws', async () => {
    const binding = makeBinding({ string: 'Name', translate: true });
    binding.meta = { string: 'Name', translate: true } as any;
    const boom = {
      get value() {
        return 'old';
      },
      set value(_v: unknown) {
        throw new Error('assign fail');
      },
    };
    binding.fieldRef = () => boom as any;

    const wrapper = mount(OFieldBase, {
      props: {
        binding,
        renderMode: 'form',
      },
      slots: {
        edit: () => h(EditStub),
      },
      global: {
        stubs: {
          ...fieldBaseStubs,
          OFieldTranslationsDialog: {
            name: 'OFieldTranslationsDialog',
            props: ['modelValue'],
            emits: ['saved'],
            template:
              '<div class="dialog-stub" :data-open="modelValue"><button class="emit-saved" @click="$emit(\'saved\', \'x\')" /></div>',
          },
        },
      },
    });

    await wrapper.find('.o-field-base__translate-btn').trigger('click');
    await nextTick();
    await expect(wrapper.find('.emit-saved').trigger('click')).resolves.toBeUndefined();
  });

  it('hides translate action outside form mode', () => {
    const wrapper = mount(OFieldBase, {
      props: {
        binding: makeBinding({ string: 'Name', translate: true }),
        renderMode: 'table',
      },
      slots: {
        edit: () => h(EditStub),
        display: () => h('div', { class: 'display-slot' }, 'display'),
      },
      global: {
        stubs: {
          ...fieldBaseStubs,
          OVColumn: {
            template: '<div class="ov-column"><slot :row="{ Id: 1 }" :$index="0" /></div>',
          },
        },
      },
    });
    expect(wrapper.find('.o-field-base__translate-btn').exists()).toBe(false);
  });
});

describe('OFieldBase list-editing-row-id gate', () => {
  const tableRowsStub = defineComponent({
    name: 'OVColumnRowsStub',
    setup(_, { slots }) {
      const rows = [
        { kind: 'record', payload: { Id: '1', Name: 'A' } },
        { kind: 'record', payload: { Id: '2', Name: 'B' } },
      ];
      return () =>
        h(
          'div',
          { class: 'ov-column' },
          rows.map((row, i) => h('div', { class: `row-${i}` }, slots.default?.({ row, $index: i })))
        );
    },
  });

  it('shows edit slot only for the row matching list-editing-row-id', () => {
    const editingId = ref<string | null>('1');
    const wrapper = mount(OFieldBase, {
      props: {
        binding: makeBinding({ string: 'Name' }, { isEditMode: true }),
        renderMode: 'table',
      },
      slots: {
        edit: ({ record }) => h('div', { class: 'edit-slot' }, `edit-${record().value.Id}`),
        display: () => h('div', { class: 'display-slot' }, 'display'),
      },
      global: {
        stubs: { ...fieldBaseStubs, OVColumn: tableRowsStub },
        provide: { 'list-editing-row-id': editingId },
      },
    });
    expect(wrapper.findAll('.edit-slot')).toHaveLength(1);
    expect(wrapper.find('.edit-slot').text()).toBe('edit-1');
    expect(wrapper.findAll('.display-slot')).toHaveLength(1);
  });

  it('shows display when row id is null under active list-editing-row-id', () => {
    const editingId = ref<string | null>('1');
    const wrapper = mount(OFieldBase, {
      props: {
        binding: makeBinding({ string: 'Name' }, { isEditMode: true }),
        renderMode: 'table',
      },
      slots: {
        edit: () => h('div', { class: 'edit-slot' }, 'edit'),
        display: () => h('div', { class: 'display-slot' }, 'display'),
      },
      global: {
        stubs: {
          ...fieldBaseStubs,
          OVColumn: defineComponent({
            setup(_, { slots }) {
              return () =>
                h('div', slots.default?.({ row: { kind: 'record', payload: { Name: 'no-id' } }, $index: 0 }));
            },
          }),
        },
        provide: { 'list-editing-row-id': editingId },
      },
    });
    expect(wrapper.find('.edit-slot').exists()).toBe(false);
    expect(wrapper.find('.display-slot').exists()).toBe(true);
  });

  it('allows all rows in edit mode when list-editing-row-id is not injected', () => {
    const wrapper = mount(OFieldBase, {
      props: {
        binding: makeBinding({ string: 'Name' }, { isEditMode: true }),
        renderMode: 'table',
      },
      slots: {
        edit: () => h('div', { class: 'edit-slot' }, 'edit'),
        display: () => h('div', { class: 'display-slot' }, 'display'),
      },
      global: {
        stubs: { ...fieldBaseStubs, OVColumn: tableRowsStub },
      },
    });
    expect(wrapper.findAll('.edit-slot')).toHaveLength(2);
  });

  it('keeps display when list-editing-row-id is set but env is not edit mode', () => {
    const editingId = ref<string | null>('1');
    const wrapper = mount(OFieldBase, {
      props: {
        binding: makeBinding({ string: 'Name' }, { isEditMode: false }),
        renderMode: 'table',
      },
      slots: {
        edit: () => h('div', { class: 'edit-slot' }, 'edit'),
        display: () => h('div', { class: 'display-slot' }, 'display'),
      },
      global: {
        stubs: { ...fieldBaseStubs, OVColumn: tableRowsStub },
        provide: { 'list-editing-row-id': editingId },
      },
    });
    expect(wrapper.findAll('.edit-slot')).toHaveLength(0);
    expect(wrapper.findAll('.display-slot').length).toBeGreaterThan(0);
  });

  it('skips list-editing-row-id gate for nested relation tables with fieldPrefix', () => {
    const editingId = ref<string | null>('1');
    const lineRowsStub = defineComponent({
      name: 'OVColumnLineRowsStub',
      setup(_, { slots }) {
        const rows = [
          { kind: 'record', payload: { Id: '10', Name: 'L1' } },
          { kind: 'record', payload: { Id: '11', Name: 'L2' } },
        ];
        return () =>
          h(
            'div',
            { class: 'ov-column' },
            rows.map((row, i) => h('div', { class: `row-${i}` }, slots.default?.({ row, $index: i })))
          );
      },
    });
    const wrapper = mount(OFieldBase, {
      props: {
        binding: makeBinding({ string: 'Name' }, { isEditMode: true, fieldPrefix: 'Lines' }),
        renderMode: 'table',
      },
      slots: {
        edit: ({ record }) => h('div', { class: 'edit-slot' }, `edit-${record().value.Id}`),
        display: () => h('div', { class: 'display-slot' }, 'display'),
      },
      global: {
        stubs: { ...fieldBaseStubs, OVColumn: lineRowsStub },
        provide: { 'list-editing-row-id': editingId },
      },
    });
    expect(wrapper.findAll('.edit-slot')).toHaveLength(2);
    expect(wrapper.findAll('.display-slot')).toHaveLength(0);
  });
});

describe('OFieldBase company values action', () => {
  it('shows company-values icon in form edit when meta.companyDependent and record Id exist', () => {
    const wrapper = mount(OFieldBase, {
      props: {
        binding: makeBinding({ string: 'Cost', companyDependent: true }),
        renderMode: 'form',
      },
      slots: {
        edit: () => h(EditStub),
      },
      global: { stubs: fieldBaseStubs },
    });
    const btn = wrapper.find('.o-field-base__company-values-btn');
    expect(btn.exists()).toBe(true);
    expect(btn.attributes('aria-label')).toContain('Company values');
  });

  it('hides company-values icon when binding.store is missing', () => {
    const binding = makeBinding({ string: 'Cost', companyDependent: true });
    (binding as any).store = undefined;
    const wrapper = mount(OFieldBase, {
      props: {
        binding,
        renderMode: 'form',
      },
      slots: {
        edit: () => h(EditStub),
      },
      global: { stubs: fieldBaseStubs },
    });
    expect(wrapper.find('.o-field-base__company-values-btn').exists()).toBe(false);
  });

  it('hides company-values icon when meta.companyDependent is missing', () => {
    const wrapper = mount(OFieldBase, {
      props: {
        binding: makeBinding({ string: 'Cost' }),
        renderMode: 'form',
      },
      slots: {
        edit: () => h(EditStub),
      },
      global: { stubs: fieldBaseStubs },
    });
    expect(wrapper.find('.o-field-base__company-values-btn').exists()).toBe(false);
  });

  it('hides company-values icon when record has no Id', () => {
    const wrapper = mount(OFieldBase, {
      props: {
        binding: makeBinding({ string: 'Cost', companyDependent: true }, { recordId: null }),
        renderMode: 'form',
      },
      slots: {
        edit: () => h(EditStub),
      },
      global: { stubs: fieldBaseStubs },
    });
    expect(wrapper.find('.o-field-base__company-values-btn').exists()).toBe(false);
  });

  it('hides company-values icon when not in edit mode', () => {
    const wrapper = mount(OFieldBase, {
      props: {
        binding: makeBinding({ string: 'Cost', companyDependent: true }, { isEditMode: false }),
        renderMode: 'form',
      },
      slots: {
        edit: () => h(EditStub),
        display: () => h('div', { class: 'display-slot' }, 'display'),
      },
      global: { stubs: fieldBaseStubs },
    });
    expect(wrapper.find('.o-field-base__company-values-btn').exists()).toBe(false);
  });

  it('opens company values dialog and applies saved value to the field binding', async () => {
    const binding = makeBinding({ string: 'Cost', companyDependent: true });
    binding.prop = 'ProductId.Cost';
    binding.meta = { string: 'Cost', type: 'number', companyDependent: true, size: 80 } as any;
    const value = binding.fieldRef() as { value: string };
    value.value = 'old';

    const wrapper = mount(OFieldBase, {
      props: {
        binding,
        renderMode: 'form',
      },
      slots: {
        edit: () => h(EditStub),
      },
      global: {
        stubs: {
          ...fieldBaseStubs,
          OFieldCompanyValuesDialog: {
            name: 'OFieldCompanyValuesDialog',
            props: ['modelValue', 'fieldName', 'maxLength', 'draftValue', 'fieldType'],
            emits: ['update:modelValue', 'saved'],
            template:
              '<div class="company-dialog-stub" :data-open="modelValue" :data-field="fieldName" :data-max="maxLength" :data-draft="draftValue" :data-type="fieldType"><button class="emit-close" @click="$emit(\'update:modelValue\', false)" /><button class="emit-saved" @click="$emit(\'saved\', \'11.5\')" /></div>',
          },
        },
      },
    });

    expect(wrapper.find('.company-dialog-stub').attributes('data-open')).toBe('false');
    await wrapper.find('.o-field-base__company-values-btn').trigger('click');
    await nextTick();
    const dialog = wrapper.find('.company-dialog-stub');
    expect(dialog.attributes('data-open')).toBe('true');
    expect(dialog.attributes('data-field')).toBe('Cost');
    expect(dialog.attributes('data-max')).toBe('80');
    expect(dialog.attributes('data-draft')).toBe('old');
    expect(dialog.attributes('data-type')).toBe('number');
    await dialog.find('.emit-close').trigger('click');
    await nextTick();
    expect(wrapper.find('.company-dialog-stub').attributes('data-open')).toBe('false');
    await wrapper.find('.o-field-base__company-values-btn').trigger('click');
    await nextTick();
    await dialog.find('.emit-saved').trigger('click');
    expect(value.value).toBe('11.5');
  });

  it('shows company values in preserveModeSlot edit wrap and applies null saved value', async () => {
    const binding = makeBinding({ string: 'Cost', companyDependent: true });
    binding.meta = { string: 'Cost', companyDependent: true, size: 0 } as any;
    const value = binding.fieldRef() as { value: string | null };
    value.value = 'draft';

    const wrapper = mount(OFieldBase, {
      props: {
        binding,
        renderMode: 'form',
        preserveModeSlot: true,
      },
      slots: {
        edit: () => h(EditStub),
        display: () => h('div', { class: 'display-slot' }, 'display'),
      },
      global: {
        stubs: {
          ...fieldBaseStubs,
          OFieldCompanyValuesDialog: {
            name: 'OFieldCompanyValuesDialog',
            props: ['modelValue', 'maxLength', 'draftValue'],
            emits: ['update:modelValue', 'saved'],
            template:
              '<div class="company-dialog-stub" :data-open="modelValue" :data-max="String(maxLength)" :data-draft="draftValue"><button class="emit-saved" @click="$emit(\'saved\', null)" /></div>',
          },
        },
      },
    });

    expect(wrapper.find('.o-field-base__company-values-btn').exists()).toBe(true);
    expect(wrapper.find('.company-dialog-stub').attributes('data-max')).toBe('undefined');
    await wrapper.find('.o-field-base__company-values-btn').trigger('click');
    await nextTick();
    await wrapper.find('.emit-saved').trigger('click');
    expect(value.value).toBeNull();
  });

  it('honors FieldsGet companyDependent overlay and falls back to Company values aria', async () => {
    const binding = makeBinding({ string: '' });
    binding.prop = 'Cost';
    binding.meta = { string: '', companyDependent: false } as any;
    binding.store = {
      getFieldMeta: (name: string) =>
        name === 'Cost'
          ? ({ type: 'float', typeAnnotation: 'number', id: '1', string: '', companyDependent: true, size: 12.5 } as any)
          : undefined,
      getFieldsGetTranslatedString: () => undefined,
    } as any;

    const wrapper = mount(OFieldBase, {
      props: {
        binding,
        renderMode: 'form',
      },
      slots: {
        edit: () => h(EditStub),
      },
      global: {
        stubs: {
          ...fieldBaseStubs,
          OFieldCompanyValuesDialog: true,
        },
      },
    });

    const btn = wrapper.find('.o-field-base__company-values-btn');
    expect(btn.exists()).toBe(true);
    expect(btn.attributes('aria-label')).toMatch(/Company values/);
  });

  it('uses bare Company values aria when label and leaf are blank', async () => {
    const binding = makeBinding({ string: '' });
    binding.prop = '';
    binding.meta = { string: '', companyDependent: true } as any;

    const wrapper = mount(OFieldBase, {
      props: { binding, renderMode: 'form' },
      slots: { edit: () => h(EditStub) },
      global: { stubs: { ...fieldBaseStubs, OFieldCompanyValuesDialog: true } },
    });

    expect(wrapper.find('.o-field-base__company-values-btn').attributes('aria-label')).toBe('Company values');
  });

  it('swallows company-values draft write failures when fieldRef assignment throws', async () => {
    const binding = makeBinding({ string: 'Cost', companyDependent: true });
    binding.meta = { string: 'Cost', companyDependent: true } as any;
    const boom = {
      get value() {
        return 'old';
      },
      set value(_v: unknown) {
        throw new Error('assign fail');
      },
    };
    binding.fieldRef = () => boom as any;

    const wrapper = mount(OFieldBase, {
      props: {
        binding,
        renderMode: 'form',
      },
      slots: {
        edit: () => h(EditStub),
      },
      global: {
        stubs: {
          ...fieldBaseStubs,
          OFieldCompanyValuesDialog: {
            name: 'OFieldCompanyValuesDialog',
            props: ['modelValue'],
            emits: ['saved'],
            template:
              '<div class="company-dialog-stub" :data-open="modelValue"><button class="emit-saved" @click="$emit(\'saved\', \'x\')" /></div>',
          },
        },
      },
    });

    await wrapper.find('.o-field-base__company-values-btn').trigger('click');
    await nextTick();
    await expect(wrapper.find('.emit-saved').trigger('click')).resolves.toBeUndefined();
  });

  it('hides company-values action outside form mode', () => {
    const wrapper = mount(OFieldBase, {
      props: {
        binding: makeBinding({ string: 'Cost', companyDependent: true }),
        renderMode: 'table',
      },
      slots: {
        edit: () => h(EditStub),
        display: () => h('div', { class: 'display-slot' }, 'display'),
      },
      global: {
        stubs: {
          ...fieldBaseStubs,
          OVColumn: {
            template: '<div class="ov-column"><slot :row="{ Id: 1 }" :$index="0" /></div>',
          },
        },
      },
    });
    expect(wrapper.find('.o-field-base__company-values-btn').exists()).toBe(false);
    // Force-evaluate the computed (form branch is not rendered in table mode).
    expect((wrapper.vm as any).$.setupState.showCompanyValuesAction).toBe(false);
  });

  it('panelRecordId catch hides company-values action', async () => {
    // recordRef() is also read at setup for recordFormRef (outside the panelRecordId
    // try/catch), so the first call must succeed; later reads exercise the catch.
    const boomRecord = makeBinding({ string: 'Cost', companyDependent: true });
    boomRecord.meta = { string: 'Cost', companyDependent: true } as any;
    let recordRefCalls = 0;
    const okRecord = computed(() => ({ Id: '1' }));
    boomRecord.recordRef = () => {
      recordRefCalls += 1;
      if (recordRefCalls === 1) return okRecord as any;
      return {
        get value() {
          throw new Error('record boom');
        },
      } as any;
    };

    const hidden = mount(OFieldBase, {
      props: { binding: boomRecord, renderMode: 'form' },
      slots: { edit: () => h(EditStub) },
      global: { stubs: { ...fieldBaseStubs, OFieldCompanyValuesDialog: true } },
    });
    expect(recordRefCalls).toBeGreaterThanOrEqual(2);
    expect(hidden.find('.o-field-base__company-values-btn').exists()).toBe(false);
  });

  it('blank field type yields undefined companyValues fieldType', async () => {
    const binding = makeBinding({ string: 'Cost', companyDependent: true });
    binding.meta = { string: 'Cost', companyDependent: true, type: '   ' } as any;

    const wrapper = mount(OFieldBase, {
      props: { binding, renderMode: 'form' },
      slots: { edit: () => h(EditStub) },
      global: {
        stubs: {
          ...fieldBaseStubs,
          OFieldCompanyValuesDialog: {
            name: 'OFieldCompanyValuesDialog',
            props: ['modelValue', 'fieldType'],
            template: '<div class="company-dialog-stub" :data-open="modelValue" :data-type="String(fieldType)" />',
          },
        },
      },
    });
    await wrapper.find('.o-field-base__company-values-btn').trigger('click');
    await nextTick();
    expect(wrapper.find('.company-dialog-stub').attributes('data-type')).toBe('undefined');
    wrapper.unmount();

    // meta present but type missing → undefined fieldType
    const noType = makeBinding({ string: 'Cost', companyDependent: true });
    noType.meta = { string: 'Cost', companyDependent: true } as any;
    const w2 = mount(OFieldBase, {
      props: { binding: noType, renderMode: 'form' },
      slots: { edit: () => h(EditStub) },
      global: {
        stubs: {
          ...fieldBaseStubs,
          OFieldCompanyValuesDialog: {
            name: 'OFieldCompanyValuesDialog',
            props: ['modelValue', 'fieldType'],
            template: '<div class="company-dialog-stub" :data-type="String(fieldType)" />',
          },
        },
      },
    });
    await w2.find('.o-field-base__company-values-btn').trigger('click');
    await nextTick();
    expect(w2.find('.company-dialog-stub').attributes('data-type')).toBe('undefined');
  });

  it('falsy fieldRef skips assignment on company-values save', async () => {
    const nullValue = ref(null as string | null);
    const binding = makeBinding({ string: 'Cost', companyDependent: true });
    binding.meta = { string: 'Cost', companyDependent: true } as any;
    binding.fieldRef = () => nullValue as any;

    const wrapper = mount(OFieldBase, {
      props: { binding, renderMode: 'form' },
      slots: { edit: () => h(EditStub) },
      global: {
        stubs: {
          ...fieldBaseStubs,
          OFieldCompanyValuesDialog: {
            name: 'OFieldCompanyValuesDialog',
            props: ['modelValue', 'draftValue'],
            emits: ['saved'],
            template:
              '<div class="company-dialog-stub" :data-open="modelValue" :data-draft="draftValue"><button class="emit-saved" @click="$emit(\'saved\', \'11\')" /></div>',
          },
        },
      },
    });
    await wrapper.find('.o-field-base__company-values-btn').trigger('click');
    await nextTick();
    // panelDraftValue: v == null → ''
    expect(wrapper.find('.company-dialog-stub').attributes('data-draft')).toBe('');
    expect(wrapper.find('.el-icon-stub').exists()).toBe(true);

    // onCompanyValuesSaved: falsy fieldRef skips assignment — capture stays untouched
    binding.fieldRef = () => null as any;
    nullValue.value = null;
    await wrapper.find('.emit-saved').trigger('click');
    expect(nullValue.value).toBeNull();
  });
});

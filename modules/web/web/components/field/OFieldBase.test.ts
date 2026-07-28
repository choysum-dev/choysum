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
  meta?: { string?: string; stringText?: ReturnType<typeof createTermReference>; translate?: boolean; companyDependent?: boolean },
  opts?: { recordId?: string | null; isEditMode?: boolean },
): UseField {
  const value = ref('x');
  const recordId = opts && 'recordId' in opts ? opts.recordId : '1';
  const record = ref(recordId ? { Id: recordId } : {});
  return {
    env: {
      isForm: true,
      isEditMode: opts?.isEditMode ?? true,
      viewMode: opts?.isEditMode === false ? 'readonly' : 'edit',
      fieldPrefix: null,
    },
    prop: 'AccessTokenId',
    meta: meta as any,
    fieldRef: () => value as any,
    fieldRefOf: () => value as any,
    recordRef: () => computed(() => record.value) as any,
    registerFields: () => {},
    store: undefined,
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
    template: '<div class="form-item" :data-label="label"><slot /></div>',
  },
  // Inline error icon; Element Plus is not registered in this unit suite.
  'el-icon': { template: '<i class="el-icon-stub"><slot /></i>' },
  'el-tooltip': { template: '<div class="tooltip-stub"><slot /></div>' },
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
    expect(wrapper.get('.form-item').attributes('data-label')).toBe('Access Token ID');
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
    expect(wrapper.get('.form-item').attributes('data-label')).toBe('Custom Name');
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
    expect(wrapper.get('.form-item').attributes('data-label')).toBe('访问令牌 ID');
  });

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

    // onCompanyValuesSaved: falsy fieldRef skips assignment
    binding.fieldRef = () => null as any;
    await expect(wrapper.find('.emit-saved').trigger('click')).resolves.toBeUndefined();
  });
});

// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount } from '@vue/test-utils';
import { computed, defineComponent, h, ref } from 'vue';
import { describe, expect, it, vi } from 'vitest';

import { createTermReference } from '@/core/service/i18n';
import type { UseField } from '@/web/web/composables/useField';
import OFieldBase from './OFieldBase.vue';

function makeBinding(meta?: { string?: string; stringText?: ReturnType<typeof createTermReference> }): UseField {
  const value = ref('x');
  const record = ref({ Id: '1' });
  return {
    env: { isForm: true, isEditMode: true, viewMode: 'edit', fieldPrefix: null },
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
      global: {
        stubs: {
          'el-form-item': {
            props: ['label'],
            template: '<div class="form-item" :data-label="label"><slot /></div>',
          },
        },
      },
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
      global: {
        stubs: {
          'el-form-item': {
            props: ['label'],
            template: '<div class="form-item" :data-label="label"><slot /></div>',
          },
        },
      },
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
      global: {
        stubs: {
          'el-form-item': {
            props: ['label'],
            template: '<div class="form-item" :data-label="label"><slot /></div>',
          },
        },
      },
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

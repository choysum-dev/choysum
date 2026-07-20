// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount, flushPromises } from '@vue/test-utils';
import { computed, defineComponent, h, nextTick, ref } from 'vue';
import { describe, expect, it, vi } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';

import type { UseField } from '@/web/web/composables/useField';
import { createFieldsGetHelpers } from '@/web/web/stores/fieldsGet';
import type { WebFieldMetadata } from '@/web/web/stores/modelStore';
import OSelectionField from './OSelectionField.vue';

function makeBinding(opts: {
  prop?: string;
  isEditMode?: boolean;
  meta?: WebFieldMetadata;
  store?: any;
}): UseField {
  const value = ref<string | null>('active');
  const record = ref({ Id: '1', Status: 'active' });
  return {
    env: {
      isForm: true,
      isEditMode: opts.isEditMode !== false,
      viewMode: opts.isEditMode === false ? 'display' : 'edit',
      fieldPrefix: null,
    },
    prop: opts.prop || 'Status',
    meta: opts.meta as any,
    fieldRef: () => value as any,
    fieldRefOf: () => value as any,
    recordRef: () => computed(() => record.value) as any,
    registerFields: () => {},
    store: opts.store,
    asView: () => ({ fieldValue: () => value }) as any,
  } as UseField;
}

const EditStub = defineComponent({
  name: 'EditStub',
  setup(_, { slots }) {
    return () => h('div', { class: 'edit-stub' }, slots.default?.({} as any));
  },
});

describe('OSelectionField FieldsGet wiring (P2)', () => {
  const staticMeta: WebFieldMetadata = {
    id: '1',
    type: 'selection',
    typeAnnotation: 'string',
    string: 'Status',
    selection: [
      { value: 'active', label: 'Active' },
      { value: 'archived', label: 'Archived' },
    ],
  };

  it('source contract: no labelText / translateTerm for options (T2.3)', () => {
    const src = readFileSync(resolve(__dirname, './OSelectionField.vue'), 'utf8');
    expect(src).not.toMatch(/\blabelText\b/);
    expect(src).not.toMatch(/translateTerm\s*\(/);
  });

  it('edit onMounted calls ensureFieldsGet and shows loading (T2.4 / T2.8)', async () => {
    const FieldsGet = vi.fn(
      async () =>
        ({
          Status: {
            ...staticMeta,
            string: '状态',
            selection: [
              { value: 'active', label: '启用' },
              { value: 'archived', label: '归档' },
            ],
          },
        }) as any
    );
    let resolveEnsure!: () => void;
    const gate = new Promise<void>(r => {
      resolveEnsure = r;
    });
    const helpers = createFieldsGetHelpers(
      {
        fieldsMetadata: { Status: staticMeta },
        FieldsGet: async (...args) => {
          await gate;
          return FieldsGet(...args);
        },
      },
      { getLang: () => 'zh_CN' }
    );
    const ensureSpy = vi.spyOn(helpers, 'ensureFieldsGet');
    const store = {
      fieldsMetadata: { Status: staticMeta },
      FieldsGet,
      ensureFieldsGet: helpers.ensureFieldsGet,
      getFieldMeta: helpers.getFieldMeta,
      getFieldsGetTranslatedString: helpers.getFieldsGetTranslatedString,
      clearFieldsGetCache: helpers.clearFieldsGetCache,
    };

    const wrapper = mount(OSelectionField as any, {
      props: {
        binding: makeBinding({ meta: staticMeta, store, isEditMode: true }),
        renderMode: 'form',
      },
      global: {
        stubs: {
          OFieldBase: {
            props: ['binding'],
            template: `<div class="ob"><slot name="edit" :fieldValue="() => ({ value: 'active' })" :record="{ Id: '1' }" /></div>`,
          },
          'el-select': {
            props: ['loading', 'disabled', 'modelValue'],
            template: `<div class="sel" :data-loading="String(loading)" :data-disabled="String(disabled)"><slot /></div>`,
          },
          'el-option': true,
        },
      },
    });

    await nextTick();
    expect(ensureSpy).toHaveBeenCalledTimes(1);
    expect(ensureSpy.mock.calls[0]![0]).toEqual(['Status']);
    expect(wrapper.get('.sel').attributes('data-loading')).toBe('true');

    resolveEnsure();
    await flushPromises();
    await nextTick();
    expect(wrapper.get('.sel').attributes('data-loading')).toBe('false');
    expect(FieldsGet).toHaveBeenCalledTimes(1);
    expect(helpers.getFieldMeta('Status')?.selection?.[0]?.label).toBe('启用');
  });

  it('two instances share one RPC via ensureFieldsGet cache (T2.5 / T2.6)', async () => {
    const FieldsGet = vi.fn(async () => ({
      Status: {
        ...staticMeta,
        selection: [{ value: 'active', label: '启用' }],
      },
    }));
    const helpers = createFieldsGetHelpers(
      { fieldsMetadata: { Status: staticMeta }, FieldsGet },
      { getLang: () => 'zh_CN' }
    );
    const store = { fieldsMetadata: { Status: staticMeta }, FieldsGet, ...helpers };

    const mountOne = (isEditMode: boolean) =>
      mount(OSelectionField as any, {
        props: {
          binding: makeBinding({ meta: staticMeta, store, isEditMode }),
          renderMode: isEditMode ? 'form' : 'table',
        },
        global: {
          stubs: {
            OFieldBase: {
              template: `<div><slot name="edit" :fieldValue="() => ({ value: 'active' })" :record="{}" /><slot name="display" :fieldValue="() => ({ value: 'active' })" :record="{}" /></div>`,
            },
            'el-select': { template: '<div class="sel"><slot /></div>' },
            'el-option': true,
          },
        },
      });

    mountOne(true);
    mountOne(false);
    mountOne(false);
    await flushPromises();
    expect(FieldsGet).toHaveBeenCalledTimes(1);
  });

  it('merges FieldsGet options with props.selection filter (T2.7)', async () => {
    const FieldsGet = vi.fn(async () => ({
      Status: {
        ...staticMeta,
        selection: [
          { value: 'active', label: '启用' },
          { value: 'archived', label: '归档' },
        ],
      },
    }));
    const helpers = createFieldsGetHelpers(
      { fieldsMetadata: { Status: staticMeta }, FieldsGet },
      { getLang: () => 'zh_CN' }
    );
    const store = { fieldsMetadata: { Status: staticMeta }, FieldsGet, ...helpers };

    const wrapper = mount(OSelectionField as any, {
      props: {
        binding: makeBinding({ meta: staticMeta, store, isEditMode: true }),
        selection: ['archived'],
        renderMode: 'form',
      },
      global: {
        stubs: {
          OFieldBase: {
            template: `<div class="ob"><slot name="edit" :fieldValue="() => ({ value: null })" :record="{}" /></div>`,
          },
          'el-select': {
            template: `<div class="sel"><slot /></div>`,
          },
          'el-option': {
            props: ['label', 'value'],
            template: `<div class="opt" :data-label="label" :data-value="value" />`,
          },
        },
      },
    });

    await flushPromises();
    await nextTick();
    const opts = wrapper.findAll('.opt');
    expect(opts).toHaveLength(1);
    expect(opts[0]!.attributes('data-value')).toBe('archived');
    expect(opts[0]!.attributes('data-label')).toBe('归档');
  });
});

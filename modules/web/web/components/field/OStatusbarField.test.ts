// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount, flushPromises } from '@vue/test-utils';
import { computed, nextTick, ref } from 'vue';
import { describe, expect, it, vi } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';

import type { UseField } from '@/web/web/composables/useField';
import { createFieldsGetHelpers } from '@/web/web/stores/fieldsGet';
import type { WebFieldMetadata } from '@/web/web/stores/modelStore';
import OStatusbarField from './OStatusbarField.vue';

function makeBinding(opts: {
  prop?: string;
  isEditMode?: boolean;
  meta?: WebFieldMetadata;
  store?: any;
  value?: string | null;
}): { binding: UseField; value: ReturnType<typeof ref<string | null>> } {
  const value = ref<string | null>(opts.value !== undefined ? opts.value : 'draft');
  const record = ref({ Id: '1', State: value.value });
  const binding = {
    env: {
      isForm: true,
      isEditMode: opts.isEditMode !== false,
      viewMode: opts.isEditMode === false ? 'display' : 'edit',
      fieldPrefix: null,
    },
    prop: opts.prop || 'State',
    meta: opts.meta as any,
    fieldRef: () => value as any,
    fieldRefOf: () => value as any,
    recordRef: () => computed(() => ({ ...record.value, State: value.value })) as any,
    registerFields: () => {},
    store: opts.store,
    asView: () => ({ fieldValue: () => value }) as any,
  } as UseField;
  return { binding, value };
}

const staticMeta: WebFieldMetadata = {
  id: '1',
  type: 'selection',
  typeAnnotation: 'string',
  string: 'State',
  selection: [
    { value: 'draft', label: 'Draft' },
    { value: 'confirmed', label: 'Confirmed' },
    { value: 'done', label: 'Done' },
  ],
};

function mountStatusbar(
  props: Record<string, unknown>,
  binding: UseField,
  opts?: { slot?: 'edit' | 'display' }
) {
  const fieldRef = binding.fieldRef();
  const slotName = opts?.slot ?? 'edit';
  return mount(OStatusbarField as any, {
    props: {
      binding,
      renderMode: 'inline',
      ...props,
    },
    global: {
      stubs: {
        OFieldBase: {
          props: ['binding', 'readonly', 'renderMode'],
          setup() {
            const fieldValue = () => fieldRef as any;
            const record = () => ({ Id: '1', State: (fieldRef as any).value });
            return { fieldValue, record, slotName };
          },
          template: `
            <div class="ob">
              <slot :name="slotName" :fieldValue="fieldValue" :record="record" />
            </div>
          `,
        },
        'el-segmented': {
          props: ['modelValue', 'options', 'disabled'],
          emits: ['update:modelValue'],
          template: `
            <div
              class="o-statusbar"
              :data-disabled="String(disabled)"
              :data-value="String(modelValue ?? '')"
              :data-options="JSON.stringify(options || [])"
            >
              <button
                v-for="opt in options || []"
                :key="String(opt.value)"
                class="seg-opt"
                :data-value="opt.value"
                :disabled="disabled || opt.disabled"
                @click="$emit('update:modelValue', opt.value)"
              >{{ opt.label }}</button>
            </div>
          `,
        },
      },
    },
  });
}

describe('OStatusbarField', () => {
  it('uses chevron class and hides EP selected slider via CSS (D11 contract)', () => {
    const src = readFileSync(resolve(__dirname, './OStatusbarField.vue'), 'utf8');
    expect(src).toContain('class="o-statusbar"');
    expect(src).toContain('el-segmented__item-selected');
    expect(src).toContain('clip-path');
    expect(src).toContain('beforeChange');
    expect(src).not.toMatch(/ElMessageBox/);
  });

  it('OFormView exposes optional #statusbar slot (D6)', () => {
    const src = readFileSync(resolve(__dirname, '../view/OFormView.vue'), 'utf8');
    expect(src).toContain('name="statusbar"');
    expect(src).toMatch(/statusbar\(\):\s*any/);
  });

  it('defaults to non-clickable (disabled segmented)', async () => {
    const helpers = createFieldsGetHelpers(
      { fieldsMetadata: { State: staticMeta }, FieldsGet: async () => ({ State: staticMeta }) },
      { getLang: () => 'en_US' }
    );
    const store = { fieldsMetadata: { State: staticMeta }, ...helpers };
    const { binding } = makeBinding({ meta: staticMeta, store });
    const wrapper = mountStatusbar({}, binding);
    await flushPromises();
    expect(wrapper.get('.o-statusbar').attributes('data-disabled')).toBe('true');
    expect(JSON.parse(wrapper.get('.o-statusbar').attributes('data-options') || '[]').map((o: any) => o.value)).toEqual([
      'draft',
      'confirmed',
      'done',
    ]);
  });

  it('clickable writes value; beforeChange false cancels write', async () => {
    const helpers = createFieldsGetHelpers(
      { fieldsMetadata: { State: staticMeta }, FieldsGet: async () => ({ State: staticMeta }) },
      { getLang: () => 'en_US' }
    );
    const store = { fieldsMetadata: { State: staticMeta }, ...helpers };
    const { binding, value } = makeBinding({ meta: staticMeta, store, value: 'draft' });
    const beforeChange = vi.fn(() => false);
    const wrapper = mountStatusbar({ clickable: true, beforeChange }, binding);
    await flushPromises();

    expect(wrapper.get('.o-statusbar').attributes('data-disabled')).toBe('false');
    await wrapper.get('[data-value="done"]').trigger('click');
    await flushPromises();
    await nextTick();
    expect(beforeChange).toHaveBeenCalledWith('done', 'draft');
    expect(value.value).toBe('draft');

    beforeChange.mockImplementation(() => true);
    await wrapper.get('[data-value="done"]').trigger('click');
    await flushPromises();
    expect(value.value).toBe('done');
  });

  it('writes immediately when clickable and beforeChange omitted', async () => {
    const helpers = createFieldsGetHelpers(
      { fieldsMetadata: { State: staticMeta }, FieldsGet: async () => ({ State: staticMeta }) },
      { getLang: () => 'en_US' }
    );
    const store = { fieldsMetadata: { State: staticMeta }, ...helpers };
    const { binding, value } = makeBinding({ meta: staticMeta, store, value: 'draft' });
    const wrapper = mountStatusbar({ clickable: true }, binding);
    await flushPromises();
    await wrapper.get('[data-value="confirmed"]').trigger('click');
    await flushPromises();
    expect(value.value).toBe('confirmed');
  });

  it('clickable still writes from display slot (viewMode-independent)', async () => {
    const helpers = createFieldsGetHelpers(
      { fieldsMetadata: { State: staticMeta }, FieldsGet: async () => ({ State: staticMeta }) },
      { getLang: () => 'en_US' }
    );
    const store = { fieldsMetadata: { State: staticMeta }, ...helpers };
    const { binding, value } = makeBinding({ meta: staticMeta, store, value: 'draft', isEditMode: false });
    const wrapper = mountStatusbar({ clickable: true }, binding, { slot: 'display' });
    await flushPromises();
    expect(wrapper.get('.o-statusbar').attributes('data-disabled')).toBe('false');
    await wrapper.get('[data-value="done"]').trigger('click');
    await flushPromises();
    expect(value.value).toBe('done');
  });

  it('fails closed when readonly predicate throws', async () => {
    const helpers = createFieldsGetHelpers(
      { fieldsMetadata: { State: staticMeta }, FieldsGet: async () => ({ State: staticMeta }) },
      { getLang: () => 'en_US' }
    );
    const store = { fieldsMetadata: { State: staticMeta }, ...helpers };
    const { binding } = makeBinding({ meta: staticMeta, store, value: 'draft' });
    const wrapper = mountStatusbar(
      {
        clickable: true,
        readonly: () => {
          throw new Error('boom');
        },
      },
      binding
    );
    await flushPromises();
    expect(wrapper.get('.o-statusbar').attributes('data-disabled')).toBe('true');
  });

  it('treats empty string as unset in internal validation (source contract)', () => {
    const src = readFileSync(resolve(__dirname, './OStatusbarField.vue'), 'utf8');
    expect(src).toMatch(/value == null \|\| value === ''/);
  });

  it('applies statusbarVisible whitelist and keeps current fallback', async () => {
    const helpers = createFieldsGetHelpers(
      { fieldsMetadata: { State: staticMeta }, FieldsGet: async () => ({ State: staticMeta }) },
      { getLang: () => 'en_US' }
    );
    const store = { fieldsMetadata: { State: staticMeta }, ...helpers };
    const { binding } = makeBinding({ meta: staticMeta, store, value: 'confirmed' });
    const wrapper = mountStatusbar({ statusbarVisible: ['draft', 'done'] }, binding);
    await flushPromises();
    expect(JSON.parse(wrapper.get('.o-statusbar').attributes('data-options') || '[]').map((o: any) => o.value)).toEqual([
      'draft',
      'done',
      'confirmed',
    ]);
  });

  it('ensures FieldsGet on mount', async () => {
    const FieldsGet = vi.fn(async () => ({ State: staticMeta }));
    const helpers = createFieldsGetHelpers(
      { fieldsMetadata: { State: staticMeta }, FieldsGet },
      { getLang: () => 'en_US' }
    );
    const ensureSpy = vi.spyOn(helpers, 'ensureFieldsGet');
    const store = { fieldsMetadata: { State: staticMeta }, FieldsGet, ...helpers };
    const { binding } = makeBinding({ meta: staticMeta, store });
    mountStatusbar({}, binding);
    await flushPromises();
    expect(ensureSpy).toHaveBeenCalled();
    expect(ensureSpy.mock.calls[0]![0]).toEqual(['State']);
  });
});

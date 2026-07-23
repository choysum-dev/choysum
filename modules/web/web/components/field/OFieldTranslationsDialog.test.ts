// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';
import { nextTick } from 'vue';

import OFieldTranslationsDialog from './OFieldTranslationsDialog.vue';

vi.mock('@/web/web/stores/registry', () => ({
  createStoreByModel: (model: string) => {
    if (model === 'base.Language') {
      return {
        GetActiveLanguages: async () => [
          { Code: 'en_US', Name: 'English (US)' },
          { Code: 'zh_CN', Name: 'Chinese (Simplified)' },
        ],
      };
    }
    return {};
  },
}));

vi.mock('element-plus', async () => {
  const actual = await vi.importActual<any>('element-plus');
  return {
    ...actual,
    ElMessage: { success: vi.fn(), error: vi.fn() },
  };
});

describe('OFieldTranslationsDialog', () => {
  it('loads active languages and saves UpdateFieldTranslations patch', async () => {
    const UpdateFieldTranslations = vi.fn(async () => true);
    const Browse = vi.fn(async () => ({ Name: '您好' }));
    const GetFieldTranslations = vi.fn(async () => ({ en_US: 'Hello', zh_CN: '你好' }));

    const wrapper = mount(OFieldTranslationsDialog, {
      props: {
        modelValue: true,
        store: { GetFieldTranslations, UpdateFieldTranslations, Browse } as any,
        recordId: 'lang-1',
        fieldName: 'Name',
        fieldLabel: 'Name',
      },
      global: {
        stubs: {
          'el-dialog': {
            props: ['modelValue'],
            template: '<div class="dialog"><slot /><slot name="footer" /></div>',
            emits: ['opened', 'closed', 'update:modelValue'],
            mounted() {
              this.$emit('opened');
            },
          },
          'el-form': { template: '<form><slot /></form>' },
          'el-form-item': { props: ['label'], template: '<div class="item" :data-label="label"><slot /></div>' },
          'el-input': {
            props: ['modelValue'],
            emits: ['update:modelValue'],
            template:
              '<input class="input" :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" />',
          },
          'el-button': {
            props: ['loading', 'type'],
            template: '<button class="btn" @click="$emit(\'click\')"><slot /></button>',
          },
        },
      },
    });

    await nextTick();
    await Promise.resolve();
    await nextTick();

    expect(GetFieldTranslations).toHaveBeenCalledWith('lang-1', 'Name');
    expect(wrapper.findAll('.item').length).toBeGreaterThanOrEqual(2);

    const zhInput = wrapper.findAll('.input')[1];
    await zhInput.setValue('您好');
    const saveBtn = wrapper.findAll('.btn').at(-1)!;
    await saveBtn.trigger('click');
    await nextTick();
    await Promise.resolve();

    expect(UpdateFieldTranslations).toHaveBeenCalledWith('lang-1', 'Name', { zh_CN: '您好' });
    expect(Browse).toHaveBeenCalled();
    expect(wrapper.emitted('saved')?.[0]?.[0]).toBe('您好');
  });
});

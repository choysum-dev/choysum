// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { mount } from '@vue/test-utils';
import { describe, expect, it, vi } from 'vitest';
import { nextTick } from 'vue';

import OFieldTranslationsDialog from './OFieldTranslationsDialog.vue';

const { languageResponses } = vi.hoisted(() => ({
  languageResponses: {
    langs: [
      { Code: 'en_US', Name: 'English (US)' },
      { Code: 'zh_CN', Name: 'Chinese (Simplified)' },
    ] as Array<{ Code: string; Name: string }>,
  },
}));

const { i18nStoreState } = vi.hoisted(() => ({
  i18nStoreState: { terminologyLang: '' as string, throwOnAccess: false },
}));

vi.mock('@/web/web/stores/registry', () => ({
  createStoreByModel: (model: string) => {
    if (model === 'base.Language') {
      return {
        GetActiveLanguages: async () => languageResponses.langs,
      };
    }
    return {};
  },
}));

vi.mock('@/web/web/stores/i18nStore', () => ({
  useI18nStore: () => {
    if (i18nStoreState.throwOnAccess) throw new Error('no pinia');
    return { terminologyLang: i18nStoreState.terminologyLang };
  },
}));

vi.mock('element-plus', async () => {
  const actual = await vi.importActual<any>('element-plus');
  return {
    ...actual,
    ElMessage: { success: vi.fn(), error: vi.fn() },
  };
});

const dialogStubs = {
  'el-dialog': {
    props: ['modelValue', 'title'],
    template:
      '<div class="dialog" :data-title="title"><button class="emit-closed" type="button" @click="$emit(\'closed\')" /><slot /><slot name="footer" /></div>',
    emits: ['opened', 'closed', 'update:modelValue'],
    mounted() {
      this.$emit('opened');
    },
  },
  'el-form': { template: '<form><slot /></form>' },
  'el-form-item': { props: ['label'], template: '<div class="item" :data-label="label"><slot /></div>' },
  'el-input': {
    props: ['modelValue', 'maxlength', 'showWordLimit'],
    emits: ['update:modelValue'],
    template:
      '<input class="input" :value="modelValue" :data-maxlength="maxlength" :data-word-limit="showWordLimit" @input="$emit(\'update:modelValue\', $event.target.value)" />',
  },
  'el-button': {
    props: ['loading', 'type'],
    template: '<button class="btn" @click="$emit(\'click\')"><slot /></button>',
  },
};

const loadingDirective = {
  mounted() {},
  updated() {},
};

function mountDialog(props: Record<string, unknown>) {
  return mount(OFieldTranslationsDialog, {
    props: props as any,
    global: {
      stubs: dialogStubs,
      // Element Plus v-loading is not installed in unit tests.
      directives: { loading: loadingDirective },
    },
  });
}

async function flushOpen() {
  await nextTick();
  await Promise.resolve();
  await nextTick();
}

describe('OFieldTranslationsDialog', () => {
  it('loads active languages and saves UpdateFieldTranslations patch', async () => {
    languageResponses.langs = [
      { Code: 'en_US', Name: 'English (US)' },
      { Code: 'zh_CN', Name: 'Chinese (Simplified)' },
    ];
    const UpdateFieldTranslations = vi.fn(async () => true);
    const Browse = vi.fn(async () => ({ Name: '您好' }));
    const GetFieldTranslations = vi.fn(async () => ({ en_US: 'Hello', zh_CN: '你好' }));

    const wrapper = mountDialog({
        modelValue: true,
        store: { GetFieldTranslations, UpdateFieldTranslations, Browse } as any,
        recordId: 'lang-1',
        fieldName: 'Name',
        fieldLabel: 'Name',
      });

    await flushOpen();

    expect(GetFieldTranslations).toHaveBeenCalledWith('lang-1', 'Name');
    const labels = wrapper.findAll('.item').map(el => el.attributes('data-label'));
    expect(labels).toContain('English (US)');
    expect(labels).toContain('Chinese (Simplified)');
    expect(labels.some(l => l?.includes('(zh_CN)'))).toBe(false);

    const zhInput = wrapper.findAll('.input')[1]!;
    await zhInput.setValue('您好');
    const buttons = wrapper.findAll('.btn');
    await buttons[buttons.length - 1]!.trigger('click');
    await flushOpen();

    expect(UpdateFieldTranslations).toHaveBeenCalledWith('lang-1', 'Name', { zh_CN: '您好' });
    expect(Browse).toHaveBeenCalled();
    expect(wrapper.emitted('saved')?.[0]?.[0]).toBe('您好');
  });

  it('clears non-en_US translation with false delete sentinel', async () => {
    languageResponses.langs = [
      { Code: 'en_US', Name: 'English (US)' },
      { Code: 'zh_CN', Name: 'Chinese (Simplified)' },
    ];
    const UpdateFieldTranslations = vi.fn(async () => true);
    const Browse = vi.fn(async () => ({ Name: 'Hello' }));
    const GetFieldTranslations = vi.fn(async () => ({ en_US: 'Hello', zh_CN: '你好' }));

    const wrapper = mountDialog({
        modelValue: true,
        store: { GetFieldTranslations, UpdateFieldTranslations, Browse } as any,
        recordId: 'lang-1',
        fieldName: 'Name',
        fieldLabel: 'Name',
      });

    await flushOpen();
    await wrapper.findAll('.input')[1]!.setValue('');
    const buttons = wrapper.findAll('.btn');
    await buttons[buttons.length - 1]!.trigger('click');
    await flushOpen();

    expect(UpdateFieldTranslations).toHaveBeenCalledWith('lang-1', 'Name', { zh_CN: false });
    expect(wrapper.emitted('saved')?.[0]?.[0]).toBe('Hello');
  });

  it('seeds the current-lang row from draftValue on open and saves it as dirty', async () => {
    languageResponses.langs = [
      { Code: 'en_US', Name: 'English (US)' },
      { Code: 'zh_CN', Name: 'Chinese (Simplified)' },
    ];
    const UpdateFieldTranslations = vi.fn(async () => true);
    const Browse = vi.fn(async () => ({ Name: '英语（美国）111' }));
    const GetFieldTranslations = vi.fn(async () => ({
      en_US: 'English (US)',
      zh_CN: '英语（美国）',
    }));

    const wrapper = mountDialog({
        modelValue: true,
        store: { GetFieldTranslations, UpdateFieldTranslations, Browse } as any,
        recordId: 'lang-1',
        fieldName: 'Name',
        fieldLabel: 'Name',
        draftValue: '英语（美国）111',
        draftLang: 'zh_CN',
      });

    await flushOpen();
    const inputs = wrapper.findAll('.input');
    expect(inputs[0]!.element).toMatchObject({ value: 'English (US)' });
    expect(inputs[1]!.element).toMatchObject({ value: '英语（美国）111' });

    const buttons = wrapper.findAll('.btn');
    await buttons[buttons.length - 1]!.trigger('click');
    await flushOpen();

    expect(UpdateFieldTranslations).toHaveBeenCalledWith('lang-1', 'Name', {
      zh_CN: '英语（美国）111',
    });
  });

  it('shows load error and keeps empty rows when GetFieldTranslations fails', async () => {
    const { ElMessage } = await import('element-plus');
    languageResponses.langs = [
      { Code: 'en_US', Name: 'English (US)' },
      { Code: 'zh_CN', Name: 'Chinese (Simplified)' },
    ];
    const GetFieldTranslations = vi.fn(async () => {
      throw new Error('load boom');
    });
    const wrapper = mountDialog({
        modelValue: true,
        store: { GetFieldTranslations, UpdateFieldTranslations: vi.fn(), Browse: vi.fn() } as any,
        recordId: 'lang-1',
        fieldName: 'Name',
        fieldLabel: 'Name',
      });
    await flushOpen();
    expect(ElMessage.error).toHaveBeenCalled();
    expect(wrapper.findAll('.input')).toHaveLength(0);
  });

  it('injects en_US when active languages omit it and uses code as label fallback', async () => {
    languageResponses.langs = [{ Code: 'zh_CN', Name: '' }];
    const GetFieldTranslations = vi.fn(async () => ({ en_US: 'Hello', zh_CN: '你好' }));
    const wrapper = mountDialog({
        modelValue: true,
        store: { GetFieldTranslations, UpdateFieldTranslations: vi.fn(), Browse: vi.fn(async () => ({})) } as any,
        recordId: 'lang-1',
        fieldName: 'Name',
      });
    await flushOpen();
    const labels = wrapper.findAll('.item').map(el => el.attributes('data-label'));
    expect(labels[0]).toBe('English (US)');
    expect(labels).toContain('zh_CN');
    expect(wrapper.get('.dialog').attributes('data-title')).toMatch(/Translate/);
  });

  it('clears en_US with empty string, respects maxLength, and cancels', async () => {
    languageResponses.langs = [
      { Code: 'en_US', Name: 'English (US)' },
      { Code: 'zh_CN', Name: 'Chinese (Simplified)' },
    ];
    const UpdateFieldTranslations = vi.fn(async () => true);
    const Browse = vi.fn(async () => ({ Name: '' }));
    const GetFieldTranslations = vi.fn(async () => ({ en_US: 'Hello', zh_CN: '你好' }));

    const wrapper = mountDialog({
        modelValue: true,
        store: { GetFieldTranslations, UpdateFieldTranslations, Browse } as any,
        recordId: 'lang-1',
        fieldName: 'Name',
        maxLength: 40,
      });
    await flushOpen();
    expect(wrapper.get('.input').attributes('data-maxlength')).toBe('40');

    await wrapper.findAll('.input')[0]!.setValue('');
    const buttons = wrapper.findAll('.btn');
    await buttons[buttons.length - 1]!.trigger('click');
    await flushOpen();
    expect(UpdateFieldTranslations).toHaveBeenCalledWith('lang-1', 'Name', { en_US: '' });

    const wrapper2 = mountDialog({
        modelValue: true,
        store: { GetFieldTranslations, UpdateFieldTranslations: vi.fn(), Browse: vi.fn() } as any,
        recordId: 'lang-1',
        fieldName: 'Name',
        fieldLabel: 'Name',
      });
    await flushOpen();
    await wrapper2.findAll('.btn')[0]!.trigger('click');
    expect(wrapper2.emitted('update:modelValue')?.at(-1)?.[0]).toBe(false);
  });

  it('skips UpdateFieldTranslations when nothing is dirty but still browses and closes', async () => {
    languageResponses.langs = [
      { Code: 'en_US', Name: 'English (US)' },
      { Code: 'zh_CN', Name: 'Chinese (Simplified)' },
    ];
    const UpdateFieldTranslations = vi.fn(async () => true);
    const Browse = vi.fn(async () => ({ Name: 'Hello' }));
    const GetFieldTranslations = vi.fn(async () => ({ en_US: 'Hello', zh_CN: '你好' }));
    const wrapper = mountDialog({
        modelValue: true,
        store: { GetFieldTranslations, UpdateFieldTranslations, Browse } as any,
        recordId: 'lang-1',
        fieldName: 'Name',
        fieldLabel: 'Name',
      });
    await flushOpen();
    const buttons = wrapper.findAll('.btn');
    await buttons[buttons.length - 1]!.trigger('click');
    await flushOpen();
    expect(UpdateFieldTranslations).not.toHaveBeenCalled();
    expect(Browse).toHaveBeenCalled();
    expect(wrapper.emitted('update:modelValue')?.at(-1)?.[0]).toBe(false);
  });

  it('surfaces save errors without closing the dialog', async () => {
    const { ElMessage } = await import('element-plus');
    languageResponses.langs = [
      { Code: 'en_US', Name: 'English (US)' },
      { Code: 'zh_CN', Name: 'Chinese (Simplified)' },
    ];
    const UpdateFieldTranslations = vi.fn(async () => {
      throw new Error('save boom');
    });
    const Browse = vi.fn(async () => ({ Name: 'Hello' }));
    const GetFieldTranslations = vi.fn(async () => ({ en_US: 'Hello', zh_CN: '你好' }));
    const wrapper = mountDialog({
        modelValue: true,
        store: { GetFieldTranslations, UpdateFieldTranslations, Browse } as any,
        recordId: 'lang-1',
        fieldName: 'Name',
        fieldLabel: 'Name',
      });
    await flushOpen();
    await wrapper.findAll('.input')[1]!.setValue('新');
    const buttons = wrapper.findAll('.btn');
    await buttons[buttons.length - 1]!.trigger('click');
    await flushOpen();
    expect(ElMessage.error).toHaveBeenCalled();
    expect(wrapper.emitted('update:modelValue')?.some(args => args[0] === false)).toBeFalsy();
  });

  it('resolves draftLang from i18n store when prop is omitted', async () => {
    i18nStoreState.terminologyLang = 'zh_CN';
    i18nStoreState.throwOnAccess = false;
    languageResponses.langs = [
      { Code: 'en_US', Name: 'English (US)' },
      { Code: 'zh_CN', Name: 'Chinese (Simplified)' },
    ];
    const GetFieldTranslations = vi.fn(async () => ({
      en_US: 'English (US)',
      zh_CN: '英语（美国）',
    }));
    const wrapper = mountDialog({
        modelValue: true,
        store: { GetFieldTranslations, UpdateFieldTranslations: vi.fn(), Browse: vi.fn() } as any,
        recordId: 'lang-1',
        fieldName: 'Name',
        draftValue: '草稿中文',
      });
    await flushOpen();
    expect(wrapper.findAll('.input')[1]!.element).toMatchObject({ value: '草稿中文' });
  });

  it('skips draft overlay when i18n store is unavailable and draftLang is empty', async () => {
    i18nStoreState.throwOnAccess = true;
    i18nStoreState.terminologyLang = '';
    languageResponses.langs = [
      { Code: 'en_US', Name: 'English (US)' },
      { Code: 'zh_CN', Name: 'Chinese (Simplified)' },
    ];
    const GetFieldTranslations = vi.fn(async () => ({
      en_US: 'Hello',
      zh_CN: '你好',
    }));
    const wrapper = mountDialog({
        modelValue: true,
        store: { GetFieldTranslations, UpdateFieldTranslations: vi.fn(), Browse: vi.fn() } as any,
        recordId: 'lang-1',
        fieldName: 'Name',
        draftValue: 'ignored-draft',
      });
    await flushOpen();
    expect(wrapper.findAll('.input')[0]!.element).toMatchObject({ value: 'Hello' });
    expect(wrapper.findAll('.input')[1]!.element).toMatchObject({ value: '你好' });
  });

  it('skips blank language codes, clears draft with null, emits closed, and shows base hint', async () => {
    i18nStoreState.throwOnAccess = false;
    i18nStoreState.terminologyLang = '';
    languageResponses.langs = [
      { Code: '', Name: 'blank' },
      { Code: 'en_US', Name: 'English (US)' },
      { Code: 'zh_CN', Name: 'Chinese (Simplified)' },
    ];
    const GetFieldTranslations = vi.fn(async () => ({ en_US: 'Hello', zh_CN: '你好' }));
    const wrapper = mountDialog({
        modelValue: true,
        store: { GetFieldTranslations, UpdateFieldTranslations: vi.fn(), Browse: vi.fn() } as any,
        recordId: 'lang-1',
        fieldName: 'Name',
        draftLang: 'zh_CN',
        draftValue: null,
      });
    await flushOpen();
    expect(wrapper.findAll('.input')).toHaveLength(2);
    expect(wrapper.findAll('.input')[1]!.element).toMatchObject({ value: '' });
    expect(wrapper.text()).toContain('Base language');
    await wrapper.find('.emit-closed').trigger('click');
    expect(wrapper.emitted('closed')).toBeTruthy();
  });

  it('treats non-object translation maps as empty and still saves new langs', async () => {
    languageResponses.langs = [
      { Code: 'en_US', Name: 'English (US)' },
      { Code: 'zh_CN', Name: 'Chinese (Simplified)' },
    ];
    const UpdateFieldTranslations = vi.fn(async () => true);
    const Browse = vi.fn(async () => ({ Name: 'Hello' }));
    const GetFieldTranslations = vi.fn(async () => null as any);
    const wrapper = mountDialog({
        modelValue: true,
        store: { GetFieldTranslations, UpdateFieldTranslations, Browse } as any,
        recordId: 'lang-1',
        fieldName: 'Name',
      });
    await flushOpen();
    await wrapper.findAll('.input')[0]!.setValue('Hello');
    await wrapper.findAll('.input')[1]!.setValue('你好');
    const buttons = wrapper.findAll('.btn');
    await buttons[buttons.length - 1]!.trigger('click');
    await flushOpen();
    expect(UpdateFieldTranslations).toHaveBeenCalledWith('lang-1', 'Name', {
      en_US: 'Hello',
      zh_CN: '你好',
    });
    expect(wrapper.emitted('saved')?.[0]?.[0]).toBe('Hello');
  });

  it('emits saved null when Browse omits the field', async () => {
    languageResponses.langs = [
      { Code: 'en_US', Name: 'English (US)' },
      { Code: 'zh_CN', Name: 'Chinese (Simplified)' },
    ];
    const UpdateFieldTranslations = vi.fn(async () => true);
    const Browse = vi.fn(async () => ({}));
    const GetFieldTranslations = vi.fn(async () => ({ en_US: 'Hello' }));
    const wrapper = mountDialog({
        modelValue: true,
        store: { GetFieldTranslations, UpdateFieldTranslations, Browse } as any,
        recordId: 'lang-1',
        fieldName: 'Name',
      });
    await flushOpen();
    await wrapper.findAll('.input')[0]!.setValue('Hello!');
    const buttons = wrapper.findAll('.btn');
    await buttons[buttons.length - 1]!.trigger('click');
    await flushOpen();
    expect(wrapper.emitted('saved')?.[0]?.[0]).toBeNull();
  });
});


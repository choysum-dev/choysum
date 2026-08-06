// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h, nextTick } from 'vue';
import { flushPromises, mount } from '@vue/test-utils';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

const listRegisteredModelNames = vi.fn(() => [
  'web.TranslationTerm',
  'auth.TranslationTerm',
  'core.TranslationTerm',
  'base.Company',
  '.TranslationTerm',
]);

const stores = new Map<string, { UpdateById: ReturnType<typeof vi.fn> }>();
const createStoreByModel = vi.fn((modelName: string) => {
  if (!stores.has(modelName)) {
    stores.set(modelName, {
      UpdateById: vi.fn(async () => ({ id: '1' })),
    });
  }
  return stores.get(modelName)!;
});

const reloadTerminology = vi.fn(async () => ({}));
const i18nStore = {
  terminologyLang: 'zh_CN',
  reloadTerminology,
};

const authStore = {
  tokens: { accessToken: 'tok' } as { accessToken?: string } | null,
};

const downloadTerminologyPo = vi.fn(async () => new Blob(['msgid ""']));
const elMessageError = vi.fn();

vi.mock('vue-router', () => ({
  useRoute: () => ({ fullPath: '/base/terminology' }),
}));

vi.mock('@/web/web/i18n', () => ({
  createTranslate: () => ({ _t: (msg: string) => msg }),
}));

vi.mock('@/web/web/stores/registry', () => ({
  listRegisteredModelNames: (...args: unknown[]) => listRegisteredModelNames(...args),
  createStoreByModel: (...args: unknown[]) => createStoreByModel(...(args as [string])),
}));

vi.mock('@/web/web/stores/storeScopeManager', () => ({
  useScopeManager: () => ({ menuScopeManager: { id: 'menu' } }),
}));

vi.mock('@/web/web/stores/i18nStore', () => ({
  useI18nStore: () => i18nStore,
}));

vi.mock('@/auth/web/stores/auth', () => ({
  useAuthStore: () => authStore,
}));

vi.mock('@/web/web/stores/i18nStore/po_download', () => ({
  downloadTerminologyPo: (...args: unknown[]) => downloadTerminologyPo(...args),
}));

vi.mock('element-plus', () => ({
  ElMessage: { error: (...args: unknown[]) => elMessageError(...args), warning: vi.fn() },
}));

vi.mock('@/web/web/components/page/OPage.vue', () => ({
  default: defineComponent({
    name: 'OPage',
    setup(_, { slots }) {
      return () => h('div', { class: 'o-page' }, slots.default?.());
    },
  }),
}));

vi.mock('@/web/web/components/view/OListView.vue', () => ({
  default: defineComponent({
    name: 'OListView',
    props: { store: { type: Object, required: true } },
    setup(_, { slots }) {
      return () => h('div', { class: 'o-list-view' }, slots.default?.());
    },
  }),
}));

vi.mock('@/web/web/components/view/OSearchView.vue', () => ({ default: {} }));
vi.mock('@/web/web/components/vtable/OVColumn.vue', () => ({ default: true }));
vi.mock('@/web/web/components/field/OVarCharField.vue', () => ({ default: true }));
vi.mock('@/web/web/components/field/OTextField.vue', () => ({ default: true }));

import TerminologyEditor from './TerminologyEditor.vue';

const ElSelectStub = defineComponent({
  name: 'ElSelect',
  props: { modelValue: { type: String, default: '' } },
  emits: ['update:modelValue', 'change'],
  setup(props, { emit, slots }) {
    return () =>
      h(
        'select',
        {
          class: 'app-select',
          value: props.modelValue,
          onChange: async (e: Event) => {
            const value = (e.target as HTMLSelectElement).value;
            // Match Element Plus: model updates before @change handlers read it.
            emit('update:modelValue', value);
            await nextTick();
            emit('change', value);
          },
        },
        slots.default?.()
      );
  },
});

const ElOptionStub = defineComponent({
  name: 'ElOption',
  props: { label: String, value: String },
  setup(props) {
    return () => h('option', { value: props.value }, props.label);
  },
});

const ElInputStub = defineComponent({
  name: 'ElInput',
  props: { modelValue: { type: String, default: '' } },
  emits: ['update:modelValue'],
  setup(props, { emit }) {
    return () =>
      h('input', {
        class: 'module-input',
        value: props.modelValue,
        onInput: (e: Event) => emit('update:modelValue', (e.target as HTMLInputElement).value),
      });
  },
});

const ElButtonStub = defineComponent({
  name: 'ElButton',
  props: { disabled: Boolean, loading: Boolean },
  emits: ['click'],
  setup(props, { emit, slots }) {
    return () =>
      h(
        'button',
        {
          class: 'download-btn',
          // Keep clickable in tests even when the real control would be disabled.
          'data-disabled': props.disabled ? '1' : '0',
          onClick: () => emit('click'),
        },
        slots.default?.()
      );
  },
});

const ElEmptyStub = defineComponent({
  name: 'ElEmpty',
  props: { description: String },
  setup(props) {
    return () => h('div', { class: 'el-empty' }, props.description);
  },
});

function mountPage() {
  return mount(TerminologyEditor as any, {
    global: {
      stubs: {
        'el-select': ElSelectStub,
        'el-option': ElOptionStub,
        'el-input': ElInputStub,
        'el-button': ElButtonStub,
        'el-empty': ElEmptyStub,
      },
    },
  });
}

async function selectApp(wrapper: ReturnType<typeof mountPage>, app: string) {
  const select = wrapper.find('.app-select');
  (select.element as HTMLSelectElement).value = app;
  await select.trigger('change');
  await flushPromises();
  await flushPromises();
}

describe('TerminologyEditor page', () => {
  let createObjectURL: ReturnType<typeof vi.fn>;
  let revokeObjectURL: ReturnType<typeof vi.fn>;
  let clickSpy: ReturnType<typeof vi.fn>;
  let createElementSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    stores.clear();
    listRegisteredModelNames.mockReset();
    listRegisteredModelNames.mockImplementation(() => [
      'web.TranslationTerm',
      'auth.TranslationTerm',
      'core.TranslationTerm',
      'base.Company',
      '.TranslationTerm',
    ]);
    createStoreByModel.mockReset();
    createStoreByModel.mockImplementation((modelName: string) => {
      if (modelName.includes('broken')) {
        throw new Error('TranslationTerm store is not available for this application');
      }
      if (!stores.has(modelName)) {
        stores.set(modelName, {
          UpdateById: vi.fn(async () => ({ id: '1' })),
        });
      }
      return stores.get(modelName)!;
    });
    reloadTerminology.mockReset();
    reloadTerminology.mockResolvedValue({});
    downloadTerminologyPo.mockReset();
    downloadTerminologyPo.mockResolvedValue(new Blob(['msgid ""']));
    elMessageError.mockReset();
    i18nStore.terminologyLang = 'zh_CN';
    authStore.tokens = { accessToken: 'tok' };

    createObjectURL = vi.fn(() => 'blob:po');
    revokeObjectURL = vi.fn();
    vi.stubGlobal('URL', { createObjectURL, revokeObjectURL });

    clickSpy = vi.fn();
    const realCreate = document.createElement.bind(document);
    createElementSpy = vi.spyOn(document, 'createElement').mockImplementation((tag: string) => {
      const el = realCreate(tag);
      if (tag === 'a') {
        el.click = clickSpy;
      }
      return el;
    });
  });

  afterEach(() => {
    createElementSpy.mockRestore();
    vi.unstubAllGlobals();
  });

  it('loads TranslationTerm applications on mount and skips core/empty', async () => {
    const wrapper = mountPage();
    await flushPromises();
    expect(listRegisteredModelNames).toHaveBeenCalled();
    const options = wrapper.findAll('option').map((o) => o.attributes('value'));
    expect(options).toEqual(['auth', 'web']);
    expect(wrapper.find('.el-empty').exists()).toBe(true);
    expect(wrapper.find('.o-list-view').exists()).toBe(false);
  });

  it('creates term store on application change and clears module filter', async () => {
    const wrapper = mountPage();
    await flushPromises();

    const moduleInput = wrapper.find('.module-input');
    await moduleInput.setValue('old-mod');
    await selectApp(wrapper, 'web');

    expect(createStoreByModel).toHaveBeenCalledWith('web.TranslationTerm', expect.any(Object));
    expect((moduleInput.element as HTMLInputElement).value).toBe('');
    expect(wrapper.find('.o-list-view').exists()).toBe(true);

    await selectApp(wrapper, '');
    expect(wrapper.find('.el-empty').exists()).toBe(true);
  });

  it('shows error when TranslationTerm store cannot be created', async () => {
    const wrapper = mountPage();
    await flushPromises();
    createStoreByModel.mockImplementationOnce(() => {
      throw new Error('nope');
    });
    await selectApp(wrapper, 'web');
    expect(elMessageError).toHaveBeenCalledWith('nope');
  });

  it('wraps UpdateById once and reloads terminology after save', async () => {
    const wrapper = mountPage();
    await flushPromises();
    await selectApp(wrapper, 'web');

    expect(createStoreByModel).toHaveBeenCalledWith('web.TranslationTerm', expect.any(Object));
    const store = stores.get('web.TranslationTerm')!;
    expect(store).toBeTruthy();
    await store.UpdateById('1', { Value: 'x' });
    expect(reloadTerminology).toHaveBeenCalledTimes(1);

    // Re-select same app: store is reused and must not nest wrappers.
    await selectApp(wrapper, 'auth');
    await selectApp(wrapper, 'web');
    await store.UpdateById('1', { Value: 'y' });
    expect(reloadTerminology).toHaveBeenCalledTimes(2);

    // Change without payload falls back to selectedApp.
    await wrapper.findComponent({ name: 'ElSelect' }).vm.$emit('change');
    await flushPromises();
    expect(stores.has('web.TranslationTerm')).toBe(true);
  });

  it('ignores reloadTerminology failures after a successful save', async () => {
    reloadTerminology.mockRejectedValueOnce(new Error('reload failed'));
    const wrapper = mountPage();
    await flushPromises();
    await selectApp(wrapper, 'web');
    const store = stores.get('web.TranslationTerm')!;
    await expect(store.UpdateById('1', { Value: 'x' })).resolves.toEqual({ id: '1' });
  });

  it('downloads PO when app, module, and lang are set', async () => {
    const wrapper = mountPage();
    await flushPromises();
    await selectApp(wrapper, 'web');
    await wrapper.find('.module-input').setValue('web');
    await nextTick();

    await wrapper.find('.download-btn').trigger('click');
    await flushPromises();

    expect(downloadTerminologyPo).toHaveBeenCalledWith({
      lang: 'zh_CN',
      application: 'web',
      module: 'web',
      accessToken: 'tok',
    });
    expect(createObjectURL).toHaveBeenCalled();
    expect(clickSpy).toHaveBeenCalled();
    expect(revokeObjectURL).toHaveBeenCalledWith('blob:po');
  });

  it('no-ops download when canDownloadPo is false', async () => {
    i18nStore.terminologyLang = '';
    const wrapper = mountPage();
    await flushPromises();
    await wrapper.find('.download-btn').trigger('click');
    await flushPromises();
    expect(downloadTerminologyPo).not.toHaveBeenCalled();
  });

  it('shows PO download errors', async () => {
    downloadTerminologyPo.mockRejectedValueOnce(new Error('boom'));
    const wrapper = mountPage();
    await flushPromises();
    await selectApp(wrapper, 'web');
    await wrapper.find('.module-input').setValue('web');
    await nextTick();
    await wrapper.find('.download-btn').trigger('click');
    await flushPromises();
    expect(elMessageError).toHaveBeenCalledWith('boom');
  });

  it('falls back to default messages when errors have no message', async () => {
    downloadTerminologyPo.mockRejectedValueOnce({});
    const wrapper = mountPage();
    await flushPromises();
    await selectApp(wrapper, 'web');
    await wrapper.find('.module-input').setValue('web');
    await nextTick();
    await wrapper.find('.download-btn').trigger('click');
    await flushPromises();
    expect(elMessageError).toHaveBeenCalledWith('PO download failed');

    createStoreByModel.mockImplementationOnce(() => {
      throw {};
    });
    await selectApp(wrapper, 'auth');
    expect(elMessageError).toHaveBeenCalledWith(
      'TranslationTerm store is not available for this application'
    );
  });

  it('passes undefined accessToken when auth tokens are missing', async () => {
    authStore.tokens = null;
    const wrapper = mountPage();
    await flushPromises();
    await selectApp(wrapper, 'web');
    await wrapper.find('.module-input').setValue('web');
    await nextTick();
    await wrapper.find('.download-btn').trigger('click');
    await flushPromises();
    expect(downloadTerminologyPo).toHaveBeenCalledWith(
      expect.objectContaining({ accessToken: undefined })
    );
  });
});

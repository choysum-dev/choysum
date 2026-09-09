// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h, nextTick } from 'vue';
import { createPinia, setActivePinia } from 'pinia';
import { createFeStubRouter } from 'vue-router';
import { ElButton, ElEmpty, ElInput, ElOption, ElSelect } from 'element-plus';

import { registerStoreFactory } from '@/web/web/stores/registry';
import { flushPromises, fnRecorder, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import OPage from '@/web/web/components/page/OPage.vue';
import OListView from '@/web/web/components/view/OListView.vue';
import OVarCharField from '@/web/web/components/field/OVarCharField.vue';
import OTextField from '@/web/web/components/field/OTextField.vue';
import OVColumn from '@/web/web/components/vtable/OVColumn.vue';
import OSearchView from '@/web/web/components/view/OSearchView.vue';
import TerminologyEditor from './TerminologyEditor.vue';

const heavySfcs = [OPage, OListView, OVarCharField, OTextField, OVColumn, OSearchView];
const epSfcs = [ElSelect, ElOption, ElInput, ElButton, ElEmpty];

describe('TerminologyEditor page', () => {
  const stores = new Map<string, { UpdateById: ReturnType<typeof fnRecorder> }>();
  const listNames = fnRecorder(() => [
    'web.TranslationTerm',
    'auth.TranslationTerm',
    'core.TranslationTerm',
    'base.Company',
    '.TranslationTerm',
  ]);
  const reloadTerminology = fnRecorder(async () => ({}));
  const createStore = fnRecorder((modelName: string) => {
    if (!stores.has(modelName)) {
      stores.set(modelName, {
        UpdateById: fnRecorder(async () => ({ id: '1' })),
      });
    }
    return stores.get(modelName)!;
  });

  beforeEach(() => {
    setActivePinia(createPinia());
    stores.clear();
    listNames.mockReset();
    listNames.mockImplementation(() => [
      'web.TranslationTerm',
      'auth.TranslationTerm',
      'core.TranslationTerm',
      'base.Company',
      '.TranslationTerm',
    ]);
    createStore.mockReset();
    createStore.mockImplementation((modelName: string) => {
      if (!stores.has(modelName)) {
        stores.set(modelName, {
          UpdateById: fnRecorder(async () => ({ id: '1' })),
        });
      }
      return stores.get(modelName)!;
    });
    reloadTerminology.mockReset();
    reloadTerminology.mockImplementation(async () => ({}));

    // Real registry (test import); page uses deps because FE path stubs replace registry on pages.
    registerStoreFactory('web.TranslationTerm', () => createStore('web.TranslationTerm'));
    registerStoreFactory('auth.TranslationTerm', () => createStore('auth.TranslationTerm'));
    registerStoreFactory('core.TranslationTerm', () => createStore('core.TranslationTerm'));

    stubSfc(OPage as any, {
      name: 'OPage',
      setup(_: any, { slots }: any) {
        return () => h('div', { class: 'o-page', 'data-test': 'page' }, slots.default?.());
      },
    } as any);
    stubSfc(OListView as any, {
      name: 'OListView',
      props: { store: { type: Object, required: true } },
      setup(_: any, { slots }: any) {
        return () => h('div', { class: 'o-list-view', 'data-test': 'list' }, slots.default?.());
      },
    } as any);
    for (const Comp of [OVarCharField, OTextField, OVColumn, OSearchView]) {
      stubSfc(Comp as any, {
        name: (Comp as any).name || 'HeavyField',
        setup: () => () => null,
      } as any);
    }

    stubSfc(ElSelect as any, {
      name: 'ElSelect',
      props: { modelValue: { type: String, default: '' } },
      emits: ['update:modelValue', 'change'],
      setup(props: any, { emit, slots }: any) {
        return () =>
          h(
            'select',
            {
              class: 'app-select',
              'data-test': 'app-select',
              value: props.modelValue,
              onChange: async (e: Event) => {
                const value = (e.target as HTMLSelectElement).value;
                emit('update:modelValue', value);
                await nextTick();
                emit('change', value);
              },
            },
            slots.default?.()
          );
      },
    } as any);
    stubSfc(ElOption as any, {
      name: 'ElOption',
      props: { label: String, value: String },
      setup(props: any) {
        return () => h('option', { value: props.value }, props.label);
      },
    } as any);
    stubSfc(ElInput as any, {
      name: 'ElInput',
      props: { modelValue: { type: String, default: '' } },
      emits: ['update:modelValue'],
      setup(props: any, { emit }: any) {
        return () =>
          h('input', {
            class: 'module-input',
            'data-test': 'module-input',
            value: props.modelValue,
            onInput: (e: Event) => emit('update:modelValue', (e.target as HTMLInputElement).value),
          });
      },
    } as any);
    stubSfc(ElButton as any, {
      name: 'ElButton',
      props: { disabled: Boolean, loading: Boolean },
      emits: ['click'],
      setup(props: any, { emit, slots }: any) {
        return () =>
          h(
            'button',
            {
              class: 'download-btn',
              'data-test': 'download-po',
              'data-disabled': props.disabled ? '1' : '0',
              onClick: () => emit('click'),
            },
            slots.default?.()
          );
      },
    } as any);
    stubSfc(ElEmpty as any, {
      name: 'ElEmpty',
      props: { description: String },
      setup(props: any) {
        return () => h('div', { class: 'el-empty', 'data-test': 'empty' }, props.description);
      },
    } as any);
  });

  afterEach(() => {
    for (const Comp of heavySfcs) restoreSfc(Comp as any);
    for (const Comp of epSfcs) restoreSfc(Comp as any);
  });

  function mountPage() {
    const { router } = createFeStubRouter({
      route: { path: '/web/terminology', fullPath: '/web/terminology' },
    });
    const pinia = createPinia();
    setActivePinia(pinia);
    return mountApp(TerminologyEditor as any, {
      props: {
        deps: {
          listRegisteredModelNames: () => listNames(),
          createStoreByModel: (model: string, _opts?: any) => createStore(model) as any,
          reloadTerminology: () => reloadTerminology(),
        },
      },
      plugins: [pinia, router],
      // Template uses unresolved el-* tags (only ElMessage is imported).
      stubs: {
        'el-select': defineComponent({
          name: 'ElSelectStub',
          props: { modelValue: { type: String, default: '' } },
          emits: ['update:modelValue', 'change'],
          setup(props: any, { emit, slots }: any) {
            return () =>
              h(
                'select',
                {
                  class: 'app-select',
                  'data-test': 'app-select',
                  value: props.modelValue,
                  onChange: async (e: Event) => {
                    const value = (e.target as HTMLSelectElement).value;
                    emit('update:modelValue', value);
                    await nextTick();
                    emit('change', value);
                  },
                },
                slots.default?.()
              );
          },
        }),
        'el-option': defineComponent({
          name: 'ElOptionStub',
          props: { label: String, value: String },
          setup(props: any) {
            return () => h('option', { value: props.value }, props.label);
          },
        }),
        'el-input': defineComponent({
          name: 'ElInputStub',
          props: { modelValue: { type: String, default: '' } },
          emits: ['update:modelValue'],
          setup(props: any, { emit }: any) {
            return () =>
              h('input', {
                class: 'module-input',
                'data-test': 'module-input',
                value: props.modelValue,
                onInput: (e: Event) => emit('update:modelValue', (e.target as HTMLInputElement).value),
              });
          },
        }),
        'el-button': defineComponent({
          name: 'ElButtonStub',
          props: { disabled: Boolean, loading: Boolean },
          emits: ['click'],
          setup(props: any, { emit, slots }: any) {
            return () =>
              h(
                'button',
                {
                  class: 'download-btn',
                  'data-test': 'download-po',
                  'data-disabled': props.disabled ? '1' : '0',
                  onClick: () => emit('click'),
                },
                slots.default?.()
              );
          },
        }),
        'el-empty': defineComponent({
          name: 'ElEmptyStub',
          props: { description: String },
          setup(props: any) {
            return () => h('div', { class: 'el-empty', 'data-test': 'empty' }, props.description);
          },
        }),
      },
    });
  }

  async function selectApp(mounted: ReturnType<typeof mountPage>, app: string) {
    const select = mounted.q('[data-test="app-select"]') as HTMLSelectElement | null;
    expect(select).toBeTruthy();
    select!.value = app;
    select!.dispatchEvent(new Event('change', { bubbles: true }));
    await flushPromises();
    await flushPromises();
  }

  test('loads TranslationTerm apps on mount and skips core/empty', async () => {
    const mounted = mountPage();
    await flushPromises();
    expect(listNames.calls.length).toBeGreaterThan(0);
    const options = mounted.qa('option').map(o => (o as HTMLOptionElement).value);
    expect(options).toEqual(['auth', 'web']);
    expect(mounted.q('[data-test="empty"]')).toBeTruthy();
    // Path stub ChildView or our list stub — either means the list is not shown yet.
    expect(mounted.q('[data-test="list"]') || mounted.q('[data-testid="fe-stub-child-view"]')).toBeFalsy();
    mounted.unmount();
  });

  test('selecting an app shows the list and UpdateById reloads terminology', async () => {
    const mounted = mountPage();
    await flushPromises();
    await selectApp(mounted, 'web');

    expect(createStore.calls.some(c => c[0] === 'web.TranslationTerm')).toBe(true);
    const listEl =
      mounted.q('[data-test="list"]') || mounted.q('[data-testid="fe-stub-child-view"]');
    expect(listEl).toBeTruthy();

    const store = stores.get('web.TranslationTerm')!;
    expect(store).toBeTruthy();
    await store.UpdateById('1', { Value: 'x' });
    expect(reloadTerminology.calls.length).toBe(1);

    await selectApp(mounted, 'auth');
    await selectApp(mounted, 'web');
    await store.UpdateById('1', { Value: 'y' });
    expect(reloadTerminology.calls.length).toBe(2);

    reloadTerminology.mockImplementation(async () => {
      throw new Error('reload failed');
    });
    const afterFail = await store.UpdateById('1', { Value: 'z' });
    expect(afterFail).toEqual({ id: '1' });

    // PO download needs URL.createObjectURL; skip assert when the host lacks it.
    if (typeof URL !== 'undefined' && typeof (URL as any).createObjectURL === 'function') {
      const btn = mounted.q('[data-test="download-po"]') as HTMLButtonElement | null;
      expect(btn).toBeTruthy();
    }

    mounted.unmount();
  });
});

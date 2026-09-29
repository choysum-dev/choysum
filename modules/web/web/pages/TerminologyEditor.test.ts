// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h } from 'vue';
import { createPinia, setActivePinia } from 'pinia';
import * as VueRouter from 'vue-router';
const createFeStubRouter = (VueRouter as any).createFeStubRouter;

import { flushPromises, fnRecorder, mountApp, restoreSfc, stubSfc } from '@/web/web/__tests__/mountApp';
import {
  ChoyButton,
  ChoyListView,
  ChoyPage,
  ChoySearchView,
  ChoyTextField,
  ChoyVarcharField,
  ChoyTableColumn,
} from '@/web';
import TerminologyEditor from './TerminologyEditor.vue';

const heavySfcs = [
  ChoyPage,
  ChoyListView,
  ChoyVarcharField,
  ChoyTextField,
  ChoyTableColumn,
  ChoySearchView,
  ChoyButton,
];

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

    stubSfc(ChoyPage as any, {
      name: 'ChoyPage',
      setup(_: any, { slots }: any) {
        return () => h('div', { class: 'choy-page', 'data-test': 'page' }, slots.default?.());
      },
    } as any);
    stubSfc(ChoyListView as any, {
      name: 'ChoyListView',
      props: { store: { type: Object, required: true } },
      setup(_: any, { slots }: any) {
        return () => h('div', { class: 'choy-list-view', 'data-test': 'list' }, slots.default?.());
      },
    } as any);
    for (const Comp of [ChoyVarcharField, ChoyTextField, ChoyTableColumn, ChoySearchView]) {
      stubSfc(Comp as any, {
        name: (Comp as any).name || 'HeavyField',
        setup: () => () => null,
      } as any);
    }
    stubSfc(ChoyButton as any, {
      name: 'ChoyButton',
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
            slots.default?.(),
          );
      },
    } as any);
  });

  afterEach(() => {
    for (const Comp of heavySfcs) restoreSfc(Comp as any);
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
    });
  }

  async function selectApp(mounted: ReturnType<typeof mountPage>, app: string) {
    const select = mounted.q('select') as HTMLSelectElement | null;
    expect(select).toBeTruthy();
    // QuickJS host select.options may be undefined; set value directly.
    (select as any).value = app;
    select!.dispatchEvent(new Event('change', { bubbles: true }));
    await flushPromises();
    await flushPromises();
  }

  test('loads TranslationTerm apps on mount and skips core/empty', async () => {
    const mounted = mountPage();
    await flushPromises();
    expect(listNames.calls.length).toBeGreaterThan(0);
    const options = mounted.qa('option').map(o => (o as HTMLOptionElement).value).filter(Boolean);
    expect(options).toEqual(['auth', 'web']);
    expect(mounted.q('.terminology-empty')).toBeTruthy();
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

    if (typeof URL !== 'undefined' && typeof (URL as any).createObjectURL === 'function') {
      const btn = mounted.q('[data-test="download-po"]') as HTMLButtonElement | null;
      expect(btn).toBeTruthy();
    }

    mounted.unmount();
  });
});

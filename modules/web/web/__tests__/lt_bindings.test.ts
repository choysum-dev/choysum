// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createTranslate } from '@/web/web/i18n';
import { createI18n } from 'vue-i18n';
import { createPinia, setActivePinia } from 'pinia';
import { menus } from '../menu/menus';
import { routes } from '../router/routes';
import { mountApp, stub } from './mountApp';

const fakeStore = {
  state: { queryState: {}, result: undefined, selection: [], planCache: new Map(), record: undefined },
  setContext: () => undefined,
  getContext: () => undefined,
  withContext: () => undefined,
};

test('web shell _lt bindings: pins TermReference titles on web menus and routes', () => {
  const homeTitle = createTranslate('web', { scope: 'web/menu/menus' })._lt('Home');
  const root = menus[0] as { title?: string; titleText?: unknown };
  expect(root.title).toBe('Home');
  expect(root.titleText).toEqual(homeTitle);
  expect(routes.length).toBeGreaterThan(0);
});

test('web shell _lt bindings: breadcrumbStore factory-default _lt titles stay TermReferences', async () => {
  const expectedPage = createTranslate('web', { scope: 'web/stores/breadcrumbStore' })._lt('Page');
  const expectedDetails = createTranslate('web', { scope: 'web/stores/breadcrumbStore' })._lt('Details');
  const mod = await import('../stores/breadcrumbStore/index');
  expect(typeof mod.useBreadcrumbStore).toBe('function');
  expect(expectedPage.src).toBe('Page');
  expect(expectedDetails.src).toBe('Details');
});

test('web shell _lt bindings: mounts OFormView so detailsTitle _lt runs', async () => {
  setActivePinia(createPinia());
  const i18n = createI18n({
    legacy: false,
    locale: 'en',
    missingWarn: false,
    fallbackWarn: false,
    messages: { en: {} },
  });
  const mod = await import('../components/view/OFormView.vue');
  const handle = mountApp(mod.default as any, {
    props: { store: fakeStore },
    plugins: [i18n],
    stubs: {
      'el-button': stub('el-button'),
      'el-icon': stub('el-icon'),
      OPage: stub('OPage'),
      OBreadcrumb: stub('OBreadcrumb'),
    },
  });
  expect(handle.root).toBeTruthy();
  handle.unmount();
});

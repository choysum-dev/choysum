// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import * as VueRouter from 'vue-router';
const createFeStubRouter = (VueRouter as any).createFeStubRouter;
import { createPinia, setActivePinia } from 'pinia';
import { createI18n } from 'vue-i18n';
import { flushPromises, mountApp, stub } from '@/web/web/__tests__/mountApp';
import { replaceStoreFactory } from '@/web/web/stores/registry';
import UserFilterList from './UserFilterList.vue';

test('UserFilterList.vue wires useUserFilterListPage into ChoyPage', async () => {
  setActivePinia(createPinia());
  const restore = replaceStoreFactory('web.UserFilter', () => ({ id: 'web-uf' }));
  const { router } = createFeStubRouter({
    route: {
      path: '/web/user-filters',
      fullPath: '/web/user-filters',
      params: {},
      meta: {},
    },
  });
  const i18n = createI18n({ legacy: false, locale: 'en', messages: { en: {} } });
  try {
    const mounted = mountApp(UserFilterList as any, {
      plugins: [router, i18n],
      stubs: {
        ChoyPage: stub('ChoyPage'),
        UserFilterListView: stub('UserFilterListView'),
      },
    });
    await flushPromises();
    expect(mounted.root).toBeTruthy();
    mounted.unmount();
  } finally {
    restore();
  }
});

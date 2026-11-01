// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import * as VueRouter from 'vue-router';
const createFeStubRouter = (VueRouter as any).createFeStubRouter;
import { createPinia, setActivePinia } from 'pinia';
import { createI18n } from 'vue-i18n';
import { flushPromises, mountApp, stub } from '@/web/web/__tests__/mountApp';
import { replaceStoreFactory } from '@/web/web/stores/registry';
import AppSettingList from './AppSettingList.vue';

test('AppSettingList.vue wires useAppSettingListPage into ChoyPage', async () => {
  setActivePinia(createPinia());
  const restore = replaceStoreFactory('auth.AppSetting', () => ({ id: 'auth-as' }));
  const { router } = createFeStubRouter({
    route: {
      path: '/web/app-settings/auth',
      fullPath: '/web/app-settings/auth',
      params: {},
      meta: { appSettingApplication: 'auth' },
    },
  });
  const i18n = createI18n({ legacy: false, locale: 'en', messages: { en: {} } });
  try {
    const mounted = mountApp(AppSettingList as any, {
      plugins: [router, i18n],
      stubs: {
        ChoyPage: stub('ChoyPage'),
        AppSettingListView: stub('AppSettingListView'),
      },
    });
    await flushPromises();
    expect(mounted.root).toBeTruthy();
    mounted.unmount();
  } finally {
    restore();
  }
});

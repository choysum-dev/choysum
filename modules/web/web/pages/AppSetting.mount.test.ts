// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import * as VueRouter from 'vue-router';
const createFeStubRouter = (VueRouter as any).createFeStubRouter;
import { createPinia, setActivePinia } from 'pinia';
import { createI18n } from 'vue-i18n';
import { flushPromises, mountApp, stub } from '@/web/web/__tests__/mountApp';
import { replaceStoreFactory } from '@/web/web/stores/registry';
import AppSetting from './AppSetting.vue';

test('AppSetting.vue wires useAppSettingDetailPage into ChoyPage', async () => {
  setActivePinia(createPinia());
  const restore = replaceStoreFactory('auth.AppSetting', () => ({ id: 'auth-as' }));
  const { router } = createFeStubRouter({
    route: {
      path: '/web/app-settings/auth/new',
      fullPath: '/web/app-settings/auth/new',
      params: { application: 'auth' },
      meta: {},
    },
  });
  const i18n = createI18n({ legacy: false, locale: 'en', messages: { en: {} } });
  try {
    const mounted = mountApp(AppSetting as any, {
      plugins: [router, i18n],
      stubs: {
        ChoyPage: stub('ChoyPage'),
        AppSettingFormView: stub('AppSettingFormView'),
      },
    });
    await flushPromises();
    expect(mounted.root).toBeTruthy();
    mounted.unmount();
  } finally {
    restore();
  }
});

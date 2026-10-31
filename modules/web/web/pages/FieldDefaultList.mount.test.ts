// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import * as VueRouter from 'vue-router';
const createFeStubRouter = (VueRouter as any).createFeStubRouter;
import { createPinia, setActivePinia } from 'pinia';
import { createI18n } from 'vue-i18n';
import { flushPromises, mountApp, stub } from '@/web/web/__tests__/mountApp';
import { replaceStoreFactory } from '@/web/web/stores/registry';
import FieldDefaultList from './FieldDefaultList.vue';

test('FieldDefaultList.vue wires useFieldDefaultListPage into ChoyPage', async () => {
  setActivePinia(createPinia());
  const restore = replaceStoreFactory('auth.FieldDefault', () => ({ id: 'auth-fd' }));
  const { router } = createFeStubRouter({
    route: {
      path: '/web/field-defaults/auth',
      fullPath: '/web/field-defaults/auth',
      params: {},
      meta: { fieldDefaultApplication: 'auth' },
    },
  });
  const i18n = createI18n({ legacy: false, locale: 'en', messages: { en: {} } });
  try {
    const mounted = mountApp(FieldDefaultList as any, {
      plugins: [router, i18n],
      stubs: {
        ChoyPage: stub('ChoyPage'),
        FieldDefaultListView: stub('FieldDefaultListView'),
      },
    });
    await flushPromises();
    expect(mounted.root).toBeTruthy();
    mounted.unmount();
  } finally {
    restore();
  }
});

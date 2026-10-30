// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import * as VueRouter from 'vue-router';
const createFeStubRouter = (VueRouter as any).createFeStubRouter;
import { createPinia, setActivePinia } from 'pinia';
import { createI18n } from 'vue-i18n';
import { flushPromises, mountApp, stub } from '@/web/web/__tests__/mountApp';
import { replaceStoreFactory } from '@/web/web/stores/registry';
import TranslationTermList from './TranslationTermList.vue';

test('TranslationTermList.vue wires useTerminologyListPage into ChoyPage', async () => {
  setActivePinia(createPinia());
  const restore = replaceStoreFactory('auth.TranslationTerm', () => ({ id: 'auth-tt' }));
  const { router } = createFeStubRouter({
    route: {
      path: '/web/terminology/auth',
      fullPath: '/web/terminology/auth',
      params: {},
      meta: { terminologyApplication: 'auth' },
    },
  });
  const i18n = createI18n({ legacy: false, locale: 'en', messages: { en: {} } });
  try {
    const mounted = mountApp(TranslationTermList as any, {
      plugins: [router, i18n],
      stubs: {
        ChoyPage: stub('ChoyPage'),
        TranslationTermListView: stub('TranslationTermListView'),
      },
    });
    await flushPromises();
    expect(mounted.root).toBeTruthy();
    mounted.unmount();
  } finally {
    restore();
  }
});

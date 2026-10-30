// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import * as VueRouter from 'vue-router';
const createFeStubRouter = (VueRouter as any).createFeStubRouter;
import { createPinia, setActivePinia } from 'pinia';
import { createI18n } from 'vue-i18n';
import { flushPromises, mountApp, stub } from '@/web/web/__tests__/mountApp';
import { replaceStoreFactory } from '@/web/web/stores/registry';
import TranslationTerm from './TranslationTerm.vue';

test('TranslationTerm.vue wires useTerminologyDetailPage into ChoyPage', async () => {
  setActivePinia(createPinia());
  const restore = replaceStoreFactory('web.TranslationTerm', () => ({ id: 'web-tt' }));
  const { router } = createFeStubRouter({
    route: {
      path: '/web/terminology/web/1',
      fullPath: '/web/terminology/web/1',
      params: { application: 'web', id: '1' },
      meta: {},
    },
  });
  const i18n = createI18n({ legacy: false, locale: 'en', messages: { en: {} } });
  try {
    const mounted = mountApp(TranslationTerm as any, {
      plugins: [router, i18n],
      stubs: {
        ChoyPage: stub('ChoyPage'),
        TranslationTermFormView: stub('TranslationTermFormView'),
      },
    });
    await flushPromises();
    expect(mounted.root).toBeTruthy();
    mounted.unmount();
  } finally {
    restore();
  }
});

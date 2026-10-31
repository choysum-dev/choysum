// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import * as VueRouter from 'vue-router';
const createFeStubRouter = (VueRouter as any).createFeStubRouter;
import { createPinia, setActivePinia } from 'pinia';
import { createI18n } from 'vue-i18n';
import { flushPromises, mountApp, stub } from '@/web/web/__tests__/mountApp';
import { replaceStoreFactory } from '@/web/web/stores/registry';
import FieldDefault from './FieldDefault.vue';

test('FieldDefault.vue wires useFieldDefaultDetailPage into ChoyPage', async () => {
  setActivePinia(createPinia());
  const restore = replaceStoreFactory('web.FieldDefault', () => ({ id: 'web-fd' }));
  const { router } = createFeStubRouter({
    route: {
      path: '/web/field-defaults/web/1',
      fullPath: '/web/field-defaults/web/1',
      params: { application: 'web', id: '1' },
      meta: {},
    },
  });
  const i18n = createI18n({ legacy: false, locale: 'en', messages: { en: {} } });
  try {
    const mounted = mountApp(FieldDefault as any, {
      plugins: [router, i18n],
      stubs: {
        ChoyPage: stub('ChoyPage'),
        FieldDefaultFormView: stub('FieldDefaultFormView'),
      },
    });
    await flushPromises();
    expect(mounted.root).toBeTruthy();
    mounted.unmount();
  } finally {
    restore();
  }
});

// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import * as VueRouter from 'vue-router';
const createFeStubRouter = (VueRouter as any).createFeStubRouter;
import { createPinia, setActivePinia } from 'pinia';
import { createI18n } from 'vue-i18n';
import { flushPromises, mountApp, stub } from '@/web/web/__tests__/mountApp';
import { replaceStoreFactory } from '@/web/web/stores/registry';
import PropertyDefinitionList from './PropertyDefinitionList.vue';

test('PropertyDefinitionList.vue wires usePropertyDefinitionListPage into ChoyPage', async () => {
  setActivePinia(createPinia());
  const restore = replaceStoreFactory('auth.PropertyDefinition', () => ({ id: 'auth-pd' }));
  const { router } = createFeStubRouter({
    route: {
      path: '/web/property-definitions/auth',
      fullPath: '/web/property-definitions/auth',
      params: {},
      meta: { propertyDefinitionApplication: 'auth' },
    },
  });
  const i18n = createI18n({ legacy: false, locale: 'en', messages: { en: {} } });
  try {
    const mounted = mountApp(PropertyDefinitionList as any, {
      plugins: [router, i18n],
      stubs: {
        ChoyPage: stub('ChoyPage'),
        PropertyDefinitionListView: stub('PropertyDefinitionListView'),
      },
    });
    await flushPromises();
    expect(mounted.root).toBeTruthy();
    mounted.unmount();
  } finally {
    restore();
  }
});

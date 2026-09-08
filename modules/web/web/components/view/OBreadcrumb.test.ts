// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { nextTick } from 'vue';
import { createI18n } from 'vue-i18n';
import { createPinia, setActivePinia } from 'pinia';
import { createRouter, createMemoryHistory } from 'vue-router';

import { createTermReference } from '@/core/service/i18n';
import { projectTerminologyMessages } from '../../i18n/terminology';
import { useBreadcrumbStore } from '../../stores/breadcrumbStore';
import { mountApp } from '@/web/web/__tests__/mountApp';
import OBreadcrumb from './OBreadcrumb.vue';

describe('OBreadcrumb terminology display', () => {
  test('translates term references directly in the template and preserves plain strings', async () => {
    const pinia = createPinia();
    setActivePinia(pinia);
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/', component: { template: '<div />' } },
        { path: '/settings', component: { template: '<div />' } },
        { path: '/legacy', component: { template: '<div />' } },
      ],
    });

    const i18n = createI18n({
      legacy: false,
      locale: 'en',
      missingWarn: false,
      fallbackWarn: false,
      messages: {
        en: {},
        'zh-CN': projectTerminologyMessages({
          base: { 'base.route.settings': { Settings: '设置' } },
        }),
      },
    });

    const store = useBreadcrumbStore();
    store.clearBreadcrumb();
    store.breadcrumbStack.splice(
      0,
      store.breadcrumbStack.length,
      {
        title: 'Settings',
        titleText: createTermReference('base', 'Settings', { scope: 'base.route.settings' }),
        path: '/settings',
        clickable: false,
        timestamp: Date.now(),
      },
      {
        title: 'Legacy page',
        path: '/legacy',
        clickable: false,
        timestamp: Date.now(),
      }
    );

    const { unmount, text } = mountApp(OBreadcrumb as any, {
      plugins: [pinia, router, i18n],
    });

    expect(text()).toContain('Settings');
    expect(text()).toContain('Legacy page');

    i18n.global.locale.value = 'zh-CN';
    await nextTick();
    expect(text()).toContain('设置');
    expect(text()).toContain('Legacy page');
    unmount();
  });
});

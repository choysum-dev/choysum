// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { nextTick } from 'vue';
import { createI18n } from 'vue-i18n';
import { createPinia, setActivePinia } from 'pinia';
import { buildPageMountGlobal } from '@choysum/page-mount';

import { createTermReference } from '@/core/service/i18n';
import { projectTerminologyMessages } from '@/web/web/i18n/terminology';
import { useBreadcrumbStore } from '@/web/web/stores/breadcrumbStore';
import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import OBreadcrumb from './OBreadcrumb.vue';

describe('OBreadcrumb terminology display', () => {
  test('translates term references directly in the template and preserves plain strings', async () => {
    const { plugins } = buildPageMountGlobal();
    const pinia = plugins[0] as ReturnType<typeof createPinia>;
    setActivePinia(pinia);

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

    const { unmount, text } = mountApp(OBreadcrumb as any, {
      plugins: [...plugins, i18n],
    });
    await flushPromises();

    const bc = useBreadcrumbStore();
    bc.breadcrumbStack.splice(0, bc.breadcrumbStack.length);
    bc.breadcrumbStack.push(
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
    await nextTick();

    expect(text()).toContain('Settings');
    expect(text()).toContain('Legacy page');

    i18n.global.locale.value = 'zh-CN';
    await nextTick();
    expect(text()).toContain('设置');
    expect(text()).toContain('Legacy page');
    unmount();
  });
});

// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed } from 'vue';
import { createI18n } from 'vue-i18n';
import { createTermReference } from '@/core/service/i18n';
import { projectTerminologyMessages } from '../i18n/terminology';
import { resolveDocumentTitle } from './documentTitle';

describe('resolveDocumentTitle', () => {
  test('formats a plain pageTitle with the app name', () => {
    expect(
      resolveDocumentTitle(
        {
          path: '/home',
          meta: { pageTitle: 'Home' },
        } as any,
        undefined,
        'Choysum'
      )
    ).toBe('Home - Choysum');
  });

  test('updates a term reference title when locale changes', async () => {
    const i18n = createI18n({
      legacy: false,
      locale: 'en',
      missingWarn: false,
      fallbackWarn: false,
      messages: {
        en: {},
        'zh-CN': projectTerminologyMessages({
          base: { 'base.route.users': { Users: '用户' } },
        }),
      },
    });

    const route = {
      path: '/home',
      meta: {
        pageTitle: 'Users',
        pageTitleText: createTermReference('base', 'Users', { scope: 'base.route.users' }),
      },
    } as any;

    const title = computed(() => resolveDocumentTitle(route, i18n.global as any, 'Choysum'));
    expect(title.value).toBe('Users - Choysum');

    i18n.global.locale.value = 'zh-CN';
    expect(title.value).toBe('用户 - Choysum');
  });
});

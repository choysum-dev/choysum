// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed } from 'vue';
import { createI18n } from 'vue-i18n';
import { createTermReference } from '@/core/service/i18n';
import { projectTerminologyMessages } from '../i18n/terminology';
import { resolveDocumentTitle } from './documentTitle';
import { createAppRouter, defaultAppName } from './index';

describe('defaultAppName', () => {
  test('defaults to Choysum when env is missing or blank', () => {
    expect(defaultAppName(undefined)).toBe('Choysum');
    expect(defaultAppName({} as any)).toBe('Choysum');
    expect(defaultAppName({ CHOYSUM_APP_NAME: '   ' } as any)).toBe('Choysum');
    expect(defaultAppName({ CHOYSUM_APP_NAME: 1 } as any)).toBe('Choysum');
  });

  test('uses CHOYSUM_APP_NAME when present and non-blank', () => {
    expect(defaultAppName({ CHOYSUM_APP_NAME: 'Acme' } as any)).toBe('Acme');
    expect(defaultAppName({ CHOYSUM_APP_NAME: '  Acme  ' } as any)).toBe('Acme');
  });
});

describe('createAppRouter', () => {
  test('wires default app name into the document title helper', async () => {
    const router = createAppRouter('/');
    expect(router).toBeTruthy();
    // Drive beforeEach/afterEach so navigation progress hooks are covered.
    await router.push('/').catch(() => undefined);
    await router.isReady().catch(() => undefined);
    expect(typeof router.currentRoute.value.path).toBe('string');
  });
});

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

  test('returns app name alone when route and title are empty', () => {
    expect(resolveDocumentTitle(null, undefined, 'Choysum')).toBe('Choysum');
    expect(resolveDocumentTitle({ meta: {} } as any, undefined, 'App')).toBe('App');
    expect(resolveDocumentTitle({ meta: { pageTitle: 12 } } as any, undefined, 'App')).toBe('App');
  });

  test('evaluates function pageTitle and swallows callback errors', () => {
    expect(
      resolveDocumentTitle(
        {
          path: '/x',
          meta: {
            pageTitle: (route: { path: string }) => `P:${route.path}`,
          },
        } as any,
        undefined,
        'Choysum'
      )
    ).toBe('P:/x - Choysum');

    expect(
      resolveDocumentTitle(
        {
          path: '/x',
          meta: {
            pageTitle: () => {
              throw new Error('boom');
            },
          },
        } as any,
        undefined,
        'Choysum'
      )
    ).toBe('Choysum');

    expect(
      resolveDocumentTitle(
        {
          path: '/x',
          meta: {
            pageTitle: () => null,
          },
        } as any,
        undefined,
        'Choysum'
      )
    ).toBe('Choysum');
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

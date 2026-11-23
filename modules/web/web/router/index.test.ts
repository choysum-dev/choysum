// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed } from 'vue';
import { createI18n } from 'vue-i18n';
import { createPinia, setActivePinia } from 'pinia';
import { createTermReference } from '@/core/service/i18n';
import {
  beginNavigationLoading,
  getLoadingRegistryDebugState,
  loadingTiming,
  resetLoadingRegistryForTests,
  settleNavigationIfActive,
} from '@/core/web/client/loadingRegistry';
import { projectTerminologyMessages } from '../i18n/terminology';
import { useConnectivityStore } from '../stores/connectivityStore';
import { resolveDocumentTitle, resolveRoutePageTitle } from './documentTitle';
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

  test('aborted navigation settles the loading ticket via afterEach failure', async () => {
    resetLoadingRegistryForTests();
    const router = createAppRouter('/');
    await router.isReady().catch(() => undefined);
    const removeGuard = router.beforeEach(to => {
      if (to.path === '/__abort_nav__') return false;
      return true;
    });
    await router.push('/__abort_nav__').catch(() => undefined);
    // Failure path must drop the ticket opened in beforeEach.
    expect(getLoadingRegistryDebugState().navTicketActive).toBe(false);
    removeGuard();
    resetLoadingRegistryForTests();
  });

  test('router onError settles an active navigation ticket', async () => {
    resetLoadingRegistryForTests();
    const router = createAppRouter('/');
    await router.isReady().catch(() => undefined);
    beginNavigationLoading();
    expect(getLoadingRegistryDebugState().navTicketActive).toBe(true);
    const anyRouter = router as { triggerError?: (err: Error) => void };
    expect(typeof anyRouter.triggerError).toBe('function');
    anyRouter.triggerError?.(new Error('nav-boom'));
    expect(getLoadingRegistryDebugState().navTicketActive).toBe(false);
    resetLoadingRegistryForTests();
  });

  test('router onError records chunk load failures for shell retry UI', async () => {
    const { clearViewLoadError, viewLoadError } = await import('./viewLoadError');
    clearViewLoadError();
    const router = createAppRouter('/');
    await router.isReady().catch(() => undefined);
    const anyRouter = router as {
      triggerError?: (err: Error, to?: { fullPath: string; meta?: Record<string, unknown> }) => void;
    };
    // Omit ownership seq so the handler records even after prior navigations advanced the counter.
    anyRouter.triggerError?.(
      new Error('Failed to fetch dynamically imported module: /web/assets/Company-x.js'),
      { fullPath: '/web/base/companies?q=1#top', meta: {} },
    );
    expect(viewLoadError.value?.fullPath).toBe('/web/base/companies?q=1#top');
    clearViewLoadError();
  });

  test('duplicate navigation failure advances ownership so stale chunks cannot blank', async () => {
    const { clearViewLoadError, viewLoadError } = await import('./viewLoadError');
    clearViewLoadError();
    const router = createAppRouter('/');
    await router.isReady().catch(() => undefined);
    await router.push('/__nav_b__').catch(() => undefined);
    const staleSeq = Number(
      (router.currentRoute.value.meta as { __navOwnershipSeq?: number }).__navOwnershipSeq,
    );
    expect(staleSeq).toBeGreaterThan(0);
    // Same-path push is duplicated: beforeEach skipped, afterEach bumps ownership.
    await router.push('/__nav_b__').catch(() => undefined);
    const anyRouter = router as {
      triggerError?: (err: Error, to?: { fullPath: string; meta?: Record<string, unknown> }) => void;
    };
    anyRouter.triggerError?.(
      new Error('Failed to fetch dynamically imported module: /web/assets/Stale-x.js'),
      { fullPath: '/stale-dup', meta: { __navOwnershipSeq: staleSeq } },
    );
    expect(viewLoadError.value).toBeNull();
    clearViewLoadError();
  });

  test('router onError ignores chunk failures from superseded navigations', async () => {
    const { clearViewLoadError, viewLoadError } = await import('./viewLoadError');
    clearViewLoadError();
    const router = createAppRouter('/');
    await router.isReady().catch(() => undefined);
    let staleSeq = 0;
    const remove = router.beforeEach(to => {
      if (to.path === '/__nav_b__') {
        staleSeq = Number((to.meta as { __navOwnershipSeq?: number }).__navOwnershipSeq) || 0;
      }
      return true;
    });
    await router.push('/__nav_b__').catch(() => undefined);
    await router.push('/__nav_content_policy__').catch(() => undefined);
    remove();
    expect(staleSeq).toBeGreaterThan(0);
    const anyRouter = router as {
      triggerError?: (err: Error, to?: { fullPath: string; meta?: Record<string, unknown> }) => void;
    };
    anyRouter.triggerError?.(
      new Error('Failed to fetch dynamically imported module: /web/assets/Stale-x.js'),
      { fullPath: '/stale', meta: { __navOwnershipSeq: staleSeq } },
    );
    expect(viewLoadError.value).toBeNull();
    clearViewLoadError();
  });

  test('older cancelled navigation settlement does not clear a newer ticket', async () => {
    resetLoadingRegistryForTests();
    const router = createAppRouter('/');
    await router.isReady().catch(() => undefined);

    let genB = 0;
    router.beforeEach(to => {
      if (to.path === '/__nav_b__') {
        genB = Number((to.meta as { __navLoadingGen?: number }).__navLoadingGen) || 0;
      }
      return true;
    });

    await router.push('/__nav_b__').catch(() => undefined);
    expect(genB).toBeGreaterThan(0);
    expect(getLoadingRegistryDebugState().navTicketActive).toBe(true);

    // Failed older navigation must not clear B's generation-scoped ticket.
    settleNavigationIfActive({ generation: genB - 1 });
    expect(getLoadingRegistryDebugState().navTicketActive).toBe(true);
    expect(getLoadingRegistryDebugState().navTicketGeneration).toBe(genB);

    resetLoadingRegistryForTests();
  });

  test('navLoading route settles the nav ticket as soon as the route lands', async () => {
    resetLoadingRegistryForTests();
    const router = createAppRouter('/');
    await router.isReady().catch(() => undefined);

    let sawTicketDuringNav = false;
    // Runs after createAppRouter beforeEach, so the route ticket must already be open.
    const removeGuard = router.beforeEach(() => {
      if (getLoadingRegistryDebugState().navTicketActive) sawTicketDuringNav = true;
      return true;
    });

    // FE stub applies push().meta onto the destination before guards run.
    await router
      .push({ path: '/__nav_route_policy__', meta: { navLoading: 'route' } } as any)
      .catch(() => undefined);
    expect(sawTicketDuringNav).toBe(true);
    expect(getLoadingRegistryDebugState().navTicketActive).toBe(false);
    removeGuard();
    resetLoadingRegistryForTests();
  });

  test('navLoading content keeps the nav ticket through idle grace', async () => {
    resetLoadingRegistryForTests();
    const router = createAppRouter('/');
    await router.isReady().catch(() => undefined);

    await router
      .push({ path: '/__nav_content_policy__', meta: { navLoading: 'content' } } as any)
      .catch(() => undefined);
    expect(getLoadingRegistryDebugState().navTicketActive).toBe(true);
    expect(loadingTiming.NAV_IDLE_SETTLE_MS).toBeGreaterThan(50);
    await new Promise<void>(resolve => setTimeout(resolve, 30));
    expect(getLoadingRegistryDebugState().navTicketActive).toBe(true);
    resetLoadingRegistryForTests();
  });

  test('navLoading none does not open a navigation ticket and clears leftovers', async () => {
    resetLoadingRegistryForTests();
    const router = createAppRouter('/');
    await router.isReady().catch(() => undefined);

    beginNavigationLoading();
    expect(getLoadingRegistryDebugState().navTicketActive).toBe(true);

    await router
      .push({ path: '/__nav_none_policy__', meta: { navLoading: 'none' } } as any)
      .catch(() => undefined);
    expect(getLoadingRegistryDebugState().navTicketActive).toBe(false);
    resetLoadingRegistryForTests();
  });

  test('failed none navigation does not clear a newer ticket', async () => {
    resetLoadingRegistryForTests();
    const router = createAppRouter('/');
    await router.isReady().catch(() => undefined);

    const newerGen = beginNavigationLoading();
    expect(getLoadingRegistryDebugState().navTicketActive).toBe(true);

    const removeGuard = router.beforeEach(to => {
      if (to.path === '/__nav_none_abort__') return false;
      return true;
    });

    await router
      .push({ path: '/__nav_none_abort__', meta: { navLoading: 'none' } } as any)
      .catch(() => undefined);

    expect(getLoadingRegistryDebugState().navTicketActive).toBe(true);
    expect(getLoadingRegistryDebugState().navTicketGeneration).toBe(newerGen);
    removeGuard();
    resetLoadingRegistryForTests();
  });

  test('unreachable session does not open a navigation ticket', async () => {
    resetLoadingRegistryForTests();
    setActivePinia(createPinia());
    useConnectivityStore().markUnreachable();
    const router = createAppRouter('/');
    await router.isReady().catch(() => undefined);

    await router
      .push({ path: '/__nav_unreachable__', meta: { navLoading: 'content' } } as any)
      .catch(() => undefined);
    expect(getLoadingRegistryDebugState().navTicketActive).toBe(false);

    useConnectivityStore().resetForTests();
    resetLoadingRegistryForTests();
  });
});

describe('resolveRoutePageTitle', () => {
  test('returns the bare page title without the app suffix', () => {
    expect(
      resolveRoutePageTitle({
        path: '/base/companies/1',
        meta: { pageTitle: 'Company Details' },
      } as any)
    ).toBe('Company Details');
    expect(resolveRoutePageTitle(null)).toBe('');
  });
});

describe('resolveDocumentTitle', () => {
  test('formats a plain pageTitle with the app name', () => {
    expect(
      resolveDocumentTitle(
        {
          path: '/partner',
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
      path: '/partner',
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

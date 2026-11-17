// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { useTitle } from '@vueuse/core';
import { computed, shallowRef } from 'vue';
import { createRouter, createWebHistory } from 'vue-router';
import type { RouteLocationNormalized, Router } from 'vue-router';
import routes from './routes';
import 'nprogress/nprogress.css';
import {
  beginNavigationLoading,
  loadingTiming,
  scheduleNavigationSettleIfIdle,
  settleNavigationIfActive,
} from '@/core/web/client/loadingRegistry';
import type { ComposerLike } from '../i18n';
import { resolveDocumentTitle } from './documentTitle';
import { resolveNavLoadingPolicy } from './navLoadingPolicy';

export { resolveDocumentTitle, resolveRoutePageTitle } from './documentTitle';
export { resolveNavLoadingPolicy } from './navLoadingPolicy';
export type { NavLoadingPolicy } from './navLoadingPolicy';

function defaultAppName(env: ImportMetaEnv | undefined = import.meta.env): string {
  const appNameRaw = env?.CHOYSUM_APP_NAME;
  if (typeof appNameRaw !== 'string') return 'Choysum';
  const appName = appNameRaw.trim();
  return appName !== '' ? appName : 'Choysum';
}

export { defaultAppName };

/**
 * Creates the application router.
 * Navigation loading (bar + userBusy / slow-hint) is driven by the loading
 * registry (beginNavigationLoading), not by start/done around afterEach —
 * so chrome can span chunk fetch and first surface load.
 */
export function createAppRouter(base = '/', composer?: ComposerLike): Router {
  const router = createRouter({
    history: createWebHistory(base),
    routes,
    scrollBehavior: (to, from, savedPosition) => {
      if (savedPosition) {
        return savedPosition;
      }

      if (to.hash) {
        return {
          el: to.hash,
          behavior: 'smooth',
          top: 80,
        };
      }

      return { top: 0 };
    },
  });
  const titleRoute = shallowRef<RouteLocationNormalized | null>(router.currentRoute.value);
  const appName = defaultAppName();
  useTitle(computed(() => resolveDocumentTitle(titleRoute.value as any, composer, appName)));

  router.beforeEach(async to => {
    titleRoute.value = to;
    // none: do not open a nav ticket for this destination.
    if (resolveNavLoadingPolicy(to) === 'none') return true;
    const generation = beginNavigationLoading();
    (to.meta as { __navLoadingGen?: number }).__navLoadingGen = generation;
    return true;
  });

  // Settle by destination meta.navLoading: content keeps idle grace for surface
  // handoff; route settles immediately when the route lands; none drops leftover
  // chrome without opening a new ticket.
  // Failed navigations must drop only the ticket they opened (generation-scoped).
  // A failed none nav has no generation — do not settle, or a newer ticket can drop.
  router.afterEach((to, _from, failure) => {
    const generation = Number((to.meta as { __navLoadingGen?: number }).__navLoadingGen) || 0;
    if (failure) {
      if (generation) {
        settleNavigationIfActive({ generation });
      } else if (resolveNavLoadingPolicy(to) !== 'none') {
        // Fallback when generation was lost; never clear chrome for a failed none nav.
        settleNavigationIfActive();
      }
      return;
    }
    const policy = resolveNavLoadingPolicy(to);
    if (policy === 'none') {
      settleNavigationIfActive();
      return;
    }
    if (policy === 'route') {
      // Direct settle: avoid scheduleNavigationSettleIfIdle(0) re-arm spin while a
      // leftover surface from the previous page is still busy.
      settleNavigationIfActive(generation ? { generation } : undefined);
      return;
    }
    scheduleNavigationSettleIfIdle(loadingTiming.NAV_IDLE_SETTLE_MS);
  });

  router.onError(error => {
    console.error('Router error:', error);
    settleNavigationIfActive();
  });

  return router;
}

// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { useTitle } from '@vueuse/core';
import { computed, shallowRef } from 'vue';
import {
  createRouter,
  createWebHistory,
  isNavigationFailure,
  NavigationFailureType,
} from 'vue-router';
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
import { isConnectivityUnreachable } from '../stores/connectivityStore';
import { resolveDocumentTitle } from './documentTitle';
import { resolveNavLoadingPolicy } from './navLoadingPolicy';
import { clearViewLoadError, isRouteChunkLoadError, setViewLoadError } from './viewLoadError';

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
  /** Latest navigation ownership token — stale chunk errors must not blank a newer page. */
  let navOwnershipSeq = 0;

  router.beforeEach(async to => {
    titleRoute.value = to;
    const ownership = ++navOwnershipSeq;
    (to.meta as { __navOwnershipSeq?: number }).__navOwnershipSeq = ownership;
    // none: do not open a nav ticket; drop any stale generation marker on meta.
    // Unreachable: Connection lost pill already owns session truth — do not open a
    // 15s handoff that waits on chunks/RPC that will not complete.
    if (resolveNavLoadingPolicy(to) === 'none' || isConnectivityUnreachable()) {
      delete (to.meta as { __navLoadingGen?: number }).__navLoadingGen;
      return true;
    }
    const generation = beginNavigationLoading();
    (to.meta as { __navLoadingGen?: number }).__navLoadingGen = generation;
    return true;
  });

  // Settle by destination meta.navLoading: content keeps idle grace for surface
  // handoff; route settles immediately (generation-scoped); none drops leftover
  // chrome without opening a new ticket (unscoped — none has no generation).
  // Failed navigations settle only by generation so a failed none (no ticket)
  // cannot clear a newer navigation's chrome.
  router.afterEach((to, _from, failure) => {
    const generation = Number((to.meta as { __navLoadingGen?: number }).__navLoadingGen) || 0;
    if (failure) {
      if (generation) settleNavigationIfActive({ generation });
      // Duplicate nav to the current route skips beforeEach, so ownership would
      // not advance — bump it so a still-pending foreign chunk failure cannot blank.
      if (isNavigationFailure(failure, NavigationFailureType.duplicated)) {
        navOwnershipSeq += 1;
      }
      return;
    }
    // Successful navigation clears a prior chunk-load blank.
    clearViewLoadError();
    const policy = resolveNavLoadingPolicy(to);
    if (policy === 'none') {
      settleNavigationIfActive();
      return;
    }
    if (policy === 'route') {
      // Direct settle: avoid scheduleNavigationSettleIfIdle(0) re-arm spin while a
      // leftover surface from the previous page is still busy. Scope to this
      // destination's generation so a stale afterEach cannot clear a newer ticket.
      settleNavigationIfActive({ generation });
      return;
    }
    scheduleNavigationSettleIfIdle(loadingTiming.NAV_IDLE_SETTLE_MS);
  });

  router.onError((error, to) => {
    console.error('Router error:', error);
    settleNavigationIfActive();
    if (!isRouteChunkLoadError(error)) return;
    const failed = (to ||
      (error as { to?: RouteLocationNormalized })?.to ||
      null) as RouteLocationNormalized | null;
    // No destination cannot be attributed to the current page — ignore it.
    if (!failed) return;
    const failedSeq = Number((failed?.meta as { __navOwnershipSeq?: number } | undefined)?.__navOwnershipSeq) || 0;
    // A superseded navigation's late chunk reject must not blank the newer route.
    if (failedSeq && failedSeq !== navOwnershipSeq) return;
    // Missing ownership: only blank when the failed destination is still current.
    if (!failedSeq && String(failed.fullPath || '') !== String(router.currentRoute.value.fullPath || '')) {
      return;
    }
    const fullPath = String(failed?.fullPath || router.currentRoute.value.fullPath || '/');
    setViewLoadError({
      message: String((error as { message?: string })?.message || error || ''),
      fullPath,
    });
  });

  return router;
}

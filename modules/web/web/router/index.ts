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
  scheduleNavigationSettleIfIdle,
  settleNavigationIfActive,
} from '@/core/web/client/loadingRegistry';
import type { ComposerLike } from '../i18n';
import { resolveDocumentTitle } from './documentTitle';

export { resolveDocumentTitle, resolveRoutePageTitle } from './documentTitle';

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
    beginNavigationLoading();

    try {
      titleRoute.value = to;

      return true;
    } catch (error) {
      console.error('Navigation error:', error);
      settleNavigationIfActive();
      return to.path === '/error/500' ? true : '/error/500';
    }
  });

  // Guest pages never open surface: settle the nav bar shortly if still idle.
  // List/Form that start surface within the grace keep the ticket until surface ends.
  router.afterEach(() => {
    scheduleNavigationSettleIfIdle();
  });

  router.onError(error => {
    console.error('Router error:', error);
  });

  return router;
}

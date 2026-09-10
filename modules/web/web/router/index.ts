// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { useTitle } from '@vueuse/core';
import { computed, shallowRef } from 'vue';
import { createRouter, createWebHistory } from 'vue-router';
import type { RouteLocationNormalized, Router } from 'vue-router';
import routes from './routes';
import NProgress from 'nprogress';
import 'nprogress/nprogress.css';
import type { ComposerLike } from '../i18n';
import { resolveDocumentTitle } from './documentTitle';

export { resolveDocumentTitle } from './documentTitle';

// Configure navigation progress feedback.
NProgress.configure({ showSpinner: false });

function defaultAppName(env: ImportMetaEnv | undefined = import.meta.env): string {
  const appNameRaw = env?.CHOYSUM_APP_NAME;
  if (typeof appNameRaw !== 'string') return 'Choysum';
  const appName = appNameRaw.trim();
  return appName !== '' ? appName : 'Choysum';
}

export { defaultAppName };

/**
 * Creates the application router.
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
    NProgress.start();

    try {
      titleRoute.value = to;

      return true;
    } catch (error) {
      console.error('Navigation error:', error);
      NProgress.done();
      return to.path === '/error/500' ? true : '/error/500';
    }
  });

  router.afterEach((_to, _from) => {
    NProgress.done();
  });

  router.onError(error => {
    console.error('Router error:', error);
  });

  return router;
}

// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { useTitle } from '@vueuse/core';
import { computed, shallowRef } from 'vue';
import { createRouter, createWebHistory } from 'vue-router';
import type { RouteLocationNormalized, Router } from 'vue-router';
import routes from './routes';
import NProgress from 'nprogress';
import 'nprogress/nprogress.css';
import { translateTerm, type ComposerLike } from '../i18n';
import { isTermReference } from '@/core/service/i18n';

// Configure navigation progress feedback.
NProgress.configure({ showSpinner: false });

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
  const titleRoute = shallowRef<RouteLocationNormalized | null>(router.currentRoute?.value ?? null);
  const appNameRaw = import.meta.env?.CHOYSUM_APP_NAME;
  const appName = typeof appNameRaw === 'string' && appNameRaw.trim() !== '' ? appNameRaw : 'Choysum';
  useTitle(
    computed(() => {
      const route = titleRoute.value;
      const pageTitle = route?.meta?.pageTitle;
      const fallback = typeof pageTitle === 'function'
        ? String(pageTitle(route!))
        : typeof pageTitle === 'string'
          ? pageTitle
          : '';
      const reference = isTermReference(route?.meta?.pageTitleText)
        ? route.meta.pageTitleText
        : undefined;
      const title = translateTerm(composer, reference, fallback);
      return title ? `${title} - ${appName}` : appName;
    })
  );

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

  router.afterEach((to, from) => {
    NProgress.done();
  });

  router.onError(error => {
    console.error('Router error:', error);
  });

  return router;
}

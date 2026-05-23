// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { useTitle } from '@vueuse/core';
import { createRouter, createWebHistory } from 'vue-router';
import type { Router } from 'vue-router';
import routes from './routes';
import NProgress from 'nprogress';
import 'nprogress/nprogress.css';

// Configure navigation progress feedback.
NProgress.configure({ showSpinner: false });

/**
 * Creates the application router.
 */
export function createAppRouter(base = '/'): Router {
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

  router.beforeEach(async (to, from, next) => {
    NProgress.start();

    try {
      let title = '';
      const appName = import.meta.env.CHOYSUM_APP_NAME;

      if (to.meta?.pageTitle) {
        title = typeof to.meta.pageTitle === 'function' ? to.meta.pageTitle(to) : (to.meta.pageTitle as string);
      }

      useTitle(title ? `${title} - ${appName}` : appName);

      next();
    } catch (error) {
      console.error('Navigation error:', error);
      NProgress.done();
      next('/error/500');
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

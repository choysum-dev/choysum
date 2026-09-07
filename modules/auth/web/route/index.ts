// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { ChoysumWebApp } from '@/core/web/application';
import { authGuard, permissionGuard } from './guard';
import { authRoutes, appRoutes } from './routes';
import type { RouteLocationNormalized, NavigationGuardNext, RouteRecordRaw } from 'vue-router';
import { nextTick } from 'vue';

/**
 * Register auth routes and guards on the application router.
 */
export function setupRouter(app: ChoysumWebApp): void {
  const router = app.router;
  // Register auth-only routes under the main layout.
  for (const route of authRoutes) {
    router.addRoute('Layout', route);
  }

  for (const route of appRoutes) {
    router.addRoute('AppLayout', route);
  }

  // Install auth before permission so redirects resolve in the expected order.
  // Two-arg wrappers keep vue-router from treating AuthGuardDeps as `next`
  // (guards declare an optional test deps object as a third parameter).
  router.beforeEach((to, from) => authGuard(to, from));
  router.beforeEach((to, from) => permissionGuard(to, from));
}

// Re-export auth routes and guards.
export { authRoutes } from './routes';
export { authGuard, permissionGuard } from './guard';

// Re-export route helper types.
export type { RouteLocationNormalized, NavigationGuardNext, RouteRecordRaw };

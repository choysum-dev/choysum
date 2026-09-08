// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Shared FE unit helpers for mounting real business pages under choysumMount + QJS.
 * Relies on host_bundle default stubs (element-plus, icons, vue-router, OPage).
 */
import { createPinia } from 'pinia';
import { createFeStubRouter } from 'vue-router';

export type PageMountOverrides = {
  route?: Record<string, unknown>;
  router?: Record<string, unknown>;
};

/**
 * Build choysumMount `global` options for a product page (Pinia + stub router).
 */
export function buildPageMountGlobal(overrides: PageMountOverrides = {}) {
  const { router } = createFeStubRouter({
    route: Object.assign({ path: '/login', fullPath: '/login', query: {} }, overrides.route || {}),
    router: overrides.router || {},
  });
  return {
    plugins: [createPinia(), router],
  };
}

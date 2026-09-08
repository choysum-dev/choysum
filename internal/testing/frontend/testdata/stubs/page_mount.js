// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

/**
 * Shared FE unit helper: choysumMount `global` options for product pages under QJS.
 * Import as `@choysum/page-mount` (aliased by host_bundle FE stubs).
 */
import { createPinia } from 'pinia';
import { createFeStubRouter } from 'vue-router';

/**
 * @param {{ route?: Record<string, unknown>, router?: Record<string, unknown> }} [overrides]
 */
export function buildPageMountGlobal(overrides) {
  overrides = overrides || {};
  var routeOverrides = Object.assign({}, overrides.route || {});
  if (routeOverrides.fullPath == null && routeOverrides.path != null) {
    routeOverrides.fullPath = routeOverrides.path;
  }
  var { router } = createFeStubRouter({
    route: Object.assign({ path: '/', fullPath: '/', query: {} }, routeOverrides),
    router: overrides.router || {},
  });
  return {
    plugins: [createPinia(), router],
  };
}

export default { buildPageMountGlobal: buildPageMountGlobal };

// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createPinia } from 'pinia';
import { createFeStubRouter } from 'vue-router';

export type PageMountOverrides = {
  route?: Record<string, unknown>;
  router?: Record<string, unknown>;
};

export function buildPageMountGlobal(overrides: PageMountOverrides = {}) {
  const { router } = createFeStubRouter({
    route: Object.assign({ path: '/', fullPath: '/', query: {} }, overrides.route || {}),
    router: overrides.router || {},
  });
  return {
    plugins: [createPinia(), router],
  };
}

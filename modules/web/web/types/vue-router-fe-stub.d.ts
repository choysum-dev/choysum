// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Augments vue-router with createFeStubRouter from the FE unit stub
 * (internal/testing/frontend/testdata/stubs/vue_router.js).
 */
export {};

declare module 'vue-router' {
  export function createFeStubRouter(overrides?: {
    route?: Record<string, unknown>;
    [key: string]: unknown;
  }): {
    router: any;
    route: any;
  };
}

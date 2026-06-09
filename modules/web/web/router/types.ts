// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { RouteLocationNormalized } from 'vue-router';

declare module 'vue-router' {
  interface RouteMeta {
    /**
     * Whether the route should be hidden from breadcrumbs.
     * @default false
     */
    hiddenInBreadcrumb?: boolean;

    /**
     * Whether the component instance should be cached with keep-alive.
     * @default false
     */
    keepAlive?: boolean;

    /**
     * Browser title override.
     * Static titles should prefer defineRoute.title; use pageTitle for dynamic titles or non-defineRoute routes.
     */
    pageTitle?: string | ((route: RouteLocationNormalized) => string);
  }
}

// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { RouteLocationNormalized } from 'vue-router';
import type { TermReference } from '../../../core/service/i18n';

declare module 'vue-router' {
  interface RouteMeta {
    /**
     * Whether the route should be hidden from breadcrumbs.
     * @default false
     */
    hiddenInBreadcrumb?: boolean;

    /**
     * Whether the component instance should be cached with keep-alive.
     * Explicit true/false wins. When omitted: defineRoute defaults true for
     * *_list / *_kanban / *_chart ids (other browse surfaces set keepAlive
     * explicitly); otherwise pageKind === 'collection' also caches. viewPaths
     * does not imply KeepAlive. Cached instances live only within the current
     * leaf menu + company scope (App shell keepAliveScopeKey). Guest shell
     * ignores this flag.
     * @default false (unless defineRoute collection default or pageKind)
     */
    keepAlive?: boolean;

    /**
     * Page role for shell policies. collection → KeepAlive when keepAlive omitted.
     * record / create / other → not cached unless keepAlive is true.
     */
    pageKind?: 'collection' | 'record' | 'create' | string;

    /**
     * Browser title override.
     * Static titles should prefer defineRoute.title; use pageTitle for dynamic titles or non-defineRoute routes.
     */
    pageTitle?: string | ((route: RouteLocationNormalized) => string);

    /**
     * Serializable term reference for the static page title.
     * pageTitle remains the English fallback for legacy consumers.
     */
    pageTitleText?: TermReference;

    /**
     * Multi-view management: active kind for this route (list / kanban / chart / …).
     * Consumed by useChoyViewRouteSwitch.
     */
    viewKind?: string;

    /** Default store appliedGroups for this management view. */
    defaultGroups?: string[];

    /** Absolute paths keyed by view kind for ViewSwitcher sibling navigation. */
    viewPaths?: Record<string, string>;

    /**
     * Keep nav chrome until content owns it, until the route lands, or not at all.
     * Omit = content. Orthogonal to runWithLoading layer (surface/control/chrome).
     */
    navLoading?: 'content' | 'route' | 'none';
  }
}

// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { RouteRecordRaw } from 'vue-router';
import { defineRoute } from '@/core/web/resource';
import { createTranslate } from '@/web/web/i18n';
// Side-effect: register web.action.user_filter_* UI resources.
import '../views/user_filter_actions';

const { _lt } = createTranslate('web', { scope: 'web/route/userFilterRoutes' });

/**
 * Settings → web → Saved Filters (UserFilter) admin ledger.
 * Owner Application is always web; no per-app :application segment.
 * OSearch still labels the end-user facet as Favorites.
 *
 * actions must be string-literal arrays (UI_DECL_008); ids match
 * defineModelActions('web.UserFilter').
 */
export const userFilterListComponent = () => import('../pages/UserFilterList.vue');

const userFilterFormComponent = () => import('../pages/UserFilter.vue');

export const userFilterRoutes: RouteRecordRaw[] = [
  defineRoute('web.route.user_filter_list', {
    sequence: 70,
    title: _lt('Saved Filters'),
    path: 'web/user-filters',
    name: 'UserFilterList_web',
    component: userFilterListComponent,
    actions: [
      'web.action.user_filter_create',
      'web.action.user_filter_edit',
      'web.action.user_filter_delete',
      'web.action.user_filter_copy',
    ],
    defaultRoles: ['sys.admin'],
    meta: { requiresAuth: true },
  }),
  defineRoute('web.route.user_filter_create', {
    sequence: 71,
    title: _lt('New Saved Filter'),
    path: 'web/user-filters/new',
    name: 'UserFilterCreate',
    component: userFilterFormComponent,
    actions: [
      'web.action.user_filter_create',
      'web.action.user_filter_edit',
      'web.action.user_filter_delete',
      'web.action.user_filter_copy',
    ],
    defaultRoles: ['sys.admin'],
    meta: { requiresAuth: true },
  }),
  defineRoute('web.route.user_filter_detail', {
    sequence: 72,
    title: _lt('Saved Filter'),
    path: 'web/user-filters/:id',
    name: 'UserFilterDetail',
    component: userFilterFormComponent,
    actions: [
      'web.action.user_filter_create',
      'web.action.user_filter_edit',
      'web.action.user_filter_delete',
      'web.action.user_filter_copy',
    ],
    defaultRoles: ['sys.admin'],
    meta: { requiresAuth: true },
  }),
];

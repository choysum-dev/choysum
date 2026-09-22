// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { RouteRecordRaw } from 'vue-router';

/**
 * Maintainer-only gallery routes (not registered in product menus).
 * Plain records (no defineRoute resourceId) so any authenticated user can open
 * the isolation gallery without a dedicated RoleUiResource grant.
 */
export const choyUiRoutes: RouteRecordRaw[] = [
  {
    path: '__choy_gallery',
    name: 'ChoyUiGallery',
    component: () => import('../pages/Gallery.vue'),
    meta: {
      requiresAuth: true,
      hideInMenu: true,
      title: 'Choy UI Gallery',
    },
  },
];

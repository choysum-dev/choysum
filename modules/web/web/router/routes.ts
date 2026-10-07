// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import 'vue-router';
import type { RouteRecordRaw } from 'vue-router';
import { resolveRuntimeDefaultLandPath } from './resolveRuntimeDefaultLandPath';

/**
 * Static route configuration for the web shell.
 * Home/Welcome are retired; `/` and catch-all resolve via the default land path.
 */
export const routes: RouteRecordRaw[] = [
  {
    path: '/',
    name: 'Root',
    redirect: () => resolveRuntimeDefaultLandPath(),
  },

  {
    path: '/',
    component: () => import('../components/layout/ChoyGuestShell.vue'),
    name: 'GuestLayout',
    props: {
      showHeader: true,
      showFooter: true,
    },
    children: [
      {
        path: 'error/:code(\\d+)',
        name: 'Error',
        component: () => import('../pages/ErrorView.vue'),
        meta: {
          requiresAuth: false,
        },
      },
    ],
  },

  {
    path: '/',
    component: () => import('../components/layout/ChoyAppShell.vue'),
    name: 'AppLayout',
    props: {
      // Sidebar menu is rendered inside ChoyAppShell (useMenu); route children
      // cannot fill named #aside slots via a plain router-view.
      showSidebar: true,
      showHeader: true,
      showFooter: true,
    },
    children: [],
  },

  {
    path: '/:pathMatch(.*)*',
    name: 'CatchAll',
    redirect: (to) => {
      const target = resolveRuntimeDefaultLandPath();
      // Unregistered land paths rematch this catch-all; bail to Error instead of looping.
      if (!target || target === to.path) {
        return { name: 'Error', params: { code: '404' } };
      }
      return target;
    },
  },
];

/**
 * Default route list consumed by the router factory.
 */
export default routes;

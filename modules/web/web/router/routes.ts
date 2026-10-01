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
    component: () => import('../components/layout/ChoyWebShell.vue'),
    name: 'Layout',
    props: {
      showSidebar: true,
      showHeader: true,
      // Footer slot cannot be filled via router-view; omit empty chrome.
      showFooter: false,
    },
    children: [],
  },

  {
    path: '/',
    component: () => import('../components/layout/ChoyWebShell.vue'),
    name: 'AppLayout',
    props: {
      // Sidebar menu is rendered inside ChoyWebShell (useMenu); route children
      // cannot fill named #aside slots via a plain router-view.
      showSidebar: true,
      showHeader: true,
      showFooter: false,
    },
    children: [],
  },

  {
    path: '/error/:code(\\d+)',
    name: 'Error',
    component: () => import('../pages/ErrorView.vue'),
    meta: {
      requiresAuth: false,
    },
  },

  {
    path: '/:pathMatch(.*)*',
    name: 'CatchAll',
    redirect: () => resolveRuntimeDefaultLandPath(),
  },
];

/**
 * Default route list consumed by the router factory.
 */
export default routes;

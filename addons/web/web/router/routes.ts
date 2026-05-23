// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import 'vue-router';
import type { RouteRecordRaw } from 'vue-router';
import { defineRoute } from '@/core/web/resource';

/**
 * Static route configuration for the web shell.
 */
export const routes: RouteRecordRaw[] = [
  {
    path: '/',
    redirect: '/home',
    name: 'Root',
  },

  {
    path: '/',
    component: () => import('../components/layout/OLayout.vue'),
    name: 'Layout',
    props: {
      showSidebar: false,
      showHeader: true,
      showFooter: true,
    },
    children: [
      defineRoute('web.route.home', {
        sequence: 1,
        title: '首页',
        defaultRoles: ['base.user'],
        path: 'home',
        name: 'Home',
        component: () => import('../pages/HomeView.vue'),
        meta: {
          requiresAuth: true,
          keepAlive: true,
        },
      }),
    ],
  },

  {
    path: '/',
    component: () => import('../components/layout/OLayout.vue'),
    name: 'AppLayout',
    props: {
      showSidebar: true,
      showHeader: true,
      showFooter: true,
    },
    children: [],
  },

  {
    path: '/:pathMatch(.*)*',
    redirect: '/home',
    name: 'CatchAll',
  },
];

/**
 * Default route list consumed by the router factory.
 */
export default routes;

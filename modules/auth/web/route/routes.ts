// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { RouteRecordRaw } from 'vue-router';
import { defineRoute } from '@/core/web/resource';
import { createTranslate } from '@/web/web/i18n';

const { _t } = createTranslate('auth', { output: 'reference', scope: 'web/route/routes' });

/**
 * Auth-only routes that do not require the main application layout.
 */
export const authRoutes: RouteRecordRaw[] = [
  // Login page.
  defineRoute('auth.route.login', {
    title: _t('Login'),
    path: 'login',
    name: 'login',
    component: () => import('../pages/Login.vue'),
    meta: {
      requiresAuth: false,
      isAuthPage: true,
    },
  }),

  // Logout page.
  defineRoute('auth.route.logout', {
    title: _t('Logout'),
    path: 'logout',
    name: 'logout',
    component: () => import('../pages/Logout.vue'),
    meta: {
      requiresAuth: false,
      isAuthPage: true,
    },
  }),
];

/**
 * Auth application routes mounted under the authenticated layout.
 */
export const appRoutes: RouteRecordRaw[] = [
  defineRoute('auth.route.user_list', {
    sequence: 10,
    title: _t('User List'),
    path: 'auth/users',
    name: 'UserList',
    component: () => import('../pages/UserList.vue'),
    actions: ['auth.action.user_create', 'auth.action.user_edit', 'auth.action.user_delete', 'auth.action.user_copy'],
    meta: { requiresAuth: true },
  }),
  defineRoute('auth.route.user_detail', {
    sequence: 20,
    title: _t('User Details'),
    path: 'auth/users/:id',
    name: 'UserDetail',
    component: () => import('../pages/User.vue'),
    props: route => ({ recordId: route.params.id, viewMode: 'display' }),
    actions: ['auth.action.user_create', 'auth.action.user_edit', 'auth.action.user_delete', 'auth.action.user_copy'],
    meta: { requiresAuth: true },
  }),
  defineRoute('auth.route.user_create', {
    sequence: 30,
    title: _t('Create User'),
    path: 'auth/users/new',
    name: 'UserCreate',
    component: () => import('../pages/User.vue'),
    props: { viewMode: 'create' },
    actions: ['auth.action.user_create', 'auth.action.user_edit', 'auth.action.user_delete', 'auth.action.user_copy'],
    meta: { requiresAuth: true },
  }),

  // Role routes.
  defineRoute('auth.route.role_list', {
    sequence: 10,
    title: _t('Role Management'),
    path: 'auth/roles',
    name: 'RoleList',
    component: () => import('../pages/RoleList.vue'),
    actions: ['auth.action.role_create', 'auth.action.role_edit', 'auth.action.role_delete', 'auth.action.role_copy'],
    meta: { requiresAuth: true },
  }),
  defineRoute('auth.route.role_detail', {
    sequence: 20,
    title: _t('Role Details'),
    path: 'auth/roles/:id',
    name: 'RoleDetail',
    component: () => import('../pages/Role.vue'),
    props: route => ({ recordId: route.params.id, viewMode: 'display' }),
    actions: ['auth.action.role_create', 'auth.action.role_edit', 'auth.action.role_delete', 'auth.action.role_copy'],
    meta: { requiresAuth: true },
  }),
  defineRoute('auth.route.role_create', {
    sequence: 30,
    title: _t('Create Role'),
    path: 'auth/roles/new',
    name: 'RoleCreate',
    component: () => import('../pages/Role.vue'),
    props: { viewMode: 'create' },
    actions: ['auth.action.role_create', 'auth.action.role_edit', 'auth.action.role_delete', 'auth.action.role_copy'],
    meta: { requiresAuth: true },
  }),
  defineRoute('auth.route.session_list', {
    sequence: 10,
    title: _t('Session Management'),
    path: 'auth/sessions',
    name: 'SessionList',
    component: () => import('../pages/SessionList.vue'),
    actions: ['auth.action.session_create', 'auth.action.session_edit', 'auth.action.session_delete', 'auth.action.session_copy'],
    meta: { requiresAuth: true },
  }),
  defineRoute('auth.route.session_detail', {
    sequence: 20,
    title: _t('Session Details'),
    path: 'auth/sessions/:id',
    name: 'SessionDetail',
    component: () => import('../pages/Session.vue'),
    props: route => ({ recordId: route.params.id, viewMode: 'display' }),
    actions: ['auth.action.session_create', 'auth.action.session_edit', 'auth.action.session_delete', 'auth.action.session_copy'],
    meta: { requiresAuth: true },
  }),
  defineRoute('auth.route.session_create', {
    sequence: 30,
    title: _t('Create Session'),
    path: 'auth/sessions/new',
    name: 'SessionCreate',
    component: () => import('../pages/Session.vue'),
    props: { viewMode: 'create' },
    actions: ['auth.action.session_create', 'auth.action.session_edit', 'auth.action.session_delete', 'auth.action.session_copy'],
    meta: { requiresAuth: true },
  }),

  // Token routes.
  defineRoute('auth.route.token_list', {
    sequence: 10,
    title: _t('Token Management'),
    path: 'auth/tokens',
    name: 'TokenList',
    component: () => import('../pages/TokenList.vue'),
    actions: ['auth.action.token_create', 'auth.action.token_edit', 'auth.action.token_delete', 'auth.action.token_copy'],
    meta: { requiresAuth: true },
  }),
  defineRoute('auth.route.token_detail', {
    sequence: 20,
    title: _t('Token Details'),
    path: 'auth/tokens/:id',
    name: 'TokenDetail',
    component: () => import('../pages/Token.vue'),
    props: route => ({ recordId: route.params.id, viewMode: 'display' }),
    actions: ['auth.action.token_create', 'auth.action.token_edit', 'auth.action.token_delete', 'auth.action.token_copy'],
    meta: { requiresAuth: true },
  }),
  defineRoute('auth.route.token_create', {
    sequence: 30,
    title: _t('Create Token'),
    path: 'auth/tokens/new',
    name: 'TokenCreate',
    component: () => import('../pages/Token.vue'),
    props: { viewMode: 'create' },
    actions: ['auth.action.token_create', 'auth.action.token_edit', 'auth.action.token_delete', 'auth.action.token_copy'],
    meta: { requiresAuth: true },
  }),
  defineRoute('auth.route.token_kanban', {
    sequence: 40,
    title: _t('Token Board'),
    path: 'auth/tokens/kanban',
    name: 'TokenKanban',
    component: () => import('../pages/TokenKanban.vue'),
    actions: ['auth.action.token_create', 'auth.action.token_edit', 'auth.action.token_delete', 'auth.action.token_copy'],
    meta: { requiresAuth: true },
  }),
];

// Add the registration route when self-service registration is enabled.
if (import.meta.env.CHOYSUM_ENABLE_REGISTRATION !== false) {
  authRoutes.push(
    defineRoute('auth.route.register', {
      title: _t('Register'),
      path: 'register',
      name: 'register',
      component: () => import('../pages/Register.vue'),
      meta: {
        requiresAuth: false,
        isAuthPage: true,
      },
    })
  );
}

export default authRoutes;

// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { RouteRecordRaw } from 'vue-router';
import { defineRoute } from '@/core/web/resource';
import { createTranslate } from '@/web/web/i18n';
// Side-effect: register web.action.field_default_* UI resources.
import '../views/field_default_actions';

const { _lt } = createTranslate('web', { scope: 'web/route/fieldDefaultRoutes' });

/**
 * Shared FieldDefault list page. Domain modules mount their own list
 * defineRoute with a literal path so UI_VAL_006 can match menu paths.
 */
export const fieldDefaultListComponent = () => import('../pages/FieldDefaultList.vue');

const fieldDefaultFormComponent = () => import('../pages/FieldDefault.vue');

/**
 * web hosts the web list route plus shared create/detail routes.
 * Other apps declare list routes in their own modules. core is excluded
 * (FieldDefault is not injected for application=core).
 *
 * actions must be string-literal arrays (UI_DECL_008); ids match
 * defineModelActions('web.FieldDefault').
 */
export const fieldDefaultRoutes: RouteRecordRaw[] = [
  defineRoute('web.route.field_default_list', {
    sequence: 60,
    title: _lt('Field Defaults'),
    path: 'web/field-defaults/web',
    name: 'FieldDefaultList_web',
    component: fieldDefaultListComponent,
    actions: [
      'web.action.field_default_create',
      'web.action.field_default_edit',
      'web.action.field_default_delete',
      'web.action.field_default_copy',
    ],
    defaultRoles: ['sys.admin'],
    meta: { requiresAuth: true, fieldDefaultApplication: 'web' },
  }),
  // Register /new before /:id so "new" is not captured as an id.
  defineRoute('web.route.field_default_create', {
    sequence: 61,
    title: _lt('New Field Default'),
    path: 'web/field-defaults/:application/new',
    name: 'FieldDefaultCreate',
    component: fieldDefaultFormComponent,
    actions: [
      'web.action.field_default_create',
      'web.action.field_default_edit',
      'web.action.field_default_delete',
      'web.action.field_default_copy',
    ],
    defaultRoles: ['sys.admin'],
    meta: { requiresAuth: true },
  }),
  defineRoute('web.route.field_default_detail', {
    sequence: 62,
    title: _lt('Field Default'),
    path: 'web/field-defaults/:application/:id',
    name: 'FieldDefaultDetail',
    component: fieldDefaultFormComponent,
    actions: [
      'web.action.field_default_create',
      'web.action.field_default_edit',
      'web.action.field_default_delete',
      'web.action.field_default_copy',
    ],
    defaultRoles: ['sys.admin'],
    meta: { requiresAuth: true },
  }),
];

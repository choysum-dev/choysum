// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { RouteRecordRaw } from 'vue-router';
import { defineRoute } from '@/core/web/resource';
import { createTranslate } from '@/web/web/i18n';
// Side-effect: register web.action.app_setting_* UI resources.
import '../views/app_setting_actions';

const { _lt } = createTranslate('web', { scope: 'web/route/appSettingRoutes' });

/**
 * Shared AppSetting list page. Domain modules mount their own list
 * defineRoute with a literal path so UI_VAL_006 can match menu paths.
 */
export const appSettingListComponent = () => import('../pages/AppSettingList.vue');

const appSettingFormComponent = () => import('../pages/AppSetting.vue');

/**
 * web hosts the web list route plus shared create/detail routes.
 * Other apps declare list routes in their own modules. core is excluded
 * (AppSetting is not injected for application=core).
 *
 * actions must be string-literal arrays (UI_DECL_008); ids match
 * defineModelActions('web.AppSetting').
 */
export const appSettingRoutes: RouteRecordRaw[] = [
  defineRoute('web.route.app_setting_list', {
    sequence: 70,
    title: _lt('Parameters'),
    path: 'web/app-settings/web',
    name: 'AppSettingList_web',
    component: appSettingListComponent,
    actions: [
      'web.action.app_setting_create',
      'web.action.app_setting_edit',
      'web.action.app_setting_delete',
      'web.action.app_setting_copy',
    ],
    defaultRoles: ['sys.admin'],
    meta: { requiresAuth: true, appSettingApplication: 'web' },
  }),
  // Register /new before /:id so "new" is not captured as an id.
  defineRoute('web.route.app_setting_create', {
    sequence: 71,
    title: _lt('New Parameter'),
    path: 'web/app-settings/:application/new',
    name: 'AppSettingCreate',
    component: appSettingFormComponent,
    actions: [
      'web.action.app_setting_create',
      'web.action.app_setting_edit',
      'web.action.app_setting_delete',
      'web.action.app_setting_copy',
    ],
    defaultRoles: ['sys.admin'],
    meta: { requiresAuth: true },
  }),
  defineRoute('web.route.app_setting_detail', {
    sequence: 72,
    title: _lt('Parameter'),
    path: 'web/app-settings/:application/:id',
    name: 'AppSettingDetail',
    component: appSettingFormComponent,
    actions: [
      'web.action.app_setting_create',
      'web.action.app_setting_edit',
      'web.action.app_setting_delete',
      'web.action.app_setting_copy',
    ],
    defaultRoles: ['sys.admin'],
    meta: { requiresAuth: true },
  }),
];

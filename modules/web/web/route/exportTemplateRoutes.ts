// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { RouteRecordRaw } from 'vue-router';
import { defineRoute } from '@/core/web/resource';
import { createTranslate } from '@/web/web/i18n';
// Side-effect: register web.action.export_template_* UI resources.
import '../views/export_template_actions';

const { _lt } = createTranslate('web', { scope: 'web/route/exportTemplateRoutes' });

/**
 * Settings → web → Export Templates admin ledger.
 * Owner Application is always web; no per-app :application segment.
 *
 * actions must be string-literal arrays (UI_DECL_008); ids match
 * defineModelActions('web.ExportTemplate').
 */
export const exportTemplateListComponent = () => import('../pages/ExportTemplateList.vue');

const exportTemplateFormComponent = () => import('../pages/ExportTemplate.vue');

export const exportTemplateRoutes: RouteRecordRaw[] = [
  defineRoute('web.route.export_template_list', {
    sequence: 80,
    title: _lt('Export Templates'),
    path: 'web/export-templates',
    name: 'ExportTemplateList_web',
    component: exportTemplateListComponent,
    actions: [
      'web.action.export_template_create',
      'web.action.export_template_edit',
      'web.action.export_template_delete',
      'web.action.export_template_copy',
    ],
    defaultRoles: ['sys.admin'],
    meta: { requiresAuth: true },
  }),
  defineRoute('web.route.export_template_create', {
    sequence: 81,
    title: _lt('New Export Template'),
    path: 'web/export-templates/new',
    name: 'ExportTemplateCreate',
    component: exportTemplateFormComponent,
    actions: [
      'web.action.export_template_create',
      'web.action.export_template_edit',
      'web.action.export_template_delete',
      'web.action.export_template_copy',
    ],
    defaultRoles: ['sys.admin'],
    meta: { requiresAuth: true },
  }),
  defineRoute('web.route.export_template_detail', {
    sequence: 82,
    title: _lt('Export Template'),
    path: 'web/export-templates/:id',
    name: 'ExportTemplateDetail',
    component: exportTemplateFormComponent,
    actions: [
      'web.action.export_template_create',
      'web.action.export_template_edit',
      'web.action.export_template_delete',
      'web.action.export_template_copy',
    ],
    defaultRoles: ['sys.admin'],
    meta: { requiresAuth: true },
  }),
];

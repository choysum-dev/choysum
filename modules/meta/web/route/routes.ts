// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { RouteRecordRaw } from 'vue-router';
import { defineRoute } from '@/core/web/resource';
import { createTranslate } from '@/web/web/i18n';

const { _lt } = createTranslate('meta', { scope: 'web/route/routes' });

/**
 * Route table for the meta module management surfaces.
 */
export const metaRoutes: RouteRecordRaw[] = [
  defineRoute('meta.route.module_board', {
    sequence: 10,
    title: _lt('Module Management'),
    path: 'meta/modules',
    name: 'MetaModuleList',
    component: () => import('../pages/ModuleList.vue'),
    actions: ['meta.action.module_install', 'meta.action.module_upgrade', 'meta.action.module_uninstall', 'meta.action.module_sync_index'],
    requires: [{ model: 'meta.MetaModuleIndex' }, { model: 'meta.MetaModule' }],
    meta: { requiresAuth: true },
  }),
  defineRoute('meta.route.module_list', {
    sequence: 20,
    title: _lt('Module List'),
    path: 'meta/modules/list',
    name: 'MetaModuleListTable',
    component: () => import('../pages/ModuleListTable.vue'),
    actions: ['meta.action.module_sync_index', 'meta.action.module_index_delete'],
    requires: [{ model: 'meta.MetaModuleIndex' }, { model: 'meta.MetaModule' }],
    meta: { requiresAuth: true },
  }),
  defineRoute('meta.route.module_history', {
    sequence: 30,
    title: _lt('Operation History'),
    path: 'meta/modules/history',
    name: 'MetaModuleHistory',
    component: () => import('../pages/ModuleHistory.vue'),
    actions: ['meta.action.module_management_log_delete'],
    requires: [{ model: 'meta.ModuleManagementLog' }, { model: 'meta.MetaModule' }, { model: 'meta.MetaModuleIndex' }],
    meta: { requiresAuth: true },
  }),
  defineRoute('meta.route.module_detail', {
    sequence: 40,
    title: _lt('Module Detail'),
    path: 'meta/modules/:id',
    name: 'MetaModuleDetail',
    component: () => import('../pages/ModuleDetail.vue'),
    props: route => ({ recordId: route.params.id }),
    actions: ['meta.action.module_index_edit', 'meta.action.module_index_delete', 'meta.action.module_index_copy'],
    requires: [{ model: 'meta.MetaModuleIndex' }, { model: 'meta.MetaModule' }],
    meta: { requiresAuth: true },
  }),
];

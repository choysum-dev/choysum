// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { RouteRecordRaw } from 'vue-router';
import { defineRoute } from '@/core/web/resource';
import { createTranslate } from '@/web/web/i18n';
import {
  appSettingListComponent,
  fieldDefaultListComponent,
  propertyDefinitionListComponent,
  terminologyListComponent,
} from '@/web/web/route';

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
    actions: ['meta.action.module_index_edit', 'meta.action.module_index_delete', 'meta.action.module_index_copy'],
    requires: [{ model: 'meta.MetaModuleIndex' }, { model: 'meta.MetaModule' }],
    meta: { requiresAuth: true },
  }),
  defineRoute('meta.route.terminology_list', {
    sequence: 40,
    title: _lt('Translations'),
    path: 'web/terminology/meta',
    name: 'TranslationTermList_meta',
    component: terminologyListComponent,
    defaultRoles: ['sys.admin'],
    meta: { requiresAuth: true, terminologyApplication: 'meta' },
  }),
  defineRoute('meta.route.property_definition_list', {
    sequence: 50,
    title: _lt('Property Definitions'),
    path: 'web/property-definitions/meta',
    name: 'PropertyDefinitionList_meta',
    component: propertyDefinitionListComponent,
    defaultRoles: ['sys.admin'],
    meta: { requiresAuth: true, propertyDefinitionApplication: 'meta' },
  }),
  defineRoute('meta.route.field_default_list', {
    sequence: 60,
    title: _lt('Field Defaults'),
    path: 'web/field-defaults/meta',
    name: 'FieldDefaultList_meta',
    component: fieldDefaultListComponent,
    defaultRoles: ['sys.admin'],
    meta: { requiresAuth: true, fieldDefaultApplication: 'meta' },
  }),
  defineRoute('meta.route.app_setting_list', {
    sequence: 70,
    title: _lt('Parameters'),
    path: 'web/app-settings/meta',
    name: 'AppSettingList_meta',
    component: appSettingListComponent,
    defaultRoles: ['sys.admin'],
    meta: { requiresAuth: true, appSettingApplication: 'meta' },
  }),
  // Owner=meta MetaModelData ledger: read-only List + detail (no /new).
  defineRoute('meta.route.external_id_list', {
    sequence: 80,
    title: _lt('External IDs'),
    path: 'web/external-ids',
    name: 'MetaModelDataList',
    component: () => import('../pages/MetaModelDataList.vue'),
    actions: [],
    defaultRoles: ['sys.admin'],
    meta: { requiresAuth: true },
  }),
  defineRoute('meta.route.external_id_detail', {
    sequence: 81,
    title: _lt('External ID'),
    path: 'web/external-ids/:id',
    name: 'MetaModelDataDetail',
    component: () => import('../pages/MetaModelData.vue'),
    actions: [],
    defaultRoles: ['sys.admin'],
    meta: { requiresAuth: true },
  }),
];

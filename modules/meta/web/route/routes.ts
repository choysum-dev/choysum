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

/** Sibling paths for Module catalog kanban / list ViewSwitcher. */
const moduleCatalogViewPaths = {
  kanban: '/meta/modules',
  list: '/meta/modules/list',
} as const;

/**
 * Route table for the meta module management surfaces.
 * Catalog kanban + list + show + operation deep-link.
 */
export const metaRoutes: RouteRecordRaw[] = [
  defineRoute('meta.route.module_catalog', {
    sequence: 10,
    title: _lt('Modules'),
    path: 'meta/modules',
    name: 'MetaModuleCatalog',
    component: () => import('../pages/ModuleCatalogKanban.vue'),
    actions: [
      'meta.action.module_install',
      'meta.action.module_upgrade',
      'meta.action.module_uninstall',
      'meta.action.module_sync',
      'meta.action.module_load_demo',
    ],
    defaultRoles: ['sys.admin'],
    // Browse/list catalog; Plan/Apply/RequestSync stay gated by action requires.
    requires: [{ model: 'meta.MetaModuleCatalog' }],
    meta: {
      requiresAuth: true,
      viewKind: 'kanban',
      defaultGroups: [],
      viewPaths: { ...moduleCatalogViewPaths },
    },
  }),
  // Static /list before /:name so "list" is never captured as a module name.
  defineRoute('meta.route.module_catalog_list', {
    sequence: 11,
    title: _lt('Modules'),
    path: 'meta/modules/list',
    name: 'MetaModuleCatalogList',
    component: () => import('../pages/ModuleCatalogList.vue'),
    actions: [
      'meta.action.module_install',
      'meta.action.module_upgrade',
      'meta.action.module_uninstall',
      'meta.action.module_sync',
      'meta.action.module_load_demo',
    ],
    defaultRoles: ['sys.admin'],
    requires: [{ model: 'meta.MetaModuleCatalog' }],
    meta: {
      requiresAuth: true,
      viewKind: 'list',
      defaultGroups: [],
      viewPaths: { ...moduleCatalogViewPaths },
    },
  }),
  // More specific path (operations/:id). Bare /meta/modules/operations still matches :name.
  defineRoute('meta.route.module_operation', {
    sequence: 12,
    title: _lt('Module Operation'),
    path: 'meta/modules/operations/:id',
    name: 'MetaModuleOperationDeepLink',
    component: () => import('../pages/ModuleCatalogKanban.vue'),
    actions: [
      'meta.action.module_install',
      'meta.action.module_upgrade',
      'meta.action.module_uninstall',
      'meta.action.module_sync',
      'meta.action.module_load_demo',
    ],
    defaultRoles: ['sys.admin'],
    requires: [{ model: 'meta.MetaModuleCatalog' }, { model: 'meta.MetaModuleOperation' }],
    meta: {
      requiresAuth: true,
      viewKind: 'kanban',
      defaultGroups: [],
      viewPaths: { ...moduleCatalogViewPaths },
    },
  }),
  defineRoute('meta.route.module_show', {
    sequence: 13,
    title: _lt('Module'),
    path: 'meta/modules/:name',
    name: 'MetaModuleShow',
    component: () => import('../pages/ModuleCatalog.vue'),
    actions: [
      'meta.action.module_install',
      'meta.action.module_upgrade',
      'meta.action.module_uninstall',
      'meta.action.module_sync',
      'meta.action.module_load_demo',
    ],
    defaultRoles: ['sys.admin'],
    // Show is catalog browse; mutation actions stay gated by action requires.
    requires: [{ model: 'meta.MetaModuleCatalog' }],
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

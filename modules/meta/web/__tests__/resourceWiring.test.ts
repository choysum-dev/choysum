// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { getResourceDeclaration, getRouteActionsFromMeta } from '@/core/web/resource';
import { metaMenus } from '../menu/menus';
import { metaRoutes } from '../route/routes';
import {
  moduleInstallAction,
  moduleLoadDemoAction,
  moduleSyncAction,
  moduleUninstallAction,
  moduleUpgradeAction,
} from '../views/module_actions';

const MODULE_ACTIONS = [
  'meta.action.module_install',
  'meta.action.module_upgrade',
  'meta.action.module_uninstall',
  'meta.action.module_sync',
  'meta.action.module_load_demo',
];

test('meta resource wiring: catalog/show/operation routes declare §6.3 actions', () => {
  const byName = Object.fromEntries(metaRoutes.map(route => [String(route.name), route]));

  expect(byName.MetaModuleCatalog?.path).toBe('meta/modules');
  expect(byName.MetaModuleCatalogList?.path).toBe('meta/modules/list');
  expect(byName.MetaModuleShow?.path).toBe('meta/modules/:name');
  expect(byName.MetaModuleOperationDeepLink?.path).toBe('meta/modules/operations/:id');

  expect((byName.MetaModuleCatalog?.meta as any)?.viewKind).toBe('kanban');
  expect((byName.MetaModuleCatalogList?.meta as any)?.viewKind).toBe('list');
  expect((byName.MetaModuleCatalog?.meta as any)?.viewPaths).toEqual({
    kanban: '/meta/modules',
    list: '/meta/modules/list',
  });
  expect((byName.MetaModuleCatalogList?.meta as any)?.viewPaths).toEqual({
    kanban: '/meta/modules',
    list: '/meta/modules/list',
  });

  expect(getRouteActionsFromMeta(byName.MetaModuleCatalog?.meta as any)).toEqual(MODULE_ACTIONS);
  expect(getRouteActionsFromMeta(byName.MetaModuleCatalogList?.meta as any)).toEqual(MODULE_ACTIONS);
  expect(getRouteActionsFromMeta(byName.MetaModuleShow?.meta as any)).toEqual(MODULE_ACTIONS);
  expect(getRouteActionsFromMeta(byName.MetaModuleOperationDeepLink?.meta as any)).toEqual(MODULE_ACTIONS);

  expect(byName.MetaModuleList).toBeUndefined();
  expect(byName.MetaModuleListTable).toBeUndefined();
  expect(byName.MetaModuleHistory).toBeUndefined();
  expect(byName.MetaModuleDetail).toBeUndefined();

  expect(getRouteActionsFromMeta(byName.MetaModelDataList?.meta as any)).toEqual([]);
  expect(getRouteActionsFromMeta(byName.MetaModelDataDetail?.meta as any)).toEqual([]);
});

test('meta resource wiring: Modules app root exposes Catalog leaf at /meta/modules', () => {
  expect(metaMenus).toHaveLength(1);
  expect(metaMenus[0]?.icon).toBeTruthy();
  expect(metaMenus[0]?.path == null || metaMenus[0]?.path === '').toBe(true);
  expect(metaMenus[0]?.children).toHaveLength(1);
  expect(metaMenus[0]?.children?.[0]?.path).toBe('/meta/modules');
  expect(getResourceDeclaration('meta.menu.root')).toBeTruthy();
  expect(getResourceDeclaration('meta.menu.modules')?.requires).toEqual([
    { kind: 'rpc', model: 'meta.MetaModuleCatalog' },
  ]);
  expect(getResourceDeclaration('meta.route.module_catalog')?.requires).toEqual([
    { kind: 'rpc', model: 'meta.MetaModuleCatalog' },
  ]);
  expect(getResourceDeclaration('meta.route.module_catalog_list')?.requires).toEqual([
    { kind: 'rpc', model: 'meta.MetaModuleCatalog' },
  ]);
  expect(getResourceDeclaration('meta.route.module_show')?.requires).toEqual([
    { kind: 'rpc', model: 'meta.MetaModuleCatalog' },
  ]);
  expect(getResourceDeclaration('meta.route.module_operation')?.requires).toEqual([
    { kind: 'rpc', model: 'meta.MetaModuleCatalog' },
    { kind: 'rpc', model: 'meta.MetaModuleOperation' },
  ]);
  expect(getResourceDeclaration('meta.menu.module_board')).toBeUndefined();
  expect(getResourceDeclaration('meta.menu.module_list')).toBeUndefined();
  expect(getResourceDeclaration('meta.menu.module_history')).toBeUndefined();
  expect(getResourceDeclaration('meta.route.module_board')).toBeUndefined();
  expect(getResourceDeclaration('meta.route.module_list')).toBeUndefined();
  expect(getResourceDeclaration('meta.route.module_history')).toBeUndefined();
  expect(getResourceDeclaration('meta.route.module_detail')).toBeUndefined();
});

test('meta resource wiring: module actions match §6.3 ids and requires', () => {
  expect(moduleInstallAction).toBe('meta.action.module_install');
  expect(moduleUpgradeAction).toBe('meta.action.module_upgrade');
  expect(moduleUninstallAction).toBe('meta.action.module_uninstall');
  expect(moduleSyncAction).toBe('meta.action.module_sync');
  expect(moduleLoadDemoAction).toBe('meta.action.module_load_demo');

  const install = getResourceDeclaration('meta.action.module_install');
  expect(install?.kind).toBe('action');
  expect(install?.requires).toEqual([
    { kind: 'rpc', model: 'meta.MetaModuleOperation', method: 'Plan' },
    { kind: 'rpc', model: 'meta.MetaModuleOperation', method: 'Apply' },
  ]);
  expect(JSON.stringify(install?.requires || [])).not.toContain('Execute');

  const sync = getResourceDeclaration('meta.action.module_sync');
  expect(sync?.requires).toEqual([
    { kind: 'rpc', model: 'meta.MetaModuleCatalog', method: 'RequestSync' },
  ]);
  expect(JSON.stringify(sync?.requires || [])).not.toContain('Plan');
  expect(JSON.stringify(sync?.requires || [])).not.toContain('Apply');

  const loadDemo = getResourceDeclaration('meta.action.module_load_demo');
  expect(loadDemo?.requires).toEqual([
    { kind: 'rpc', model: 'meta.MetaModuleOperation', method: 'PlanDemo' },
    { kind: 'rpc', model: 'meta.MetaModuleOperation', method: 'Apply' },
  ]);
});

test('meta resource wiring: legacy sync_index and catalog mutate actions are gone', () => {
  expect(getResourceDeclaration('meta.action.module_sync_index')).toBeUndefined();
  expect(getResourceDeclaration('meta.action.module_catalog_edit')).toBeUndefined();
  expect(getResourceDeclaration('meta.action.module_catalog_delete')).toBeUndefined();
  expect(getResourceDeclaration('meta.action.module_catalog_copy')).toBeUndefined();
  expect(getResourceDeclaration('meta.action.module_operation_delete')).toBeUndefined();
});

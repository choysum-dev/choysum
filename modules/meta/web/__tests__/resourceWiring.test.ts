// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { getResourceDeclaration, getRouteActionsFromMeta } from '@/core/web/resource';
import { metaRoutes } from '../route/routes';
import {
  moduleInstallAction,
  moduleLoadDemoAction,
  moduleSyncAction,
  moduleUninstallAction,
  moduleUpgradeAction,
} from '../views/module_actions';

test('meta resource wiring: each route declares expected actions', () => {
  const byName = Object.fromEntries(metaRoutes.map(route => [String(route.name), route]));

  expect(getRouteActionsFromMeta(byName.MetaModuleList?.meta as any)).toEqual([
    'meta.action.module_install',
    'meta.action.module_upgrade',
    'meta.action.module_uninstall',
    'meta.action.module_sync',
    'meta.action.module_load_demo',
  ]);
  expect(getRouteActionsFromMeta(byName.MetaModuleListTable?.meta as any)).toEqual([
    'meta.action.module_sync',
  ]);
  expect(getRouteActionsFromMeta(byName.MetaModuleHistory?.meta as any)).toEqual([]);
  expect(getRouteActionsFromMeta(byName.MetaModuleDetail?.meta as any)).toEqual([]);
  expect(getRouteActionsFromMeta(byName.MetaModelDataList?.meta as any)).toEqual([]);
  expect(getRouteActionsFromMeta(byName.MetaModelDataDetail?.meta as any)).toEqual([]);
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
  // Execute is task-worker only — never granted via interactive install action.
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

// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createTranslate } from '@/core/service/i18n';
import { defineAction, getResourceDeclarationFromMeta } from '@/core/web/resource';
import { metaRoutes } from '../route/routes';

const { _lt } = createTranslate('meta', { scope: 'web/views' });

test('meta resource wiring: each route declares expected actions', () => {
  const byName = Object.fromEntries(metaRoutes.map(route => [String(route.name), route]));

  expect(getResourceDeclarationFromMeta(byName.MetaModuleList?.meta as any)?.actions).toEqual([
    'meta.action.module_install',
    'meta.action.module_upgrade',
    'meta.action.module_uninstall',
    'meta.action.module_sync_index',
  ]);
  expect(getResourceDeclarationFromMeta(byName.MetaModuleListTable?.meta as any)?.actions).toEqual([
    'meta.action.module_sync_index',
    'meta.action.module_index_delete',
  ]);
  expect(getResourceDeclarationFromMeta(byName.MetaModuleHistory?.meta as any)?.actions).toEqual([
    'meta.action.module_management_log_delete',
  ]);
  expect(getResourceDeclarationFromMeta(byName.MetaModuleDetail?.meta as any)?.actions).toEqual([
    'meta.action.module_index_edit',
    'meta.action.module_index_delete',
    'meta.action.module_index_copy',
  ]);
});

test('meta resource wiring: defineAction returns the registered action id', () => {
  expect(defineAction('meta.action.module_install', { title: _lt('Install Module') })).toBe(
    'meta.action.module_install'
  );
  expect(defineAction('meta.action.module_upgrade', { title: _lt('Upgrade Module') })).toBe(
    'meta.action.module_upgrade'
  );
  expect(defineAction('meta.action.module_uninstall', { title: _lt('Uninstall Module') })).toBe(
    'meta.action.module_uninstall'
  );
  expect(defineAction('meta.action.module_sync_index', { title: _lt('Sync Module Index') })).toBe(
    'meta.action.module_sync_index'
  );
});

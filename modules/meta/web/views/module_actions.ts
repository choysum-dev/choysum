// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineAction } from '@/core/web/resource';
import { createTranslate } from '@/web/web/i18n';

const { _lt } = createTranslate('meta', { scope: 'web/views/module_actions' });

/** Install modules: Plan + Apply with Action=install (UI filters by planned action). */
export const moduleInstallAction = defineAction('meta.action.module_install', {
  title: _lt('Install Module'),
  defaultRoles: ['sys.admin'],
  requires: [
    { model: 'meta.MetaModuleOperation', method: 'Plan' },
    { model: 'meta.MetaModuleOperation', method: 'Apply' },
  ],
});

/** Upgrade modules: Plan + Apply with Action=upgrade. */
export const moduleUpgradeAction = defineAction('meta.action.module_upgrade', {
  title: _lt('Upgrade Module'),
  defaultRoles: ['sys.admin'],
  requires: [
    { model: 'meta.MetaModuleOperation', method: 'Plan' },
    { model: 'meta.MetaModuleOperation', method: 'Apply' },
  ],
});

/** Uninstall modules: Plan + Apply with Action=uninstall. */
export const moduleUninstallAction = defineAction('meta.action.module_uninstall', {
  title: _lt('Uninstall Module'),
  defaultRoles: ['sys.admin'],
  requires: [
    { model: 'meta.MetaModuleOperation', method: 'Plan' },
    { model: 'meta.MetaModuleOperation', method: 'Apply' },
  ],
});

/** Sync the module catalog snapshot (local origin). */
export const moduleSyncAction = defineAction('meta.action.module_sync', {
  title: _lt('Sync Modules'),
  defaultRoles: ['sys.admin'],
  requires: [{ model: 'meta.MetaModuleCatalog', method: 'RequestSync' }],
});

/** Load demo data for modules that still need it (PlanDemo + Apply). */
export const moduleLoadDemoAction = defineAction('meta.action.module_load_demo', {
  title: _lt('Load Demo Data'),
  defaultRoles: ['sys.admin'],
  requires: [
    { model: 'meta.MetaModuleOperation', method: 'PlanDemo' },
    { model: 'meta.MetaModuleOperation', method: 'Apply' },
  ],
});

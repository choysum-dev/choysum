// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';

/**
 * Loads a meta web source file relative to the current test directory.
 */
function source(relativePath: string): string {
  return readFileSync(resolve(__dirname, '..', relativePath), 'utf8');
}

describe('meta resource wiring', () => {
  it('declares business defineAction resources in the corresponding views', () => {
    const list = source('views/ModuleListView.vue');
    const kanban = source('views/ModuleKanbanView.vue');

    expect(kanban).toContain("defineAction('meta.action.module_install'");
    expect(kanban).toContain("defineAction('meta.action.module_upgrade'");
    expect(kanban).toContain("defineAction('meta.action.module_uninstall'");
    expect(kanban).toContain("defineAction('meta.action.module_sync_index'");
    expect(list).toContain("defineAction('meta.action.module_sync_index'");
  });

  it('declares route actions for all meta routes', () => {
    const routes = source('route/routes.ts');

    expect(routes).toContain(
      "actions: ['meta.action.module_install', 'meta.action.module_upgrade', 'meta.action.module_uninstall', 'meta.action.module_sync_index']"
    );
    expect(routes).toContain("actions: ['meta.action.module_sync_index', 'meta.action.ir_module_index_delete']");
    expect(routes).toContain("actions: ['meta.action.module_management_log_delete']");
    expect(routes).toContain("actions: ['meta.action.ir_module_index_edit', 'meta.action.ir_module_index_delete', 'meta.action.ir_module_index_copy']");
  });

  it('wires meta views to permission-aware action ids', () => {
    const list = source('views/ModuleListView.vue');
    const log = source('views/ModuleLogListView.vue');
    const detail = source('views/ModuleDetailView.vue');
    const kanban = source('views/ModuleKanbanView.vue');

    expect(list).toContain("defineModelActions('meta.IrModuleIndex', {");
    expect(list).toContain("entityTitle: _tRef('Module Index')");
    expect(list).toContain(':action-ids="{ delete: moduleIndexActions.delete }"');
    expect(list).toContain(':has-action="hasAction"');
    expect(list).toContain('moduleSyncIndexAction');
    expect(list).toContain('usePermission');

    expect(log).toContain("defineModelActions('meta.ModuleManagementLog', {");
    expect(log).toContain("entityTitle: _tRef('Module Operation History')");
    expect(log).toContain(':action-ids="{ delete: moduleLogActions.delete }"');
    expect(log).toContain(':has-action="hasAction"');
    expect(log).toContain('usePermission');

    expect(detail).toContain("defineModelActions('meta.IrModuleIndex', {");
    expect(detail).toContain("entityTitle: _tRef('Module Index')");
    expect(detail).toContain('edit: moduleIndexActions.edit');
    expect(detail).toContain('copy: moduleIndexActions.copy');
    expect(detail).toContain('delete: moduleIndexActions.delete');
    expect(detail).toContain(':has-action="hasAction"');

    expect(kanban).toContain('hasAction(moduleInstallAction)');
    expect(kanban).toContain('hasAction(moduleUpgradeAction)');
    expect(kanban).toContain('hasAction(moduleUninstallAction)');
    expect(kanban).toContain('hasAction(moduleSyncIndexAction)');
    expect(kanban).toContain("canRoute('meta.route.module_list')");
    expect(kanban).toContain("canRoute('meta.route.module_history')");
  });
});

// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { menus } from '../menu/menus';
import { userFilterActions } from '../views/user_filter_actions';
import { exportTemplateActions } from '../views/export_template_actions';
import { userFilterRoutes } from '../route/userFilterRoutes';
import { exportTemplateRoutes } from '../route/exportTemplateRoutes';

/**
 * Nail primary-vs-admin invariant at the wiring layer:
 * Settings → web mounts Saved Filters / Export Templates for sys.admin only;
 * create/copy/edit/delete actions exist for the ledger; paths are Owner-web only
 * (no per-app Settings List). OSearch / ExportPanel remain the end-user entry.
 */
test('Settings → web mounts Saved Filters and Export Templates for sys.admin only', () => {
  const web = menus[0]?.children?.find(c => c.id === 'web.menu.settings_app_web');
  expect(web).toBeTruthy();
  const ids = web?.children?.map(c => c.id) ?? [];
  expect(ids).toContain('web.menu.user_filters');
  expect(ids).toContain('web.menu.export_templates');

  const favorites = web?.children?.find(c => c.id === 'web.menu.user_filters') as any;
  expect(favorites?.path).toBe('/web/user-filters');
  expect(favorites?.meta?.resource?.defaultRoles).toEqual(['sys.admin']);

  const templates = web?.children?.find(c => c.id === 'web.menu.export_templates') as any;
  expect(templates?.path).toBe('/web/export-templates');
  expect(templates?.meta?.resource?.defaultRoles).toEqual(['sys.admin']);
});

test('admin ledger actions expose create/edit/delete/copy for both models', () => {
  expect(userFilterActions.create).toBe('web.action.user_filter_create');
  expect(userFilterActions.edit).toBe('web.action.user_filter_edit');
  expect(userFilterActions.delete).toBe('web.action.user_filter_delete');
  expect(userFilterActions.copy).toBe('web.action.user_filter_copy');

  expect(exportTemplateActions.create).toBe('web.action.export_template_create');
  expect(exportTemplateActions.edit).toBe('web.action.export_template_edit');
  expect(exportTemplateActions.delete).toBe('web.action.export_template_delete');
  expect(exportTemplateActions.copy).toBe('web.action.export_template_copy');
});

test('admin routes are Owner-web only (no :application per-app host)', () => {
  for (const r of [...userFilterRoutes, ...exportTemplateRoutes]) {
    expect(String(r.path)).not.toContain(':application');
    expect((r as any).meta?.resource?.defaultRoles).toEqual(['sys.admin']);
  }
});

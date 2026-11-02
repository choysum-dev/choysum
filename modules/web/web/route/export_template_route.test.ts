// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { exportTemplateListComponent, exportTemplateRoutes } from './exportTemplateRoutes';

test('exportTemplateListComponent is exported', () => {
  expect(typeof exportTemplateListComponent).toBe('function');
});

test('exportTemplateRoutes host web list plus create/detail with full CRUD actions', () => {
  const names = exportTemplateRoutes.map(r => String(r.name));
  expect(names).toEqual(['ExportTemplateList_web', 'ExportTemplateCreate', 'ExportTemplateDetail']);

  const list = exportTemplateRoutes.find(r => r.name === 'ExportTemplateList_web') as any;
  expect(list.path).toBe('web/export-templates');
  expect(list.meta?.resource?.defaultRoles).toEqual(['sys.admin']);
  expect(list.component).toBe(exportTemplateListComponent);
  expect(list.meta?.resource?.actions).toEqual([
    'web.action.export_template_create',
    'web.action.export_template_edit',
    'web.action.export_template_delete',
    'web.action.export_template_copy',
  ]);

  const create = exportTemplateRoutes.find(r => r.name === 'ExportTemplateCreate') as any;
  expect(create.path).toBe('web/export-templates/new');

  const detail = exportTemplateRoutes.find(r => r.name === 'ExportTemplateDetail') as any;
  expect(detail.path).toBe('web/export-templates/:id');

  const createIdx = exportTemplateRoutes.findIndex(r => r.name === 'ExportTemplateCreate');
  const detailIdx = exportTemplateRoutes.findIndex(r => r.name === 'ExportTemplateDetail');
  expect(createIdx).toBeGreaterThanOrEqual(0);
  expect(createIdx).toBeLessThan(detailIdx);
});

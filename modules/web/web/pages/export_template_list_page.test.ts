// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { useExportTemplateListPage } from './export_template_list_page';

test('useExportTemplateListPage binds fixed web.ExportTemplate store and title', () => {
  const created: string[] = [];
  const api = useExportTemplateListPage({
    scopeManager: {},
    createStore: ((name: string) => {
      created.push(name);
      return { id: name } as any;
    }) as any,
    translate: m => m,
  });
  expect(api.pageTitle.value).toBe('Export Templates');
  expect(api.exportTemplateStore.value).toEqual({ id: 'web.ExportTemplate' });
  expect(created).toEqual(['web.ExportTemplate']);
});

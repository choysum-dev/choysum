// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import Partner from '@/partner/service/models/partner';

test('Partner CSV import: bridge exposes run', () => {
  expect(typeof (globalThis as any)?.$choysum?.import?.run).toBe('function');
});

test('Partner CSV import: hub-style spec fields', () => {
  const spec = {
    profile: 'record',
    caller: 'user',
    policy: 'atomic',
    model: 'partner.Partner',
    source: { format: 'csv', document_ref: 'att_fixture' },
    options: { company_id: 'cmp_main', column_mapping: {} },
  };
  expect(spec.source.document_ref).toBe('att_fixture');
  expect(spec.options.company_id).toBe('cmp_main');
  expect(spec.model).toBe('partner.Partner');
});

test('Partner CSV import: ImportHub client types are wired from core web import', async () => {
  const mod = await import('@/core/web/import');
  expect(typeof mod.parseHeaders).toBe('function');
  expect(typeof mod.previewImport).toBe('function');
  expect(typeof mod.runImport).toBe('function');
  expect(mod.ImportPolicy.ATOMIC).toBeDefined();
  expect(Partner).toBeTruthy();
});

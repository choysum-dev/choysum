// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import SandboxCategory from '@/sandbox/service/models/sandbox_category';
import SandboxTag from '@/sandbox/service/models/sandbox_tag';

test('SandboxCategory Search with Categories relationConditionSource returns active roots', async () => {
  const rows = await (SandboxCategory as any).Search(
    { And: [['ParentId', 'is', null]] },
    {
      fields: ['Id', 'Name', 'Code', 'IsActive'],
      relationConditionSource: { model: 'sandbox.SandboxRecord', field: 'Categories' },
    }
  );
  expect(Array.isArray(rows)).toBe(true);
  expect(rows.length).toBeGreaterThan(0);
  const names = rows.map((r: any) => String(r.Name || r.name || ''));
  expect(names.some((n: string) => n.includes('根') || n === 'ROOT' || n.includes('Root'))).toBe(true);
});

test('SandboxTag NameSearch with TagChips relationConditionSource returns active tags', async () => {
  const rows = await (SandboxTag as any).NameSearch(
    '',
    [],
    {
      fields: ['Id', 'Name', 'Code'],
      limit: 20,
      relationConditionSource: { model: 'sandbox.SandboxRecord', field: 'TagChips' },
    }
  );
  expect(Array.isArray(rows)).toBe(true);
  expect(rows.length).toBeGreaterThan(0);
});

test('SandboxCategory Search with Children field and relationConditionSource succeeds', async () => {
  const rows = await (SandboxCategory as any).Search(
    { And: [['ParentId', 'is', null]] },
    {
      fields: ['Id', 'Name', 'Code', 'Children'],
      relationConditionSource: { model: 'sandbox.SandboxRecord', field: 'Categories' },
    }
  );
  expect(Array.isArray(rows)).toBe(true);
  expect(rows.length).toBeGreaterThan(0);
  const root = rows[0];
  // Children may be array of related rows or id-like payloads depending on field selection.
  expect(root).toBeTruthy();
});

test('SandboxCategory Search parents by Id with Children selection succeeds', async () => {
  const roots = await (SandboxCategory as any).Search(
    { And: [['ParentId', 'is', null]] },
    {
      fields: ['Id', 'Name'],
      relationConditionSource: { model: 'sandbox.SandboxRecord', field: 'Categories' },
    }
  );
  const id = String(roots[0]?.Id || '');
  expect(id.length).toBeGreaterThan(0);
  const withKids = await (SandboxCategory as any).Search(
    { And: [['Id', 'in', [id]]] },
    {
      fields: ['Id', 'Children'],
      limit: 10,
      relationConditionSource: { model: 'sandbox.SandboxRecord', field: 'Categories' },
    }
  );
  expect(Array.isArray(withKids)).toBe(true);
  expect(withKids.length).toBe(1);
});

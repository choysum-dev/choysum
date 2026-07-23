// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import Role from '@/auth/service/models/role';
import { MetadataStorage } from '@/core/service/api/metadata';
import { withContext } from '@/core/service/api/context';

function uid(prefix: string): string {
  const xid = (globalThis as any).$choysum?.xid?.New?.();
  const u = typeof xid === 'string' && xid.trim() ? xid.trim() : String(Date.now());
  return `${prefix}_${u}`;
}

test('auth.role: Description translate metadata is enabled', () => {
  const field = MetadataStorage.instance.getModelMetadata(Role).fields.get('Description');
  expect(field?.translate).toBe(true);
  expect(field?.type).toBe('varchar');
  expect(field?.column?.size).toBeUndefined();
  expect(field?.column?.index).toBe('trigram');
  expect(field?.storageHints?.size).toBe(255);
});

test('auth.role: Description bilingual write/read unwraps by lang', async () => {
  const code = `t_${uid('role').replace(/[^a-zA-Z0-9._]/g, '').slice(0, 40)}`.slice(0, 50);
  const enDesc = uid('RoleDescEn');
  const zhDesc = uid('RoleDescZh');

  const created = await Role.Create(
    {
      Name: uid('RoleName'),
      Code: code,
      Description: { en_US: enDesc, zh_CN: zhDesc } as any,
      IsActive: true,
    } as any,
    ['Id', 'Description', 'Code'] as any
  );
  expect(String((created as any).Description)).toBe(enDesc);

  const id = String((created as any).Id);
  const zhBrowse = await withContext({ lang: 'zh_CN' }, () => Role.Browse(id, ['Id', 'Description'] as any));
  expect(String((zhBrowse as any).Description)).toBe(zhDesc);

  const hit = await withContext({ lang: 'zh_CN' }, () =>
    Role.Search(['Description', 'ilike', zhDesc] as any, { fields: ['Id', 'Code'], limit: 5 } as any)
  );
  expect(hit?.some((r: any) => String(r.Code) === code)).toBe(true);
});

test('auth.role: DisplayName stays SqlCompute (not translate)', () => {
  const field = MetadataStorage.instance.getModelMetadata(Role).fields.get('DisplayName');
  expect(field?.translate).toBeFalsy();
});

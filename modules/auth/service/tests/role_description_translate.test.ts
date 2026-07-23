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

function ensureRequestContext(): any {
  const root: any = (globalThis as any).$choysum;
  if (!root) {
    throw new Error('missing global $choysum; role translate tests must run under the QuickJS-first harness');
  }
  if (!root.request) root.request = {};
  if (!root.request.context) root.request.context = {};
  const jsCtx = root.request.context;
  if (!jsCtx.ctx) jsCtx.ctx = {};
  if (!jsCtx.req) jsCtx.req = {};
  return jsCtx;
}

/** Bypass record-rule default-deny so Role fixtures can be created in unit tests. */
function allowRoleWrites(): void {
  const jsCtx = ensureRequestContext();
  jsCtx.req = {
    ...(jsCtx.req || {}),
    depth: 0,
    fieldRuleMode: 'skip',
    recordRuleMode: 'allowlist',
    recordRuleAllow: [
      'auth.Role:read',
      'auth.Role:write',
      'auth.Role:create',
      'auth.Role:delete',
      'Role:read',
      'Role:write',
      'Role:create',
      'Role:delete',
    ],
  };
}

test('auth.role: Description translate metadata is enabled', () => {
  const field = MetadataStorage.instance.getModelMetadata(Role).fields.get('Description');
  expect(field?.translate).toBe(true);
  expect(field?.type).toBe('varchar');
  expect(field?.column?.size).toBeUndefined();
  expect(field?.column?.index).toBe('trigram');
  expect(field?.storageHints?.size).toBe(255);
});

test('auth.role: Name translate metadata is enabled (no unique)', () => {
  const field = MetadataStorage.instance.getModelMetadata(Role).fields.get('Name');
  expect(field?.translate).toBe(true);
  expect(field?.column?.unique).toBeFalsy();
  expect(field?.column?.index).toBe('trigram');
  expect(field?.storageHints?.size).toBe(100);
});

test('auth.role: Name/Description bilingual write/read unwraps by lang', async () => {
  allowRoleWrites();

  const code = `t_${uid('role').replace(/[^a-zA-Z0-9._]/g, '').slice(0, 40)}`.slice(0, 50);
  const enName = uid('RoleNameEn');
  const zhName = uid('RoleNameZh');
  const enDesc = uid('RoleDescEn');
  const zhDesc = uid('RoleDescZh');

  const created = await Role.Create(
    {
      Name: { en_US: enName, zh_CN: zhName } as any,
      Code: code,
      Description: { en_US: enDesc, zh_CN: zhDesc } as any,
      IsActive: true,
    } as any,
    ['Id', 'Name', 'Description', 'Code'] as any
  );
  expect(String((created as any).Name)).toBe(enName);
  expect(String((created as any).Description)).toBe(enDesc);

  const id = String((created as any).Id);
  const zhBrowse = await withContext({ lang: 'zh_CN' }, () =>
    Role.Browse(id, ['Id', 'Name', 'Description'] as any)
  );
  expect(String((zhBrowse as any).Name)).toBe(zhName);
  expect(String((zhBrowse as any).Description)).toBe(zhDesc);

  const hit = await withContext({ lang: 'zh_CN' }, () =>
    Role.Search(['Name', 'ilike', zhName] as any, { fields: ['Id', 'Code'], limit: 5 } as any)
  );
  expect(hit?.some((r: any) => String(r.Code) === code)).toBe(true);
});

test('auth.role: DisplayName stays SqlCompute (not translate)', () => {
  const field = MetadataStorage.instance.getModelMetadata(Role).fields.get('DisplayName');
  expect(field?.translate).toBeFalsy();
});

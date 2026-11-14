// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import User from '@/auth/service/models/user/user';
import { createServiceByModel } from '@/core/service/rpc';

function uid(prefix: string): string {
  return `${prefix}_${Date.now().toString(36)}_${Math.random().toString(36).slice(2, 8)}`;
}

function code8(): string {
  return Math.random().toString(36).slice(2, 10);
}

function withFieldRulesSkipped<T>(fn: () => Promise<T>): Promise<T> {
  const root: any = (globalThis as any).$choysum ?? {};
  if (!root.request) root.request = {};
  if (!root.request.context) root.request.context = {};
  const jsCtx = root.request.context;
  jsCtx.ctx = jsCtx.ctx || {};
  const prev = jsCtx.req;
  jsCtx.req = { depth: 0, fieldRuleMode: 'skip', ...(prev || {}) };
  (globalThis as any).$choysum = root;
  return fn().finally(() => {
    jsCtx.req = prev;
  });
}

test('auth.User FullName uses Language.PersonNameFormat last_first', async () => {
  await withFieldRulesSkipped(async () => {
    type Company = typeof import('@/base/service/models/company').default;
    type Language = typeof import('@/base/service/models/language').default;
    const CompanyService = createServiceByModel<Company>('base.Company');
    const LanguageService = createServiceByModel<Language>('base.Language');

    const companies = await (CompanyService as any).Search(['Code', '=', 'MAIN'] as any, {
      fields: ['Id'],
      limit: 1,
    } as any);
    const companyId = String(companies?.[0]?.Id || '').trim();
    expect(companyId).toBeTruthy();

    const langCode = `f_${code8()}`.slice(0, 16);
    const lang = await (LanguageService as any).Create(
      {
        Name: uid('FullNameLang'),
        Code: langCode,
        Direction: 'ltr',
        IsActive: true,
        PersonNameFormat: '{LastName}{FirstName}',
      } as any,
      ['Id'] as any
    );

    const username = `u_${code8()}`.slice(0, 20);
    const user = await User.Create(
      {
        Username: username,
        Email: `${username}@example.com`,
        PasswordHash: 'test',
        FirstName: '步科',
        LastName: '王',
        LanguageId: (lang as any).Id,
        CompanyId: companyId,
        CompanyIds: [companyId],
        IsActive: true,
      } as any,
      ['Id', 'FullName'] as any
    );

    expect(String((user as any).FullName || '')).toBe('王步科');
  });
});

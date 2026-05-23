// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ChoysumError } from '@/core/service/error';
import CompanyScopedResource from '@/auth/service/models/company_scoped_resource';
import UoMCategory from '@/base/service/models/uom_category';

function ensureRequestContext(): any {
  const root: any = (globalThis as any).$choysum ?? {};
  if (!root.request) root.request = {};
  if (!root.request.context) root.request.context = {};

  const jsCtx = root.request.context;
  if (!jsCtx.ctx) jsCtx.ctx = {};
  if (!jsCtx.req) jsCtx.req = {};
  if (!jsCtx.identity) jsCtx.identity = {};

  (globalThis as any).$choysum = root;
  return jsCtx;
}

function resetRequestContext(): void {
  const jsCtx = ensureRequestContext();
  jsCtx.ctx = {};
  jsCtx.req = {
    depth: 0,
    recordRuleMode: 'allowlist',
    fieldRuleMode: 'skip',
    recordRuleAllow: ['auth.CompanyScopedResource:read', 'CompanyScopedResource:read', 'base.UoMCategory:read', 'UoMCategory:read'],
  };
  jsCtx.identity = {};

  delete (jsCtx as any)[Symbol.for('choysum.ctx.override')];
  delete (jsCtx as any)[Symbol.for('choysum.ctx.frozen')];
}

async function expectInvalidArgumentDomain(fn: () => Promise<any>, expectedDomain: string): Promise<void> {
  try {
    await fn();
    throw new Error('expected InvalidArgument error');
  } catch (err) {
    expect(err instanceof ChoysumError).toBe(true);
    const oe = err as ChoysumError;
    expect(oe.code).toBe('InvalidArgument');
    expect(oe.domain).toBe(expectedDomain);
  }
}

test('auth.soft_delete_options: cross-app conflict domain matrix (Search/Count/ReadGroup)', async () => {
  resetRequestContext();

  const cases = [
    { model: UoMCategory as any, domain: 'base' },
    { model: CompanyScopedResource as any, domain: 'auth' },
  ];

  for (const c of cases) {
    await expectInvalidArgumentDomain(async () => {
      await c.model.Search([] as any, { withDeleted: true, onlyDeleted: true } as any);
    }, c.domain);

    await expectInvalidArgumentDomain(async () => {
      await c.model.Count([] as any, { withDeleted: true, onlyDeleted: true } as any);
    }, c.domain);

    await expectInvalidArgumentDomain(async () => {
      await c.model.ReadGroup(['Id'] as any, [] as any, { withDeleted: true, onlyDeleted: true } as any);
    }, c.domain);
  }
});

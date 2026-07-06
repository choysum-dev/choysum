// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ChoysumError } from '@/core/service/error';
import CompanyScopedResource from '@/auth/service/models/company_scoped_resource';
import { withContext as withModelContext } from '@/core/service/api/context';

function ensureRequestContext(): any {
  const root: any = (globalThis as any).$choysum ?? {};
  if (!root.request) root.request = {};
  let jsCtx: any;
  const getRequestContext = root.getRequestContext;
  if (typeof getRequestContext === 'function') {
    try {
      jsCtx = getRequestContext();
    } catch {
      jsCtx = undefined;
    }
  }
  if (!jsCtx || typeof jsCtx !== 'object') {
    if (!root.context || typeof root.context !== 'object') root.context = {};
    jsCtx = root.context;
  }

  // Use $choysum.context as the canonical jsCtx root (see runtime/context).
  root.context = jsCtx;
  root.request.context = jsCtx;
  if (!jsCtx.ctx) jsCtx.ctx = {};
  if (!jsCtx.req) jsCtx.req = {};
  if (!jsCtx.identity) jsCtx.identity = {};

  (globalThis as any).$choysum = root;
  return jsCtx;
}

function resetRequestContext(): void {
  const jsCtx = ensureRequestContext();
  jsCtx.ctx = {};
  // The test runner imports all test files before executing cases, so do not toggle
  // record or field rule behavior via import.meta.env inside a test file.
  // Use a request-level allowlist here so RecordRule always bypasses in this file,
  // keeping the focus on company-layer semantics.
  jsCtx.req = {
    depth: 0,
    recordRuleMode: 'allowlist',
    // This file does not test FieldRule. With FieldRule enabled, create and update would fetch
    // specs and fail because these company-isolation tests intentionally do not inject identity,
    // so use top-level skip to avoid unrelated failures.
    fieldRuleMode: 'skip',
    // Allowlist keys must be `${model}:${op}`; model may be the full or short model name.
    recordRuleAllow: ['auth.CompanyScopedResource:read', 'auth.CompanyScopedResource:create', 'CompanyScopedResource:read', 'CompanyScopedResource:create'],
  };
  jsCtx.identity = {};

  // Clear runtime/context caches and overrides to avoid cross-test leakage.
  delete (jsCtx as any)[Symbol.for('choysum.ctx.override')];
  delete (jsCtx as any)[Symbol.for('choysum.ctx.frozen')];
}

function uid(prefix: string): string {
  const xid = (globalThis as any).$choysum?.xid?.New?.();
  const u = typeof xid === 'string' && xid.trim() ? xid.trim() : String(Date.now());
  return `${prefix}_${u}`;
}

test('P2-1 company scope: search filters by companyIds + shared rows', async () => {
  resetRequestContext();

  // Test semantics depend only on CompanyId and ctx.enabledCompanyIds/activeCompanyId.
  const c1 = { Id: uid('C1') };
  const c2 = { Id: uid('C2') };

  // Create three rows under a full-scope view: shared / c1 / c2.
  await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id, c2.Id] } as any,
    async () => {
      await CompanyScopedResource.Create({ Name: uid('csr_shared'), CompanyId: null as any }, ['Id'] as any);
      await CompanyScopedResource.Create({ Name: uid('csr_c1'), CompanyId: c1.Id }, ['Id'] as any);
      await CompanyScopedResource.Create({ Name: uid('csr_c2'), CompanyId: c2.Id }, ['Id'] as any);
    },
    { merge: false }
  );

  // A c1-only view should see shared + c1.
  const c1Names = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      const rows = await CompanyScopedResource.Search([], { fields: ['Id', 'Name', 'CompanyId'] as any });
      return rows.map(r => (r as any).Name).filter((n: string) => String(n).includes('csr_'));
    },
    { merge: false }
  );
  expect(c1Names.some(n => n.includes('csr_shared'))).toBe(true);
  expect(c1Names.some(n => n.includes('csr_c1'))).toBe(true);
  expect(c1Names.some(n => n.includes('csr_c2'))).toBe(false);

  // A c2-only view should see shared + c2.
  const c2Names = await withModelContext(
    { activeCompanyId: c2.Id, enabledCompanyIds: [c2.Id] } as any,
    async () => {
      const rows = await CompanyScopedResource.Search([], { fields: ['Id', 'Name', 'CompanyId'] as any });
      return rows.map(r => (r as any).Name).filter((n: string) => String(n).includes('csr_'));
    },
    { merge: false }
  );
  expect(c2Names.some(n => n.includes('csr_shared'))).toBe(true);
  expect(c2Names.some(n => n.includes('csr_c2'))).toBe(true);
  expect(c2Names.some(n => n.includes('csr_c1'))).toBe(false);
});

test('P2-1 company scope: create defaults CompanyId to ctx.activeCompanyId', async () => {
  resetRequestContext();

  const c1 = { Id: uid('C1') };

  const locationId = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      const created = await CompanyScopedResource.Create({ Name: uid('csr_auto') }, ['Id'] as any);
      return created.Id;
    },
    { merge: false }
  );

  const reloaded = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      return await CompanyScopedResource.Browse(locationId, ['Id', 'Name', 'CompanyId'] as any);
    },
    { merge: false }
  );

  expect((reloaded as any).CompanyId).toBe(c1.Id);
});

test('P2-1 company scope: create with out-of-scope CompanyId is denied (fail-closed)', async () => {
  resetRequestContext();

  const c1 = { Id: uid('C1') };
  const c2 = { Id: uid('C2') };

  try {
    await withModelContext(
      { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
      async () => {
        await CompanyScopedResource.Create({ Name: uid('csr_bad'), CompanyId: c2.Id }, ['Id'] as any);
      },
      { merge: false }
    );
    throw new Error('expected create to throw');
  } catch (err) {
    expect(err instanceof ChoysumError).toBe(true);
    const oe = err as ChoysumError;
    expect(oe.domain).toBe('core.repository');
    expect(oe.code).toBe('company_scope_violation');
  }
});

test('P2-1 company scope: missing ctx.enabledCompanyIds/activeCompanyId is denied (fail-closed)', async () => {
  resetRequestContext();

  // merge=false keeps ctx empty so the default context cannot leak in.
  try {
    await withModelContext(
      {} as any,
      async () => {
        await CompanyScopedResource.Search([], { fields: ['Id'] as any });
      },
      { merge: false }
    );
    throw new Error('expected search to throw');
  } catch (err) {
    expect(err instanceof ChoysumError).toBe(true);
    const oe = err as ChoysumError;
    expect(oe.domain).toBe('core.repository');
    expect(oe.code).toBe('company_scope_missing_ctx_company');
  }
});

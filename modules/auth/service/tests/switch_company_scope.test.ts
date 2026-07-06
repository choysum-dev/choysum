// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ChoysumError } from '@/core/service/error';
import { withContext as withModelContext } from '@/core/service/api/context';
import User from '@/auth/service/models/user';

const RR_CACHE_KEY = Symbol.for('choysum.recordrule.cache');
const FR_CACHE_KEY = Symbol.for('choysum.fieldrule.cache');

function ensureRequestContext(): any {
  const root: any = (globalThis as any).$choysum;
  if (!root) {
    throw new Error('missing global $choysum; switch_company_scope tests must run under the QuickJS-first harness');
  }

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

  root.context = jsCtx;
  root.request.context = jsCtx;
  if (!jsCtx.ctx) jsCtx.ctx = {};
  if (!jsCtx.req) jsCtx.req = {};
  if (!jsCtx.identity) jsCtx.identity = {};

  return jsCtx;
}

function uid(prefix: string): string {
  const xid = (globalThis as any).$choysum?.xid?.New?.();
  const u = typeof xid === 'string' && xid.trim() ? xid.trim() : String(Date.now());
  return `${prefix}_${u}`;
}

function setIdentity(userId?: string): void {
  const jsCtx = ensureRequestContext();
  if (!jsCtx.identity) jsCtx.identity = {};
  if (userId) jsCtx.identity.userId = userId;
  else delete jsCtx.identity.userId;
}

function setReq(patch: Record<string, any>): void {
  const jsCtx = ensureRequestContext();
  if (!jsCtx.req) jsCtx.req = {};
  Object.assign(jsCtx.req, patch);
}

function resetRequestContext(): void {
  const jsCtx = ensureRequestContext();
  jsCtx.ctx = {};

  // SwitchCompanyScope tests do not focus on RecordRule or FieldRule.
  // - recordRule: use allowlist bypass so fixtures and Token.Create avoid default deny paths
  // - fieldRule: use skip so User.UpdateById and Token.Create are not blocked by field rules
  jsCtx.req = {
    depth: 0,
    recordRuleMode: 'allowlist',
    recordRuleAllow: [
      // auth.User
      'auth.User:read',
      'auth.User:write',
      'auth.User:create',
      'auth.User:delete',
      'User:read',
      'User:write',
      'User:create',
      'User:delete',

      // Permission graph (extractUserMetadata -> _computePermStateVersion)
      'auth.Role:read',
      'auth.Role:write',
      'auth.Role:create',
      'auth.Role:delete',
      'Role:read',
      'Role:write',
      'Role:create',
      'Role:delete',

      'auth.UserRole:read',
      'auth.UserRole:write',
      'auth.UserRole:create',
      'auth.UserRole:delete',
      'UserRole:read',
      'UserRole:write',
      'UserRole:create',
      'UserRole:delete',

      'auth.RoleInheritance:read',
      'auth.RoleInheritance:write',
      'auth.RoleInheritance:create',
      'auth.RoleInheritance:delete',
      'RoleInheritance:read',
      'RoleInheritance:write',
      'RoleInheritance:create',
      'RoleInheritance:delete',

      'auth.RoleMethodAccess:read',
      'auth.RoleMethodAccess:write',
      'auth.RoleMethodAccess:create',
      'auth.RoleMethodAccess:delete',
      'RoleMethodAccess:read',
      'RoleMethodAccess:write',
      'RoleMethodAccess:create',
      'RoleMethodAccess:delete',

      // auth.Token (CreateTokenPair writes two Token rows)
      'auth.Token:read',
      'auth.Token:write',
      'auth.Token:create',
      'auth.Token:delete',
      'Token:read',
      'Token:write',
      'Token:create',
      'Token:delete',
    ],
    fieldRuleMode: 'skip',
  };
  jsCtx.identity = {};

  delete (jsCtx as any)[Symbol.for('choysum.ctx.override')];
  delete (jsCtx as any)[Symbol.for('choysum.ctx.frozen')];
  delete (jsCtx as any)[RR_CACHE_KEY];
  delete (jsCtx as any)[FR_CACHE_KEY];
}

function toChoysumErrorLike(err: any): { domain?: string; code?: string; message?: string; grpcCode?: number } | null {
  if (!err) return null;
  if (err instanceof ChoysumError) {
    return { domain: err.domain, code: String(err.code), message: err.message, grpcCode: (err as any).grpcCode };
  }

  const msg = typeof err === 'string' ? err : typeof err?.message === 'string' ? err.message : '';
  if (!msg) return null;
  const m = msg.match(/\[(?<domain>[^\]]+)\]\s+(?<code>[^:\s]+):\s+(?<message>[\s\S]*)$/);
  if (m?.groups?.domain || m?.groups?.code) {
    return { domain: m.groups?.domain, code: m.groups?.code, message: m.groups?.message };
  }
  return { message: msg };
}

function readTokenMetadata(token: string, tokenType: 'access' | 'refresh' = 'access'): any {
  const root: any = (globalThis as any).$choysum;
  const id = root?.auth?.validateToken?.(token, tokenType, false);
  const md = (id as any)?.metadata ?? (id as any)?.claims?.metadata ?? (id as any)?.payload?.metadata ?? (id as any)?.payload?.claims?.metadata ?? {};
  return md && typeof md === 'object' ? md : {};
}

async function createUser(params: { companyId: string; companyIds?: string[]; preferences?: any }): Promise<string> {
  const created = await User.Create(
    {
      Username: uid('u'),
      PasswordHash: 'test',
      FirstName: 'T',
      LastName: 'U',
      CompanyId: params.companyId,
      // NOTE: writing multiple company_ids under sqlite can trigger "row value misused".
      // This test only relies on a single-company allowlist (CompanyId) and uses a synthetic companyId
      // to cover the fail-closed validation path.
      CompanyIds: (params.companyIds && params.companyIds.length ? params.companyIds : [params.companyId]).slice(0, 1),
      Preferences: params.preferences ?? {},
      IsActive: true,
    } as any,
    ['Id'] as any
  );
  return created.Id;
}

test('SwitchCompanyScope: enabled omitted defaults to [active] when prefs empty; writes prefs + updates token metadata', async () => {
  resetRequestContext();

  setReq({ traceId: 'trace_switch_company_scope_ok' });

  const auditLines: string[] = [];
  const prevConsoleInfo = console.info;
  console.info = (...args: any[]) => {
    auditLines.push(args.map(a => (typeof a === 'string' ? a : JSON.stringify(a))).join(' '));
    return undefined as any;
  };

  const c1 = { Id: uid('C1') };

  let userId = '';
  try {
    const tokens = await withModelContext(
      { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
      async () => {
        userId = await createUser({ companyId: c1.Id, companyIds: [c1.Id], preferences: {} });
        setIdentity(userId);

        // omit enabledCompanyIds
        try {
          return await User.SwitchCompanyScope(c1.Id);
        } catch (err) {
          if (err instanceof ChoysumError) throw new Error(err.toString());
          throw err;
        }
      },
      { merge: false }
    );

    const md = readTokenMetadata(tokens.accessToken, 'access');
    expect(md.activeCompanyId).toBe(c1.Id);
    expect(Array.isArray(md.enabledCompanyIds)).toBe(true);
    expect((md.enabledCompanyIds || []).includes(c1.Id)).toBe(true);
    expect(Array.isArray(md.allowedCompanyIds)).toBe(true);
    expect((md.allowedCompanyIds || []).includes(c1.Id)).toBe(true);

    // audit
    const line = auditLines.find(l => typeof l === 'string' && l.startsWith('[AUDIT] '));
    expect(!!line).toBe(true);
    if (line) {
      const json = line.slice('[AUDIT] '.length);
      const evt: any = JSON.parse(json);
      expect(evt.event).toBe('auth.user.switch_company_scope');
      expect(evt.ok).toBe(true);
      expect(evt.traceId).toBe('trace_switch_company_scope_ok');
      expect(evt.userId).toBe(userId);
      expect(evt.targetActive).toBe(c1.Id);
      expect(evt.targetEnabled).toEqual([c1.Id]);
    }
  } finally {
    console.info = prevConsoleInfo;
  }
});

test('SwitchCompanyScope: preserves other Preferences fields', async () => {
  resetRequestContext();

  const c1 = { Id: uid('C1') };

  const userId = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      return await createUser({
        companyId: c1.Id,
        companyIds: [c1.Id],
        preferences: { theme: 'dark', density: 'compact' },
      });
    },
    { merge: false }
  );

  const tokens = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      setIdentity(userId);
      try {
        return await User.SwitchCompanyScope(c1.Id, [c1.Id]);
      } catch (err) {
        if (err instanceof ChoysumError) throw new Error(err.toString());
        throw err;
      }
    },
    { merge: false }
  );

  // prefs persisted
  const reloaded = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      return await User.Browse(userId, ['Id', 'Preferences'] as any);
    },
    { merge: false }
  );

  const prefs: any = (reloaded as any).Preferences || {};
  expect(prefs.theme).toBe('dark');
  expect(prefs.density).toBe('compact');
  expect(prefs.activeCompanyId).toBe(c1.Id);
  expect(prefs.enabledCompanyIds).toEqual([c1.Id]);

  const md = readTokenMetadata(tokens.accessToken, 'access');
  expect(md.activeCompanyId).toBe(c1.Id);
  expect(md.enabledCompanyIds).toEqual([c1.Id]);
});

test('SwitchCompanyScope: enabledCompanyIds must be string[] or omitted (fail-closed)', async () => {
  resetRequestContext();

  const c1 = { Id: uid('C1') };

  try {
    await withModelContext(
      { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
      async () => {
        const userId = await createUser({ companyId: c1.Id, companyIds: [c1.Id] });
        setIdentity(userId);
        // invalid type
        await User.SwitchCompanyScope(c1.Id, 'bad' as any);
      },
      { merge: false }
    );
    throw new Error('expected SwitchCompanyScope to throw');
  } catch (err) {
    const oe = toChoysumErrorLike(err);
    expect(oe?.domain).toBe('auth');
    expect(oe?.code).toBe('VALIDATION_FAILED');
  }
});

test('SwitchCompanyScope: enabled ⊆ allowed (fail-closed)', async () => {
  resetRequestContext();
  setReq({ traceId: 'trace_switch_company_scope_enabled_subset_fail' });

  const c1 = { Id: uid('C1') };
  const bad = uid('C_BAD');
  const auditWarnLines: string[] = [];
  const prevConsoleWarn = console.warn;
  console.warn = (...args: any[]) => {
    auditWarnLines.push(args.map(a => (typeof a === 'string' ? a : JSON.stringify(a))).join(' '));
    return undefined as any;
  };

  try {
    await withModelContext(
      { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
      async () => {
        const userId = await createUser({ companyId: c1.Id, companyIds: [c1.Id] });
        setIdentity(userId);
        await User.SwitchCompanyScope(c1.Id, [c1.Id, bad]);
      },
      { merge: false }
    );
    throw new Error('expected SwitchCompanyScope to throw');
  } catch (err) {
    const oe = toChoysumErrorLike(err);
    expect(oe?.domain).toBe('auth');
    expect(oe?.code).toBe('VALIDATION_FAILED');

    const line = auditWarnLines.find(l => typeof l === 'string' && l.startsWith('[AUDIT] '));
    expect(!!line).toBe(true);
    if (line) {
      const json = line.slice('[AUDIT] '.length);
      const evt: any = JSON.parse(json);
      expect(evt.event).toBe('auth.user.switch_company_scope');
      expect(evt.ok).toBe(false);
      expect(evt.traceId).toBe('trace_switch_company_scope_enabled_subset_fail');
      expect(evt.targetActive).toBe(c1.Id);
      expect(evt.targetEnabled).toEqual([c1.Id, bad]);
      expect(String(evt.reason || '')).toContain('enabledCompanyIds');
      expect(evt.companyId).toBe(bad);
    }
  } finally {
    console.warn = prevConsoleWarn;
  }
});

test('SwitchCompanyScope: activeCompanyId must be in allowed (fail-closed)', async () => {
  resetRequestContext();

  const c1 = { Id: uid('C1') };
  const badActive = uid('C_BAD_ACTIVE');

  try {
    await withModelContext(
      { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
      async () => {
        const userId = await createUser({ companyId: c1.Id, companyIds: [c1.Id] });
        setIdentity(userId);
        await User.SwitchCompanyScope(badActive, [c1.Id]);
      },
      { merge: false }
    );
    throw new Error('expected SwitchCompanyScope to throw');
  } catch (err) {
    const oe = toChoysumErrorLike(err);
    expect(oe?.domain).toBe('auth');
    expect(oe?.code).toBe('VALIDATION_FAILED');
  }
});

test('SwitchCompanyScope: activeCompanyId must be in enabledCompanyIds (fail-closed)', async () => {
  resetRequestContext();

  const c1 = { Id: uid('C1') };

  try {
    await withModelContext(
      { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
      async () => {
        const userId = await createUser({ companyId: c1.Id, companyIds: [c1.Id] });
        setIdentity(userId);
        await User.SwitchCompanyScope(c1.Id, []);
      },
      { merge: false }
    );
    throw new Error('expected SwitchCompanyScope to throw');
  } catch (err) {
    const oe = toChoysumErrorLike(err);
    expect(oe?.domain).toBe('auth');
    expect(oe?.code).toBe('VALIDATION_FAILED');
  }
});

test('RefreshTokens: preserves active/enabled company scope after SwitchCompanyScope', async () => {
  resetRequestContext();

  const c1 = { Id: uid('C1') };
  const c2 = { Id: uid('C2') };

  const tokens = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      const userId = await createUser({ companyId: c1.Id, companyIds: [c2.Id], preferences: {} });
      setIdentity(userId);

      // Switch to c2 as active, with [c1, c2] enabled (both must be allowed).
      return await User.SwitchCompanyScope(c2.Id, [c1.Id, c2.Id]);
    },
    { merge: false }
  );

  const md1 = readTokenMetadata(tokens.accessToken, 'access');
  expect(md1.activeCompanyId).toBe(c2.Id);
  expect(Array.isArray(md1.enabledCompanyIds)).toBe(true);
  expect((md1.enabledCompanyIds || []).includes(c1.Id)).toBe(true);
  expect((md1.enabledCompanyIds || []).includes(c2.Id)).toBe(true);
  expect(Array.isArray(md1.allowedCompanyIds)).toBe(true);
  expect((md1.allowedCompanyIds || []).includes(c1.Id)).toBe(true);
  expect((md1.allowedCompanyIds || []).includes(c2.Id)).toBe(true);

  const refreshed = await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      return await User.RefreshTokens(tokens.refreshToken);
    },
    { merge: false }
  );

  const md2 = readTokenMetadata(refreshed.accessToken, 'access');
  expect(md2.activeCompanyId).toBe(c2.Id);
  expect(Array.isArray(md2.enabledCompanyIds)).toBe(true);
  expect((md2.enabledCompanyIds || []).includes(c1.Id)).toBe(true);
  expect((md2.enabledCompanyIds || []).includes(c2.Id)).toBe(true);
});

test('User.UpdateById: CompanyId must stay inside enabledCompanyIds (validation_failed)', async () => {
  resetRequestContext();

  const c1 = { Id: uid('C1') };
  const c2 = { Id: uid('C2') };

  let error: unknown;
  await withModelContext(
    { activeCompanyId: c1.Id, enabledCompanyIds: [c1.Id] } as any,
    async () => {
      const userId = await createUser({ companyId: c1.Id, companyIds: [c1.Id] });

      try {
        await User.UpdateById(
          userId,
          {
            CompanyId: c2.Id,
          } as any,
          ['Id', 'CompanyId'] as any
        );
      } catch (err) {
        error = err;
      }
    },
    { merge: false }
  );

  expect(error instanceof ChoysumError).toBe(true);
  const oe = error as ChoysumError;
  expect(oe.domain).toBe('core.repository');
  expect(oe.code).toBe('validation_failed');
  expect(oe.metadata?.field).toBe('CompanyId');
  expect(oe.metadata?.issueCode).toBe('platform_cross_company_reference_violation');

  const fieldIssues = JSON.parse(String(oe.metadata?.fieldIssues || '{}')) as Record<string, any[]>;
  expect(Array.isArray(fieldIssues.CompanyId)).toBe(true);
  expect(fieldIssues.CompanyId?.[0]?.code).toBe('platform_cross_company_reference_violation');
});

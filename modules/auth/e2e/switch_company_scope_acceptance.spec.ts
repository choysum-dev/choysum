// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { test, expect, page, runtime } from '@choysum/e2e';
import { createClient, type Interceptor, ConnectError, Code } from '@connectrpc/connect';
import { createGrpcWebTransport } from '@connectrpc/connect-web';
import { create } from '@bufbuild/protobuf';
import { ValueSchema, ListValueSchema, StructSchema, NullValue, type Value } from '@bufbuild/protobuf/wkt';
import { loginAsE2EAdmin } from './utils/login.ts';
import { waitForGrpcWebUnaryOk } from './utils/grpcweb.ts';

type AuthPbModule = {
  User: any;
  UserSwitchCompanyScopeReqSchema: any;
  UserRefreshTokensReqSchema: any;
};

let authPbModulePromise: Promise<AuthPbModule> | null = null;

async function loadAuthPbModule(): Promise<AuthPbModule> {
  // Bundle plugin remaps *_pb.ts to <runDir>/.choysum/generated/...
  const mod = (await import(
    /* @vite-ignore */ './.generated/auth_pb.ts' as string
  )) as AuthPbModule;
  return mod;
}

async function getAuthPbModule(): Promise<AuthPbModule> {
  if (!authPbModulePromise) {
    authPbModulePromise = loadAuthPbModule();
  }
  return await authPbModulePromise;
}

function getServerLogPath(): string {
  const runDir = String((runtime as any).runDir || '').trim();
  if (!runDir) throw new Error('runtime.runDir is not set');
  return `${runDir.replace(/\/$/, '')}/server.log`;
}

async function readTextFile(path: string): Promise<string> {
  const host = (globalThis as any).__choysum_e2e_host__;
  if (!host || typeof host.readTextFile !== 'function') {
    throw new Error('@choysum/e2e: host.readTextFile is not available');
  }
  return String(await host.readTextFile(path));
}

async function readAuthTokens(): Promise<{ accessToken: string; refreshToken: string }> {
  return page.evaluate(() => {
    const raw = localStorage.getItem('choysum.auth') || sessionStorage.getItem('choysum.auth');
    if (!raw) return { accessToken: '', refreshToken: '' };
    try {
      const data = JSON.parse(raw);
      return {
        accessToken: String(data?.tokens?.accessToken || ''),
        refreshToken: String(data?.tokens?.refreshToken || ''),
      };
    } catch {
      return { accessToken: '', refreshToken: '' };
    }
  });
}

async function readAuthState(): Promise<{ accessToken: string; refreshToken: string; identity: any }> {
  return page.evaluate(() => {
    const raw = localStorage.getItem('choysum.auth') || sessionStorage.getItem('choysum.auth');
    if (!raw) return { accessToken: '', refreshToken: '', identity: null };
    try {
      const data = JSON.parse(raw);
      return {
        accessToken: String(data?.tokens?.accessToken || ''),
        refreshToken: String(data?.tokens?.refreshToken || ''),
        identity: data?.identity ?? null,
      };
    } catch {
      return { accessToken: '', refreshToken: '', identity: null };
    }
  });
}

function decodeJwtPayload(token: string): any {
  const parts = String(token || '').split('.');
  if (parts.length < 2) return null;
  const b64url = parts[1];
  const b64 = b64url.replace(/-/g, '+').replace(/_/g, '/');
  const pad = b64.length % 4 === 0 ? '' : '='.repeat(4 - (b64.length % 4));
  const bin = atob(b64 + pad);
  let json = '';
  for (let i = 0; i < bin.length; i++) json += String.fromCharCode(bin.charCodeAt(i));
  try {
    return JSON.parse(json);
  } catch {
    return null;
  }
}

function extractCompanyScopeFromToken(accessToken: string): { activeCompanyId: string; enabledCompanyIds: string[] } {
  const payload = decodeJwtPayload(accessToken);
  const normalizeId = (v: any) => String(v ?? '').trim();
  const uniq = (xs: string[]) => Array.from(new Set(xs.map(normalizeId).filter(Boolean)));

  const search = (node: any): { active?: any; enabled?: any } => {
    if (!node || typeof node !== 'object') return {};
    if ('activeCompanyId' in node || 'enabledCompanyIds' in node) {
      return { active: (node as any).activeCompanyId, enabled: (node as any).enabledCompanyIds };
    }
    for (const key of ['meta', 'metadata', 'identity', 'claims', 'data']) {
      if (node && typeof node[key] === 'object') {
        const hit = search(node[key]);
        if (hit.active !== undefined || hit.enabled !== undefined) return hit;
      }
    }
    for (const v of Object.values(node)) {
      const hit = search(v);
      if (hit.active !== undefined || hit.enabled !== undefined) return hit;
    }
    return {};
  };

  const hit = search(payload);
  const activeCompanyId = normalizeId(hit.active);
  const enabledCompanyIds = Array.isArray(hit.enabled) ? uniq(hit.enabled) : [];
  return { activeCompanyId, enabledCompanyIds };
}

function toValue(val: any): Value {
  if (val === null || val === undefined) {
    return create(ValueSchema, {
      kind: { case: 'nullValue', value: NullValue.NULL_VALUE },
    });
  }
  if (typeof val === 'string') {
    return create(ValueSchema, { kind: { case: 'stringValue', value: val } });
  }
  if (typeof val === 'number') {
    return create(ValueSchema, { kind: { case: 'numberValue', value: val } });
  }
  if (typeof val === 'boolean') {
    return create(ValueSchema, { kind: { case: 'boolValue', value: val } });
  }
  if (Array.isArray(val)) {
    const values = val.map(item => toValue(item));
    return create(ValueSchema, {
      kind: {
        case: 'listValue',
        value: create(ListValueSchema, { values }),
      },
    });
  }
  if (typeof val === 'object') {
    const fields: Record<string, any> = {};
    for (const [k, v] of Object.entries(val)) {
      fields[k] = toValue(v);
    }
    return create(ValueSchema, {
      kind: {
        case: 'structValue',
        value: create(StructSchema, { fields }),
      },
    });
  }
  return create(ValueSchema, {
    kind: { case: 'nullValue', value: NullValue.NULL_VALUE },
  });
}

function fromValue(v?: Value): any {
  const toJs = (x: any): any => {
    if (!x || typeof x !== 'object' || !x.kind) return x;
    switch (x.kind.case) {
      case 'nullValue':
        return null;
      case 'stringValue':
      case 'numberValue':
      case 'boolValue':
        return x.kind.value;
      case 'listValue': {
        const arr = x.kind.value?.values ?? [];
        return arr.map((it: any) => toJs(it));
      }
      case 'structValue': {
        const fields = x.kind.value?.fields ?? {};
        const obj: Record<string, any> = {};
        for (const [k, vv] of Object.entries(fields)) obj[k] = toJs(vv);
        return obj;
      }
      default:
        return null;
    }
  };
  return toJs(v);
}

function makeAuthInterceptor(accessToken: string): Interceptor {
  return next => async req => {
    if (accessToken) {
      req.header.set('Authorization', `Bearer ${accessToken}`);
    }
    return await next(req);
  };
}

function makeUserClient(baseURL: string, accessToken: string, userService: any): any {
  const transport = createGrpcWebTransport({
    baseUrl: baseURL,
    interceptors: [makeAuthInterceptor(accessToken)],
  });
  return createClient(userService as any, transport) as any;
}

async function waitForServerLogContains(needle: string, timeoutMs = 10_000): Promise<void> {
  const logPath = getServerLogPath();
  const escapedNeedle = (() => {
    try {
      return JSON.stringify(needle).slice(1, -1);
    } catch {
      return needle;
    }
  })();
  const start = Date.now();
  while (Date.now() - start < timeoutMs) {
    try {
      const raw = await readTextFile(logPath);
      if (raw.includes(needle) || (escapedNeedle !== needle && raw.includes(escapedNeedle))) return;
    } catch {
      // ignore
    }
    await page.waitForTimeout(200);
  }
  const tail = (() => {
    return readTextFile(logPath)
      .then(raw => raw.slice(-32_000))
      .catch(() => '');
  })();
  throw new Error(`timeout waiting for server.log to contain: ${needle}\n--- server.log tail ---\n${await tail}`);
}

async function switchCompanyViaUI(): Promise<void> {
  const trigger = page.getByTestId('company-switch-trigger');
  await expect(trigger).toBeVisible();

  await trigger.click();
  await page.getByTestId('company-active-select').click();

  await expect
    .poll(
      async () =>
        page.evaluate(() => {
          const opts = Array.from(
            document.querySelectorAll('.el-select-dropdown li, [role="option"]')
          ) as HTMLElement[];
          if (opts.length < 2) return false;
          const target =
            opts.find(o => o.getAttribute('aria-selected') !== 'true') || opts[opts.length - 1];
          target.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true, view: window }));
          return true;
        }),
      { timeout: 10_000 }
    )
    .toBe(true);

  const applyButton = page.getByTestId('company-switch-apply');
  await expect.poll(async () => await applyButton.isEnabled(), { timeout: 15_000 }).toBe(true);
  await expect(page.getByTestId('company-switch-hint')).toHaveCount(0);

  const switchOk = waitForGrpcWebUnaryOk(page, '/auth.User/SwitchCompanyScope', { timeoutMs: 30_000 });
  await applyButton.click();
  try {
    await switchOk;
  } catch {
    // One retry: Element Plus sometimes swallows the first apply click.
    const retryOk = waitForGrpcWebUnaryOk(page, '/auth.User/SwitchCompanyScope', { timeoutMs: 30_000 });
    await applyButton.click();
    await retryOk;
  }
}

async function discoverTwoCompanyIdsByUISwitch(): Promise<{ a: string; b: string }> {
  const trigger = page.getByTestId('company-switch-trigger');
  await expect(trigger).toBeVisible();

  const before = await readAuthTokens();
  if (!before.accessToken) {
    throw new Error('discoverTwoCompanyIdsByUISwitch: missing access token before switch');
  }
  const scopeA = extractCompanyScopeFromToken(before.accessToken);
  if (!scopeA.activeCompanyId) {
    throw new Error('discoverTwoCompanyIdsByUISwitch: missing activeCompanyId before switch');
  }

  await switchCompanyViaUI();

  await expect
    .poll(
      async () => {
        const after = await readAuthTokens();
        const next = extractCompanyScopeFromToken(after.accessToken).activeCompanyId;
        return next && next !== scopeA.activeCompanyId ? next : scopeA.activeCompanyId;
      },
      { timeout: 30_000 }
    )
    .not.toBe(scopeA.activeCompanyId);

  const after = await readAuthTokens();
  if (!after.accessToken) {
    throw new Error('discoverTwoCompanyIdsByUISwitch: missing access token after switch');
  }
  const scopeB = extractCompanyScopeFromToken(after.accessToken);
  if (!scopeB.activeCompanyId) {
    throw new Error('discoverTwoCompanyIdsByUISwitch: missing activeCompanyId after switch');
  }
  if (scopeB.activeCompanyId === scopeA.activeCompanyId) {
    throw new Error('discoverTwoCompanyIdsByUISwitch: company id did not change after UI switch');
  }

  return { a: scopeA.activeCompanyId, b: scopeB.activeCompanyId };
}

function expectIncludesAll(actual: string[], want: string[]) {
  for (const id of want) {
    expect(actual.includes(id)).toBe(true);
  }
}

test('auth: SwitchCompanyScope default enabled uses Preferences (enabledCompanyIds omitted)', async () => {
  test.setTimeout(120_000);

  const baseURL = runtime.baseURL;
  const authPb = await getAuthPbModule();

  await loginAsE2EAdmin(page, baseURL);

  const { accessToken } = await readAuthState();
  expect(accessToken).not.toBe('');

  const pair = await discoverTwoCompanyIdsByUISwitch();

  const client0: any = makeUserClient(baseURL, accessToken, authPb.User);

  const r1: any = await (client0 as any).switchCompanyScope(
    create(authPb.UserSwitchCompanyScopeReqSchema, {
      activeCompanyId: pair.a,
      enabledCompanyIds: toValue([pair.a, pair.b]),
    })
  );
  const tokenPair1 = fromValue(r1.result);
  expect(!!tokenPair1?.accessToken).toBe(true);

  const payload1 = decodeJwtPayload(String(tokenPair1.accessToken));
  expect(!!payload1).toBe(true);
  const scope1 = extractCompanyScopeFromToken(String(tokenPair1.accessToken));
  expect(scope1.activeCompanyId).toBe(pair.a);
  expectIncludesAll(scope1.enabledCompanyIds, [pair.a, pair.b]);

  const client1: any = makeUserClient(baseURL, String(tokenPair1.accessToken), authPb.User);
  const r2: any = await (client1 as any).switchCompanyScope(
    create(authPb.UserSwitchCompanyScopeReqSchema, {
      activeCompanyId: pair.b,
    })
  );
  const tokenPair2 = fromValue(r2.result);
  expect(!!tokenPair2?.accessToken).toBe(true);

  const payload2 = decodeJwtPayload(String(tokenPair2.accessToken));
  expect(!!payload2).toBe(true);
  const scope2 = extractCompanyScopeFromToken(String(tokenPair2.accessToken));
  expect(scope2.activeCompanyId).toBe(pair.b);
  expectIncludesAll(scope2.enabledCompanyIds, [pair.a, pair.b]);
});

test('auth: SwitchCompanyScope persists view; RefreshTokens reproduces the same active/enabled', async () => {
  test.setTimeout(120_000);

  const baseURL = runtime.baseURL;
  const authPb = await getAuthPbModule();

  await loginAsE2EAdmin(page, baseURL);

  const { accessToken } = await readAuthState();
  expect(accessToken).not.toBe('');

  const pair = await discoverTwoCompanyIdsByUISwitch();

  const client0: any = makeUserClient(baseURL, accessToken, authPb.User);

  const r1: any = await (client0 as any).switchCompanyScope(
    create(authPb.UserSwitchCompanyScopeReqSchema, {
      activeCompanyId: pair.b,
      enabledCompanyIds: toValue([pair.a, pair.b]),
    })
  );
  const tokenPair1 = fromValue(r1.result);
  expect(!!tokenPair1?.refreshToken).toBe(true);

  const client1: any = makeUserClient(baseURL, String(tokenPair1.accessToken), authPb.User);
  const r2: any = await (client1 as any).refreshTokens(
    create(authPb.UserRefreshTokensReqSchema, {
      refreshToken: String(tokenPair1.refreshToken),
    })
  );
  const tokenPair2 = fromValue(r2.result);
  expect(!!tokenPair2?.accessToken).toBe(true);

  const payload2 = decodeJwtPayload(String(tokenPair2.accessToken));
  expect(!!payload2).toBe(true);
  const scope2 = extractCompanyScopeFromToken(String(tokenPair2.accessToken));
  expect(scope2.activeCompanyId).toBe(pair.b);
  expectIncludesAll(scope2.enabledCompanyIds, [pair.a, pair.b]);
});

test('auth: SwitchCompanyScope illegal enabledCompanyIds fails closed and emits audit log', async () => {
  test.setTimeout(120_000);

  const baseURL = runtime.baseURL;
  const authPb = await getAuthPbModule();

  await loginAsE2EAdmin(page, baseURL);

  const { accessToken, identity } = await readAuthState();
  expect(accessToken).not.toBe('');

  const userId = String(identity?.userId ?? '');
  const pair = await discoverTwoCompanyIdsByUISwitch();

  const client: any = makeUserClient(baseURL, accessToken, authPb.User);

  const illegalCompanyId = 'e2e-illegal-company-not-allowed';

  let err: any = null;
  try {
    await (client as any).switchCompanyScope(
      create(authPb.UserSwitchCompanyScopeReqSchema, {
        activeCompanyId: pair.a,
        enabledCompanyIds: toValue([pair.a, illegalCompanyId]),
      })
    );
  } catch (e) {
    err = e;
  }

  expect(!!err).toBe(true);
  const errCode = Number((err as any)?.code);
  const errMessage = String((err as any)?.rawMessage || (err as any)?.message || '');
  expect(err instanceof ConnectError || Number.isFinite(errCode)).toBe(true);
  expect(errCode).toBe(Code.InvalidArgument);
  expect(errMessage).toContain('enabledCompanyIds');

  await waitForServerLogContains('auth.user.switch_company_scope', 15_000);
  await waitForServerLogContains('"ok":false', 15_000);
  await waitForServerLogContains('enabledCompanyIds', 15_000);
  if (userId) {
    await waitForServerLogContains(userId, 15_000);
  }
});

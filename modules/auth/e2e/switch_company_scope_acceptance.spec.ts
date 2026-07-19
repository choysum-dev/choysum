// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { test, expect } from '@playwright/test';
import fs from 'node:fs';
import path from 'node:path';
import { createClient, type Interceptor, ConnectError, Code } from '@connectrpc/connect';
import { createGrpcWebTransport } from '@connectrpc/connect-web';
import { create } from '@bufbuild/protobuf';
import { ValueSchema, ListValueSchema, StructSchema, NullValue, type Value } from '@bufbuild/protobuf/wkt';

type RuntimeInfo = {
  baseURL: string;
  specsDir: string;
  module: string;
  scenario: string;
  fixtures: string[];
  configPath?: string;
};

type AuthPbModule = {
  User: any;
  UserSwitchCompanyScopeReqSchema: any;
  UserRefreshTokensReqSchema: any;
};

let authPbModulePromise: Promise<AuthPbModule> | null = null;

function readRuntimeInfo(): RuntimeInfo {
  const runtimePath = process.env.CHOYSUM_E2E_RUNTIME_JSON;
  if (!runtimePath) {
    throw new Error('CHOYSUM_E2E_RUNTIME_JSON env var not set');
  }
  const raw = fs.readFileSync(runtimePath, 'utf-8');
  return JSON.parse(raw) as RuntimeInfo;
}

async function loadAuthPbModule(): Promise<AuthPbModule> {
  const runtime = readRuntimeInfo();
  // E2E runner stages generated pb under specsDir/.generated so Playwright
  // transforms it under testDir (absolute file:// imports bypass that and break
  // @bufbuild/protobuf/codegenv2 under PW_DISABLE_TS_ESM=1).
  const staged = path.join(runtime.specsDir, '.generated', 'auth_pb.ts');
  if (!fs.existsSync(staged)) {
    throw new Error(`Cannot find staged auth_pb.ts at ${staged} (e2e runner should link it)`);
  }
  const mod = await import('./.generated/auth_pb.ts');
  return mod as AuthPbModule;
}

async function getAuthPbModule(): Promise<AuthPbModule> {
  if (!authPbModulePromise) {
    authPbModulePromise = loadAuthPbModule();
  }
  return await authPbModulePromise;
}

function getServerLogPath(): string {
  const runtimePath = process.env.CHOYSUM_E2E_RUNTIME_JSON;
  if (!runtimePath) throw new Error('CHOYSUM_E2E_RUNTIME_JSON env var not set');
  return path.join(path.dirname(runtimePath), 'server.log');
}

async function readAuthTokens(page: any): Promise<{ accessToken: string; refreshToken: string }> {
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

async function readAuthState(page: any): Promise<{ accessToken: string; refreshToken: string; identity: any }> {
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
  const json = Buffer.from(b64 + pad, 'base64').toString('utf-8');
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

    // Direct hit
    if ('activeCompanyId' in node || 'enabledCompanyIds' in node) {
      return { active: (node as any).activeCompanyId, enabled: (node as any).enabledCompanyIds };
    }

    // Common nesting patterns
    for (const key of ['metadata', 'identity', 'claims', 'data']) {
      if (node && typeof node[key] === 'object') {
        const hit = search(node[key]);
        if (hit.active !== undefined || hit.enabled !== undefined) return hit;
      }
    }

    // Fallback: shallow walk
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

async function loginAsE2EAdmin(page: any, baseURL: string): Promise<void> {
  await page.goto(`${baseURL}/web/auth/users`, { waitUntil: 'domcontentloaded' });

  await page.waitForSelector('input[placeholder*="username"]', { timeout: 10_000 });
  await page.getByPlaceholder(/username/i).fill('e2e-admin');
  await page.getByPlaceholder(/password/i).fill('e2e-admin');
  await page.locator('button[type="submit"]').click();
  await expect(page).toHaveURL(/\/web\/auth\/users/, { timeout: 15_000 });
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
      const raw = fs.readFileSync(logPath, 'utf-8');
      if (raw.includes(needle) || (escapedNeedle !== needle && raw.includes(escapedNeedle))) return;
    } catch {
      // ignore
    }
    await new Promise(r => setTimeout(r, 200));
  }
  const tail = (() => {
    try {
      const raw = fs.readFileSync(logPath, 'utf-8');
      return raw.slice(-32_000);
    } catch {
      return '';
    }
  })();
  throw new Error(`timeout waiting for server.log to contain: ${needle}\n--- server.log tail ---\n${tail}`);
}

async function switchCompanyViaUI(page: any): Promise<void> {
  const trigger = page.getByTestId('company-switch-trigger');
  await expect(trigger).toBeVisible();

  await trigger.click();

  await page.getByTestId('company-active-select').click();
  const options = page.getByRole('option');
  await expect
    .poll(async () => {
      return await options.count();
    })
    .toBeGreaterThanOrEqual(2);
  const count = await options.count();
  for (let i = 0; i < count; i++) {
    const opt = options.nth(i);
    const selected = await opt.getAttribute('aria-selected');
    if (selected === 'true') continue;
    await opt.click();
    break;
  }

  const applyButton = page.getByTestId('company-switch-apply');
  await expect(applyButton).toBeEnabled();
  await applyButton.click();
}

async function discoverTwoCompanyIdsByUISwitch(page: any): Promise<{ a: string; b: string } | null> {
  const trigger = page.getByTestId('company-switch-trigger');
  await expect(trigger).toBeVisible();

  const before = await readAuthTokens(page);
  if (!before.accessToken) return null;
  const scopeA = extractCompanyScopeFromToken(before.accessToken);
  if (!scopeA.activeCompanyId) return null;

  await switchCompanyViaUI(page);

  await expect
    .poll(
      async () => {
        const after = await readAuthTokens(page);
        return after.accessToken;
      },
      { timeout: 30_000 }
    )
    .not.toBe(before.accessToken);

  const after = await readAuthTokens(page);
  if (!after.accessToken) return null;
  const scopeB = extractCompanyScopeFromToken(after.accessToken);
  if (!scopeB.activeCompanyId) return null;
  if (scopeB.activeCompanyId === scopeA.activeCompanyId) return null;

  return { a: scopeA.activeCompanyId, b: scopeB.activeCompanyId };
}

test('auth: SwitchCompanyScope default enabled uses Preferences (enabledCompanyIds omitted)', async ({ page }) => {
  test.setTimeout(120_000);

  const runtime = readRuntimeInfo();
  const baseURL = runtime.baseURL;
  const authPb = await getAuthPbModule();

  await loginAsE2EAdmin(page, baseURL);

  const { accessToken } = await readAuthState(page);
  expect(accessToken).not.toBe('');

  const pair = await discoverTwoCompanyIdsByUISwitch(page);
  test.skip(!pair, 'Need two distinct companies discoverable via UI switcher');

  const client0: any = makeUserClient(baseURL, accessToken, authPb.User);

  // Step 1: explicitly persist enabled=[a,b], active=a
  const r1: any = await (client0 as any).switchCompanyScope(
    create(authPb.UserSwitchCompanyScopeReqSchema, {
      activeCompanyId: pair!.a,
      enabledCompanyIds: toValue([pair!.a, pair!.b]),
    })
  );
  const tokenPair1 = fromValue(r1.result);
  expect(tokenPair1?.accessToken).toBeTruthy();

  const payload1 = decodeJwtPayload(String(tokenPair1.accessToken));
  expect(payload1).toBeTruthy();
  const scope1 = extractCompanyScopeFromToken(String(tokenPair1.accessToken));
  expect(scope1.activeCompanyId).toBe(pair!.a);
  expect(scope1.enabledCompanyIds).toEqual(expect.arrayContaining([pair!.a, pair!.b]));

  // Step 2: omit enabledCompanyIds; server must fall back to Preferences.enabledCompanyIds
  const client1: any = makeUserClient(baseURL, String(tokenPair1.accessToken), authPb.User);
  const r2: any = await (client1 as any).switchCompanyScope(
    create(authPb.UserSwitchCompanyScopeReqSchema, {
      activeCompanyId: pair!.b,
      // enabledCompanyIds intentionally omitted
    })
  );
  const tokenPair2 = fromValue(r2.result);
  expect(tokenPair2?.accessToken).toBeTruthy();

  const payload2 = decodeJwtPayload(String(tokenPair2.accessToken));
  expect(payload2).toBeTruthy();
  const scope2 = extractCompanyScopeFromToken(String(tokenPair2.accessToken));
  expect(scope2.activeCompanyId).toBe(pair!.b);
  expect(scope2.enabledCompanyIds).toEqual(expect.arrayContaining([pair!.a, pair!.b]));
});

test('auth: SwitchCompanyScope persists view; RefreshTokens reproduces the same active/enabled', async ({ page }) => {
  test.setTimeout(120_000);

  const runtime = readRuntimeInfo();
  const baseURL = runtime.baseURL;
  const authPb = await getAuthPbModule();

  await loginAsE2EAdmin(page, baseURL);

  const { accessToken } = await readAuthState(page);
  expect(accessToken).not.toBe('');

  const pair = await discoverTwoCompanyIdsByUISwitch(page);
  test.skip(!pair, 'Need two distinct companies discoverable via UI switcher');

  const client0: any = makeUserClient(baseURL, accessToken, authPb.User);

  // Persist a known scope: active=b, enabled=[a,b]
  const r1: any = await (client0 as any).switchCompanyScope(
    create(authPb.UserSwitchCompanyScopeReqSchema, {
      activeCompanyId: pair!.b,
      enabledCompanyIds: toValue([pair!.a, pair!.b]),
    })
  );
  const tokenPair1 = fromValue(r1.result);
  expect(tokenPair1?.refreshToken).toBeTruthy();

  // RefreshTokens must load metadata from DB and reproduce the same view.
  const client1: any = makeUserClient(baseURL, String(tokenPair1.accessToken), authPb.User);
  const r2: any = await (client1 as any).refreshTokens(
    create(authPb.UserRefreshTokensReqSchema, {
      refreshToken: String(tokenPair1.refreshToken),
    })
  );
  const tokenPair2 = fromValue(r2.result);
  expect(tokenPair2?.accessToken).toBeTruthy();

  const payload2 = decodeJwtPayload(String(tokenPair2.accessToken));
  expect(payload2).toBeTruthy();
  const scope2 = extractCompanyScopeFromToken(String(tokenPair2.accessToken));
  expect(scope2.activeCompanyId).toBe(pair!.b);
  expect(scope2.enabledCompanyIds).toEqual(expect.arrayContaining([pair!.a, pair!.b]));
});

test('auth: SwitchCompanyScope illegal enabledCompanyIds fails closed and emits audit log', async ({ page }) => {
  test.setTimeout(120_000);

  const runtime = readRuntimeInfo();
  const baseURL = runtime.baseURL;
  const authPb = await getAuthPbModule();

  await loginAsE2EAdmin(page, baseURL);

  const { accessToken, identity } = await readAuthState(page);
  expect(accessToken).not.toBe('');

  const userId = String(identity?.userId ?? '');
  const pair = await discoverTwoCompanyIdsByUISwitch(page);
  test.skip(!pair, 'Need two distinct companies discoverable via UI switcher');

  const client: any = makeUserClient(baseURL, accessToken, authPb.User);

  const illegalCompanyId = 'e2e-illegal-company-not-allowed';

  let err: any = null;
  try {
    await (client as any).switchCompanyScope(
      create(authPb.UserSwitchCompanyScopeReqSchema, {
        activeCompanyId: pair!.a,
        enabledCompanyIds: toValue([pair!.a, illegalCompanyId]),
      })
    );
  } catch (e) {
    err = e;
  }

  expect(err, 'SwitchCompanyScope should fail').toBeTruthy();
  const errCode = Number((err as any)?.code);
  const errMessage = String((err as any)?.rawMessage || (err as any)?.message || '');
  expect(err instanceof ConnectError || Number.isFinite(errCode)).toBeTruthy();
  expect(errCode).toBe(Code.InvalidArgument);
  expect(errMessage).toContain('enabledCompanyIds');

  // Verify audit log contains the expected event. (Server writes to runDir/server.log)
  await waitForServerLogContains('auth.user.switch_company_scope', 15_000);
  await waitForServerLogContains('"ok":false', 15_000);
  await waitForServerLogContains('enabledCompanyIds', 15_000);
  if (userId) {
    await waitForServerLogContains(userId, 15_000);
  }
});

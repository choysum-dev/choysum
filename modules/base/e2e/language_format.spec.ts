// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { test, expect, type Page } from '@playwright/test';
import fs from 'node:fs';
import path from 'node:path';
import { createClient, type Interceptor } from '@connectrpc/connect';
import { createGrpcWebTransport } from '@connectrpc/connect-web';
import { create } from '@bufbuild/protobuf';
import { ValueSchema, ListValueSchema, StructSchema, NullValue, type Value } from '@bufbuild/protobuf/wkt';

/**
 * T2.6: Changing Language thousand separator / Grouping updates business list number
 * display without editing FE source catalog constants.
 *
 * Mutation uses gRPC-Web UpdateById (same admin write path as the Language form Save)
 * to avoid OFormView Edit races where beginEdit() no-ops while original is still null.
 */

type RuntimeInfo = {
  baseURL: string;
  specsDir: string;
  module: string;
  scenario: string;
  fixtures: string[];
};

type BasePbModule = {
  Language: any;
  LanguageSearchReqSchema: any;
  LanguageUpdateByIdReqSchema: any;
};

let basePbModulePromise: Promise<BasePbModule> | null = null;

function readRuntimeInfo(): RuntimeInfo {
  const runtimePath = process.env.CHOYSUM_E2E_RUNTIME_JSON;
  if (!runtimePath) {
    throw new Error('CHOYSUM_E2E_RUNTIME_JSON env var not set');
  }
  const raw = fs.readFileSync(runtimePath, 'utf-8');
  return JSON.parse(raw) as RuntimeInfo;
}

async function loadBasePbModule(): Promise<BasePbModule> {
  const runtime = readRuntimeInfo();
  // E2E runner stages generated pb under specsDir/.generated so Playwright
  // transforms it under testDir (absolute file:// imports bypass that and break
  // @bufbuild/protobuf/codegenv2 under PW_DISABLE_TS_ESM=1).
  const staged = path.join(runtime.specsDir, '.generated', 'base_pb.ts');
  if (!fs.existsSync(staged)) {
    throw new Error(`Cannot find staged base_pb.ts at ${staged} (e2e runner should link it)`);
  }
  const mod = await import('./.generated/base_pb.ts');
  return mod as BasePbModule;
}

async function getBasePbModule(): Promise<BasePbModule> {
  if (!basePbModulePromise) {
    basePbModulePromise = loadBasePbModule();
  }
  return await basePbModulePromise;
}

async function readAccessToken(page: Page): Promise<string> {
  return page.evaluate(() => {
    const raw = localStorage.getItem('choysum.auth') || sessionStorage.getItem('choysum.auth');
    if (!raw) return '';
    try {
      const data = JSON.parse(raw);
      return String(data?.tokens?.accessToken || '');
    } catch {
      return '';
    }
  });
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
    return create(ValueSchema, {
      kind: {
        case: 'listValue',
        value: create(ListValueSchema, { values: val.map(item => toValue(item)) }),
      },
    });
  }
  if (typeof val === 'object') {
    const fields: Record<string, Value> = {};
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
      case 'listValue':
        return (x.kind.value?.values ?? []).map((it: any) => toJs(it));
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

async function makeLanguageClient(page: Page, baseURL: string) {
  const accessToken = await readAccessToken(page);
  expect(accessToken, 'access token after login').not.toBe('');
  const basePb = await getBasePbModule();
  const client = createClient(
    basePb.Language as any,
    createGrpcWebTransport({
      baseUrl: baseURL,
      interceptors: [makeAuthInterceptor(accessToken)],
    })
  ) as any;
  return { client, basePb };
}

async function resolveLanguageIdByCode(page: Page, baseURL: string, code: string): Promise<string> {
  const { client, basePb } = await makeLanguageClient(page, baseURL);
  const resp: any = await client.search(
    create(basePb.LanguageSearchReqSchema, {
      condition: toValue(['Code', '=', code]),
      options: toValue({ fields: ['Id', 'Code', 'Name'], limit: 1 }),
    })
  );
  const rows = fromValue(resp.result);
  expect(Array.isArray(rows) && rows.length > 0, `Language Search for Code=${code}`).toBe(true);
  const id = String(rows[0]?.Id || '').trim();
  expect(id, `Language.Id for Code=${code}`).not.toBe('');
  return id;
}

async function updateLanguageSeparators(page: Page, baseURL: string, languageId: string) {
  const { client, basePb } = await makeLanguageClient(page, baseURL);
  const resp: any = await client.updateById(
    create(basePb.LanguageUpdateByIdReqSchema, {
      id: languageId,
      values: toValue({
        DecimalSeparator: ',',
        ThousandSeparator: '.',
        Grouping: '[3,0]',
      }),
      returnFields: toValue(['Id', 'Code', 'DecimalSeparator', 'ThousandSeparator', 'Grouping']),
      options: toValue({}),
    })
  );
  const raw = fromValue(resp.result);
  const row = Array.isArray(raw) ? raw[0] : raw;
  expect(String(row?.ThousandSeparator ?? ''), 'ThousandSeparator after UpdateById').toBe('.');
  expect(String(row?.DecimalSeparator ?? ''), 'DecimalSeparator after UpdateById').toBe(',');
}

async function loginAsE2EAdmin(page: Page, baseURL: string) {
  await page.goto(`${baseURL}/web/auth/users`, { waitUntil: 'domcontentloaded' });
  await page.waitForSelector('input[placeholder*="username"]', { timeout: 15_000 });
  await page.getByPlaceholder(/username/i).fill('e2e-admin');
  await page.getByPlaceholder(/password/i).fill('e2e-admin');
  await page.locator('button[type="submit"]').click();
  await expect(page).toHaveURL(/\/web\/auth\/users/, { timeout: 20_000 });
}

test('base T2.6: Language thousand separator change updates exchange rate list display', async ({ page }) => {
  test.setTimeout(180_000);

  page.on('pageerror', err => {
    console.log(`[pageerror] ${err?.message || String(err)}`);
  });
  page.on('console', msg => {
    if (msg.type() === 'error') {
      console.log(`[console.error] ${msg.text()}`);
    }
  });

  const runtime = readRuntimeInfo();
  const baseURL = runtime.baseURL;

  await loginAsE2EAdmin(page, baseURL);

  // Baseline: list shows US-style grouping from seeded zh_CN (',' thousands, '.' decimal).
  await page.goto(`${baseURL}/web/base/exchange-rates`, { waitUntil: 'domcontentloaded' });
  await expect(page.locator('body')).toBeVisible();
  const rateCell = page.locator('.o-field-display-text', { hasText: /1[,.]234[,.]567/ }).first();
  await expect(rateCell).toBeVisible({ timeout: 30_000 });
  const beforeText = ((await rateCell.textContent()) || '').trim();
  expect(beforeText).toMatch(/1,234,567/);

  const languageId = await resolveLanguageIdByCode(page, baseURL, 'zh_CN');
  await updateLanguageSeparators(page, baseURL, languageId);

  // Full reload so i18n init re-fetches GetActiveLanguages overlays (authoritative for list formatting).
  const activeLangs = page.waitForResponse(
    r => r.url().includes('/base.Language/GetActiveLanguages') && r.request().method() === 'POST' && r.status() === 200,
    { timeout: 45_000 }
  );
  await page.goto(`${baseURL}/web/base/exchange-rates`, { waitUntil: 'domcontentloaded' });
  await page.reload({ waitUntil: 'domcontentloaded' });
  await activeLangs;
  await expect
    .poll(
      async () => {
        const texts = await page.locator('.o-field-display-text').allTextContents();
        return texts.map(t => String(t || '').trim()).find(t => /1[,.]234[,.]567/.test(t)) || '';
      },
      { timeout: 45_000 }
    )
    .toMatch(/1\.234\.567,/);
  expect(beforeText).toMatch(/1,234,567/);
});

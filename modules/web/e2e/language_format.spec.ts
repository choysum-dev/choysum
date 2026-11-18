// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { test, expect, page, runtime } from '@choysum/e2e';
import { createClient, type Interceptor } from '@connectrpc/connect';
import { createGrpcWebTransport } from '@connectrpc/connect-web';
import { create } from '@bufbuild/protobuf';
import { ValueSchema, ListValueSchema, StructSchema, NullValue, type Value } from '@bufbuild/protobuf/wkt';
import { loginAsE2EAdmin } from '../../auth/e2e/utils/login.ts';

/**
 * T2.6: Changing Language thousand separator / Grouping updates business list number
 * display without editing FE source catalog constants.
 *
 * Mutation uses gRPC-Web UpdateById (same admin write path as the Language form Save)
 * to avoid FormView Edit races where beginEdit() no-ops while original is still null.
 */

type BasePbModule = {
  Language: any;
  LanguageSearchReqSchema: any;
  LanguageUpdateByIdReqSchema: any;
};

let basePbModulePromise: Promise<BasePbModule> | null = null;

async function loadBasePbModule(): Promise<BasePbModule> {
  // Bundle plugin remaps *_pb.ts to <runDir>/.choysum/generated/...
  const mod = (await import(
    /* @vite-ignore */ './.generated/base_pb.ts' as string
  )) as BasePbModule;
  return mod;
}

async function getBasePbModule(): Promise<BasePbModule> {
  if (!basePbModulePromise) {
    basePbModulePromise = loadBasePbModule();
  }
  return await basePbModulePromise;
}

async function readAccessToken(): Promise<string> {
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

async function makeLanguageClient(baseURL: string) {
  const accessToken = await readAccessToken();
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

async function resolveLanguageIdByCode(baseURL: string, code: string): Promise<string> {
  const { client, basePb } = await makeLanguageClient(baseURL);
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

async function updateLanguageSeparators(
  baseURL: string,
  languageId: string,
  values: {
    DecimalSeparator: string;
    ThousandSeparator: string;
    Grouping: string;
  } = {
    DecimalSeparator: ',',
    ThousandSeparator: '.',
    Grouping: '[3,0]',
  }
) {
  const { client, basePb } = await makeLanguageClient(baseURL);
  const resp: any = await client.updateById(
    create(basePb.LanguageUpdateByIdReqSchema, {
      id: languageId,
      values: toValue(values),
      returnFields: toValue(['Id', 'Code', 'DecimalSeparator', 'ThousandSeparator', 'Grouping']),
      options: toValue({}),
    })
  );
  const raw = fromValue(resp.result);
  const row = Array.isArray(raw) ? raw[0] : raw;
  expect(String(row?.ThousandSeparator ?? ''), 'ThousandSeparator after UpdateById').toBe(
    values.ThousandSeparator
  );
  expect(String(row?.DecimalSeparator ?? ''), 'DecimalSeparator after UpdateById').toBe(
    values.DecimalSeparator
  );
}

/** Weekday header labels from the open date-picker calendar (first row). */
async function readDatePickerWeekdayHeaders(): Promise<string[]> {
  return page.evaluate(() => {
    const root = document.querySelector('.choy-date-picker__calendar');
    if (!root) return [] as string[];
    // DatePicker uses reka-ui CalendarHeadCell directly (no data-slot); weekday row is first.
    const row = root.querySelector('.flex');
    const cells = row ? Array.from(row.children) : [];
    const fromRow = cells.map(el => String(el.textContent || '').trim()).filter(Boolean);
    if (fromRow.length >= 7) return fromRow.slice(0, 7);
    return Array.from(root.querySelectorAll('.text-muted-foreground'))
      .map(el => String(el.textContent || '').trim())
      .filter(t => t.length > 0 && t.length <= 4)
      .slice(0, 7);
  });
}

async function updateLanguageDateAndWeekStart(
  baseURL: string,
  languageId: string,
  values: { DateFormat: string; FirstDayOfWeek: string }
) {
  const { client, basePb } = await makeLanguageClient(baseURL);
  const resp: any = await client.updateById(
    create(basePb.LanguageUpdateByIdReqSchema, {
      id: languageId,
      values: toValue(values),
      returnFields: toValue(['Id', 'Code', 'DateFormat', 'FirstDayOfWeek']),
      options: toValue({}),
    })
  );
  const raw = fromValue(resp.result);
  const row = Array.isArray(raw) ? raw[0] : raw;
  expect(String(row?.DateFormat ?? ''), 'DateFormat after UpdateById').toBe(values.DateFormat);
  expect(String(row?.FirstDayOfWeek ?? ''), 'FirstDayOfWeek after UpdateById').toBe(values.FirstDayOfWeek);
}

/**
 * Same overlay refresh Language form Save uses (`loadActiveLanguages`), without a full page reload.
 */
async function reloadActiveLanguageOverlaysLive(): Promise<boolean> {
  return page.evaluate(async () => {
    const root =
      document.querySelector('#app') ||
      document.querySelector('[data-v-app]') ||
      document.body;
    const app = (root as any)?.__vue_app__;
    const pinia = app?.config?.globalProperties?.$pinia;
    const store = pinia?._s?.get?.('choysum.web.i18n');
    if (!store || typeof store.loadActiveLanguages !== 'function') {
      return false;
    }
    return !!(await store.loadActiveLanguages());
  });
}
test('base T2.6: Language thousand separator change updates exchange rate list display', async () => {
  test.setTimeout(180_000);
  test.skip(runtime.scenario !== 'default', 'only runs under default scenario');

  const baseURL = runtime.baseURL;

  await loginAsE2EAdmin(page, baseURL);

  // Baseline: list shows US-style grouping from seeded zh_CN (',' thousands, '.' decimal).
  await page.goto(`${baseURL}/web/base/exchange-rates`, { waitUntil: 'domcontentloaded' });
  await expect(page.locator('body')).toBeVisible();
  const rateCell = page.locator('.choy-field-display-text', { hasText: /1[,.]234[,.]567/ }).first();
  await expect(rateCell).toBeVisible({ timeout: 30_000 });
  const beforeText = ((await rateCell.textContent()) || '').trim();
  expect(beforeText).toMatch(/1,234,567/);

  const languageId = await resolveLanguageIdByCode(baseURL, 'zh_CN');
  try {
    await updateLanguageSeparators(baseURL, languageId);

    // Full reload so i18n init re-fetches GetActiveLanguages overlays (authoritative for list formatting).
    const activeLangs = page.waitForResponse(
      { urlIncludes: '/base.Language/GetActiveLanguages', method: 'POST' },
      { timeout: 45_000 }
    );
    await page.goto(`${baseURL}/web/base/exchange-rates`, { waitUntil: 'domcontentloaded' });
    await page.reload({ waitUntil: 'domcontentloaded' });
    await activeLangs;
    await expect
      .poll(
        async () => {
          const texts = await page.locator('.choy-field-display-text').allTextContents();
          return texts.map(t => String(t || '').trim()).find(t => /1[,.]234[,.]567/.test(t)) || '';
        },
        { timeout: 45_000 }
      )
      .toMatch(/1\.234\.567,/);
    expect(beforeText).toMatch(/1,234,567/);
  } finally {
    // Restore seeded separators so later specs (and e2e retries) keep a known baseline.
    await updateLanguageSeparators(baseURL, languageId, {
      DecimalSeparator: '.',
      ThousandSeparator: ',',
      Grouping: '[3,0]',
    });
  }
});

test('base: Language DateFormat + FirstDayOfWeek update applies live to date field and calendar', async () => {
  test.setTimeout(180_000);
  test.skip(runtime.scenario !== 'default', 'only runs under default scenario');

  const baseURL = runtime.baseURL;
  await loginAsE2EAdmin(page, baseURL);

  // Seeded zh_CN uses YYYY-MM-DD + Monday week start; fixture date is 2026-07-22.
  await page.goto(`${baseURL}/web/base/exchange-rates`, { waitUntil: 'domcontentloaded' });
  await expect(page.locator('body')).toBeVisible();
  const dateCell = page.locator('.choy-field-display-text', { hasText: /2026/ }).first();
  await expect(dateCell).toBeVisible({ timeout: 30_000 });
  await expect
    .poll(async () => ((await dateCell.textContent()) || '').trim(), { timeout: 30_000 })
    .toMatch(/2026-07-22/);

  // Open create-form calendar before overlay update so week-start is observed on the same document.
  await page.goto(`${baseURL}/web/base/exchange-rates/new`, { waitUntil: 'domcontentloaded' });
  const dateTrigger = page.locator('[data-anchor="choy.internal.date-picker"]').first();
  await expect(dateTrigger).toBeVisible({ timeout: 30_000 });
  await dateTrigger.click();
  const beforeHeads = await readDatePickerWeekdayHeaders();
  expect(beforeHeads.length, `weekday headers before: ${JSON.stringify(beforeHeads)}`).toBe(7);
  // Monday-first for seeded zh_CN (一 / 周一 / Mon …) — not Sunday 日 alone.
  expect(beforeHeads[0], `Monday-first before overlay: ${JSON.stringify(beforeHeads)}`).toMatch(
    /^(一|周一|星期一|Mon|Mo|M|lun|Lu)$/i
  );
  // Dismiss calendar so the next open re-reads weekStartsOn from the store.
  await page.keyboard.press('Escape');

  const languageId = await resolveLanguageIdByCode(baseURL, 'zh_CN');
  try {
    await updateLanguageDateAndWeekStart(baseURL, languageId, {
      DateFormat: 'DD.MM.YYYY',
      FirstDayOfWeek: '0',
    });

    const activeLangs = page.waitForResponse(
      { urlIncludes: '/base.Language/GetActiveLanguages', method: 'POST' },
      { timeout: 45_000 }
    );
    const reloaded = await reloadActiveLanguageOverlaysLive();
    expect(reloaded, 'loadActiveLanguages via pinia (live, no full reload)').toBe(true);
    await activeLangs;

    // Same document: reopen calendar and assert Sunday-first without navigation.
    await dateTrigger.click();
    const afterHeads = await readDatePickerWeekdayHeaders();
    expect(afterHeads.length, `weekday headers after: ${JSON.stringify(afterHeads)}`).toBe(7);
    // zh-CN short Sunday is 日; reject bare "S" (also matches Sat) and Saturday forms.
    expect(afterHeads[0], `Sunday-first after overlay: ${JSON.stringify(afterHeads)}`).toMatch(
      /^(Su|Sun|周日|星期日|日|domingo|dom|Do)$/i
    );
    expect(afterHeads[0], `must not be Saturday: ${JSON.stringify(afterHeads)}`).not.toMatch(
      /^(Sat|Saturday|六|周六|星期六|sáb)/i
    );
    await page.keyboard.press('Escape');

    // DateFormat overlay on the list (live, after returning from create form).
    await page.goto(`${baseURL}/web/base/exchange-rates`, { waitUntil: 'domcontentloaded' });
    const liveDateCell = page.locator('.choy-field-display-text', { hasText: /2026/ }).first();
    await expect(liveDateCell).toBeVisible({ timeout: 30_000 });
    const listLangs = page.waitForResponse(
      { urlIncludes: '/base.Language/GetActiveLanguages', method: 'POST' },
      { timeout: 45_000 }
    );
    expect(await reloadActiveLanguageOverlaysLive()).toBe(true);
    await listLangs;
    await expect
      .poll(async () => ((await liveDateCell.textContent()) || '').trim(), { timeout: 30_000 })
      .toMatch(/22\.07\.2026/);
  } finally {
    // Restore zh_CN defaults even when an assertion fails mid-test.
    await updateLanguageDateAndWeekStart(baseURL, languageId, {
      DateFormat: 'YYYY-MM-DD',
      FirstDayOfWeek: '1',
    });
  }
});

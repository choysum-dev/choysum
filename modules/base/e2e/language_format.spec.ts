// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { test, expect, type Page } from '@playwright/test';
import fs from 'node:fs';

/**
 * T2.6: Admin changes Language thousand separator / Grouping → business list number display updates
 * without editing FE source catalog constants.
 */

type RuntimeInfo = {
  baseURL: string;
  specsDir: string;
  module: string;
  scenario: string;
  fixtures: string[];
};

function readRuntimeInfo(): RuntimeInfo {
  const runtimePath = process.env.CHOYSUM_E2E_RUNTIME_JSON;
  if (!runtimePath) {
    throw new Error('CHOYSUM_E2E_RUNTIME_JSON env var not set');
  }
  const raw = fs.readFileSync(runtimePath, 'utf-8');
  return JSON.parse(raw) as RuntimeInfo;
}

async function loginAsE2EAdmin(page: Page, baseURL: string) {
  await page.goto(`${baseURL}/web/auth/users`, { waitUntil: 'domcontentloaded' });
  await page.waitForSelector('input[placeholder*="username"]', { timeout: 15_000 });
  await page.getByPlaceholder(/username/i).fill('e2e-admin');
  await page.getByPlaceholder(/password/i).fill('e2e-admin');
  await page.locator('button[type="submit"]').click();
  await expect(page).toHaveURL(/\/web\/auth\/users/, { timeout: 20_000 });
}

async function fillFormFieldByLabel(page: Page, label: RegExp, value: string) {
  const item = page.locator('.el-form-item').filter({
    has: page.locator('.el-form-item__label', { hasText: label }),
  });
  await expect(item.first()).toBeVisible({ timeout: 15_000 });
  const input = item.first().locator('input.el-input__inner, input').first();
  await expect(input).toBeVisible({ timeout: 15_000 });
  await input.click();
  await input.fill('');
  await input.fill(value);
  // OVarCharField uses buffered commit; blur so the form model picks up the value before Save.
  await input.blur();
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

  // Admin UI: open Chinese (Simplified) language and change separators + Grouping.
  await page.goto(`${baseURL}/web/base/languages`, { waitUntil: 'domcontentloaded' });
  await expect(page.getByText(/Chinese \(Simplified\)|zh_CN|简体/i).first()).toBeVisible({ timeout: 30_000 });
  await page.getByText(/Chinese \(Simplified\)|简体/i).first().click();
  await expect(page).toHaveURL(/\/web\/base\/languages\/[^/]+/, { timeout: 20_000 });

  // Wait until form finished loading (avoids initializeForm resetting edit mode after a premature Edit click).
  await expect(page.locator('.el-form-item__label', { hasText: /千位分隔符|Thousands Separator/ })).toBeVisible({
    timeout: 30_000,
  });
  const actionBar = page.locator('.form-view__system-actions');
  await expect(actionBar.getByText(/编辑|Edit/)).toBeVisible({ timeout: 15_000 });
  // Small settle so the display-mode initializeForm promise has finished.
  await page.waitForTimeout(500);
  await actionBar.getByText(/编辑|Edit/).click();
  await expect(actionBar.getByText(/保存|Save/)).toBeVisible({ timeout: 15_000 });

  await fillFormFieldByLabel(page, /^(Decimal Separator|小数分隔符)$/, ',');
  await fillFormFieldByLabel(page, /^(Thousands Separator|Thousand Separator|千位分隔符)$/, '.');
  await fillFormFieldByLabel(page, /^(Grouping|分组)$/, '[3,0]');

  const saveResp = page.waitForResponse(
    r => r.url().includes('/base.Language/') && r.request().method() === 'POST' && r.status() === 200,
    { timeout: 30_000 }
  );
  await actionBar.getByText(/保存|Save/).click();
  await saveResp;
  await expect(page.getByText(/Saved successfully|保存成功/i)).toBeVisible({ timeout: 15_000 });

  // Confirm Language form itself reflects the new thousand separator after save (display mode).
  await expect(
    page.locator('.el-form-item').filter({
      has: page.locator('.el-form-item__label', { hasText: /千位分隔符|Thousands Separator/ }),
    }).locator('.o-field-display-text')
  ).toHaveText('.', { timeout: 15_000 });

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

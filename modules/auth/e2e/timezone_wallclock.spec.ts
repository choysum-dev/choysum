// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { test, expect, type Page } from '@playwright/test';
import fs from 'node:fs';

/**
 * Scenario #12 / §11.4 S2: changing User.Timezone updates list datetime wall-clock
 * (ODatetimeField via formatDateTime), without rewriting stored UTC.
 */

type RuntimeInfo = {
  baseURL: string;
  specsDir: string;
  module: string;
  scenario: string;
  fixtures: string[];
};

const DATETIME_CELL = /^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}/;

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

async function firstDatetimeDisplayText(page: Page): Promise<string> {
  const texts = await page.locator('.o-field-display-text').allTextContents();
  return texts.map(t => String(t || '').trim()).find(t => DATETIME_CELL.test(t)) || '';
}

async function waitForDatetimeCell(page: Page): Promise<string> {
  await expect
    .poll(async () => firstDatetimeDisplayText(page), { timeout: 45_000 })
    .toMatch(DATETIME_CELL);
  return firstDatetimeDisplayText(page);
}

async function setUserTimezoneViaPreferences(page: Page, iana: string) {
  // Labels are translated (fixture Language=zh_CN → 用户菜单 / 设置 / …).
  const userMenu = page.getByRole('button', { name: /User menu|用户菜单/i });
  await expect(userMenu).toBeVisible({ timeout: 20_000 });
  await userMenu.click();
  await page.getByRole('menuitem', { name: /Settings|Profile|设置|个人资料/i }).first().click();

  const dialog = page.locator('.o-preferences-dialog');
  await expect(dialog).toBeVisible({ timeout: 15_000 });

  // Language is the first select; Timezone is the second (labels may be translated).
  const tzSelect = dialog.locator('.el-form-item').nth(1).locator('.el-select');
  await tzSelect.click();
  const filterInput = tzSelect.locator('input');
  await filterInput.fill(iana);
  await page.locator('.el-select-dropdown:visible').getByRole('option', { name: iana, exact: true }).click();

  const save = dialog.getByRole('button', { name: /Update preferences|更新偏好设置/i });
  await Promise.all([page.waitForEvent('load', { timeout: 45_000 }), save.click()]);
  await expect(page).toHaveURL(/\/web\/auth\/users/, { timeout: 30_000 });
  await expect(page.locator('.o-preferences-dialog')).toHaveCount(0, { timeout: 15_000 });
}

test('auth e2e: User.Timezone change updates users list datetime wall-clock', async ({ page }) => {
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
  await page.goto(`${baseURL}/web/auth/users`, { waitUntil: 'domcontentloaded' });
  await waitForDatetimeCell(page);

  await setUserTimezoneViaPreferences(page, 'UTC');
  await page.goto(`${baseURL}/web/auth/users`, { waitUntil: 'domcontentloaded' });
  const utcText = await waitForDatetimeCell(page);

  await setUserTimezoneViaPreferences(page, 'America/New_York');
  await page.goto(`${baseURL}/web/auth/users`, { waitUntil: 'domcontentloaded' });
  const nyText = await waitForDatetimeCell(page);

  expect(utcText).toMatch(DATETIME_CELL);
  expect(nyText).toMatch(DATETIME_CELL);
  expect(utcText).not.toBe(nyText);
});

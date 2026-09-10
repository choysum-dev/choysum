// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { test, expect, page, runtime, type Page } from '@choysum/e2e';
import { loginAsE2EAdmin } from './utils/login.ts';

/**
 * Scenario #12 / §11.4 S2: changing User.Timezone updates list datetime wall-clock
 * (ODatetimeField via formatDateTime), without rewriting stored UTC.
 */

const DATETIME_CELL = /^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}/;

async function firstDatetimeDisplayText(p: Page): Promise<string> {
  const texts = await p.locator('.o-field-display-text').allTextContents();
  return texts.map(t => String(t || '').trim()).find(t => DATETIME_CELL.test(t)) || '';
}

async function waitForDatetimeCell(p: Page): Promise<string> {
  await expect.poll(async () => firstDatetimeDisplayText(p), { timeout: 45_000 }).toMatch(DATETIME_CELL);
  return firstDatetimeDisplayText(p);
}

async function setUserTimezoneViaPreferences(p: Page, iana: string) {
  const userMenu = p.getByRole('button', { name: /User menu|用户菜单/i });
  await expect(userMenu).toBeVisible({ timeout: 20_000 });
  await userMenu.click();
  await p.getByRole('menuitem', { name: /Settings|Profile|设置|个人资料/i }).first().click();

  const dialog = p.locator('.o-preferences-dialog');
  await expect(dialog).toBeVisible({ timeout: 15_000 });

  const tzSelect = dialog.locator('.el-form-item').nth(1).locator('.el-select');
  await tzSelect.click();
  const filterInput = tzSelect.locator('input');
  await filterInput.fill(iana);
  // Click the matching option in any open Element Plus dropdown (avoid :visible).
  await expect
    .poll(
      async () =>
        p.evaluate((want: string) => {
          const opts = Array.from(document.querySelectorAll('.el-select-dropdown li, [role="option"]'));
          const el = opts.find(o => String(o.textContent || '').trim() === want);
          if (!el) return false;
          el.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true, view: window }));
          return true;
        }, iana),
      { timeout: 15_000 }
    )
    .toBe(true);

  const save = dialog.getByRole('button', { name: /Update preferences|更新偏好设置/i });
  // Prefer reload after save rather than waitForEvent('load').
  await save.click();
  await p.waitForFunction(() => !document.querySelector('.o-preferences-dialog'), undefined, { timeout: 45_000 }).catch(() => undefined);
  await expect(p).toHaveURL(/\/web\/auth\/users/, { timeout: 30_000 });
  await expect(p.locator('.o-preferences-dialog')).toHaveCount(0, { timeout: 15_000 });
}

test('auth e2e: User.Timezone change updates users list datetime wall-clock', async () => {
  test.setTimeout(180_000);

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

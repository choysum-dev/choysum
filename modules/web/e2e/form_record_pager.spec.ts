// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { test, expect, page, runtime } from '@choysum/e2e';
import { loginAsE2EAdmin } from '../../auth/e2e/utils/login.ts';

/**
 * Form record pager (browser):
 * - open from list page 2 → global n/N
 * - in-window next changes id
 * - deep link (after list clear) hides pager
 *
 * Cross-page Search neighbor resolution: recordSetStore unit tests.
 * Dirty discard before next: useFormRecordPager + confirmDiscardIfDirty (Form lifecycle).
 */
test('form record pager: page-2 open, next, deep link hides pager', async () => {
  test.setTimeout(120_000);
  const baseURL = runtime.baseURL;

  await loginAsE2EAdmin(page, baseURL);

  await page.goto(`${baseURL}/web/base/countries`, { waitUntil: 'domcontentloaded' });
  const pagination = page.getByTestId('choy-pagination');
  await expect(pagination).toBeVisible({ timeout: 60_000 });

  const rangeValue = pagination.locator('.choy-pagination__range-value');
  const totalLabel = pagination.locator('.choy-pagination__total');
  await expect
    .poll(async () => Number((await totalLabel.textContent())?.trim() || '0'), { timeout: 60_000 })
    .toBeGreaterThan(1);
  const listTotal = Number((await totalLabel.textContent())?.trim() || '0');
  const rangeText = String((await rangeValue.textContent()) || '').trim();
  const rangeMatch = /^(\d+)\s*-\s*(\d+)$/.exec(rangeText);
  expect(!!rangeMatch).toBe(true);
  const pageStart = Number(rangeMatch![1]);
  const pageEnd = Number(rangeMatch![2]);
  const pageSize = pageEnd - pageStart + 1;
  expect(listTotal > pageSize).toBe(true);

  await page.getByTestId('choy-pagination-next').click();
  const page2Start = pageStart + pageSize;
  await expect(rangeValue).toHaveText(new RegExp(`^${page2Start}-`), { timeout: 30_000 });

  const firstName = page.locator('[role="row"][aria-rowindex="2"] .choy-field-display-text').first();
  await expect(firstName).toBeVisible({ timeout: 20_000 });
  await firstName.click();
  await expect(page).toHaveURL(/\/web\/base\/countries\/[^/]+$/, { timeout: 20_000 });

  const pager = page.getByTestId('choy-form-pager');
  await expect(pager).toBeVisible({ timeout: 20_000 });
  await expect(pager).toHaveText(new RegExp(`${page2Start}\\s*/\\s*${listTotal}`));

  const urlOnPage2First = String(await page.url());
  const idOnPage2First = urlOnPage2First.replace(/.*\/countries\//, '').replace(/\/$/, '');

  const crumbBeforeNext = page.getByTestId('choy-shell-breadcrumb');
  await expect(crumbBeforeNext).toBeVisible({ timeout: 15_000 });
  const crumbCountBefore = await crumbBeforeNext.locator('[data-slot=breadcrumb-item]').count();

  await page.getByTestId('choy-form-pager-next').click();
  await expect.poll(async () => String(await page.url()), { timeout: 20_000 }).not.toBe(urlOnPage2First);
  await expect(pager).toHaveText(new RegExp(`${page2Start + 1}\\s*/\\s*${listTotal}`));
  // Sibling pager must replace the detail crumb, not deepen the trail.
  await expect(crumbBeforeNext.locator('[data-slot=breadcrumb-item]')).toHaveCount(crumbCountBefore);

  // Deep link after list clear: no pager.
  await page.goto(`${baseURL}/web/base/countries`, { waitUntil: 'domcontentloaded' });
  await expect(pagination).toBeVisible({ timeout: 60_000 });
  await page.goto(`${baseURL}/web/base/countries/${idOnPage2First}`, {
    waitUntil: 'domcontentloaded',
  });
  await expect(page).toHaveURL(new RegExp(`/web/base/countries/${idOnPage2First}`), {
    timeout: 20_000,
  });
  await expect(page.getByTestId('choy-form-pager')).toHaveCount(0, { timeout: 15_000 });
});

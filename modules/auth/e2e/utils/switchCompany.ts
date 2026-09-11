// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { expect, page } from '@choysum/e2e';
import { waitForGrpcWebUnaryOk } from './grpcweb.ts';

/**
 * Pick a non-selected company option in the open active-company el-select.
 * Uses HTMLElement.click() so Element Plus Vue handlers run (MouseEvent dispatch is flaky).
 */
async function pickOtherActiveCompanyOption(): Promise<void> {
  // Do not toggle-close an already-open dropdown (retry paths can leave it expanded).
  const dropdownOpen = await page.evaluate(() => {
    const nodes = Array.from(
      document.querySelectorAll('.el-select-dropdown')
    ) as HTMLElement[];
    return nodes.some(el => {
      const style = window.getComputedStyle(el);
      if (!style || style.visibility === 'hidden' || style.display === 'none' || style.opacity === '0') {
        return false;
      }
      const r = el.getBoundingClientRect();
      return r.width > 0 && r.height > 0;
    });
  });
  if (!dropdownOpen) {
    await page.getByTestId('company-active-select').click();
  }

  await expect
    .poll(
      async () =>
        page.evaluate(() => {
          const opts = Array.from(
            document.querySelectorAll(
              '.el-select-dropdown__item, .el-select-dropdown li, [role="option"]'
            )
          ) as HTMLElement[];
          return opts.length;
        }),
      { timeout: 10_000 }
    )
    .toBeGreaterThanOrEqual(2);

  const clicked = await page.evaluate(() => {
    const opts = Array.from(
      document.querySelectorAll(
        '.el-select-dropdown__item, .el-select-dropdown li, [role="option"]'
      )
    ) as HTMLElement[];
    const target =
      opts.find(
        o => !o.classList.contains('is-selected') && o.getAttribute('aria-selected') !== 'true'
      ) || opts[opts.length - 1];
    if (!target) return false;
    target.click();
    return true;
  });
  expect(clicked).toBe(true);
}

async function clickApplyButton(): Promise<void> {
  const clicked = await page.evaluate(() => {
    const btn = document.querySelector(
      '[data-testid="company-switch-apply"]'
    ) as HTMLButtonElement | null;
    if (!btn) return false;
    if (btn.disabled || btn.getAttribute('aria-disabled') === 'true') return false;
    btn.click();
    return true;
  });
  expect(clicked).toBe(true);
}

/**
 * Open the company switcher, select another company, and wait for SwitchCompanyScope.
 *
 * Retries the full open→select→apply path: Element Plus often swallows the first
 * apply click under CDP, and panel-open refreshToken can race with a thin click path.
 */
export async function switchCompanyViaUI(): Promise<void> {
  let lastErr: unknown;
  for (let attempt = 1; attempt <= 3; attempt++) {
    try {
      const trigger = page.getByTestId('company-switch-trigger');
      await expect(trigger).toBeVisible();

      // Ensure panel is open (re-open after a previous apply that closed it).
      // Presence alone is not enough: Element Plus may keep the panel mounted while hidden.
      const panel = page.getByTestId('company-switch-panel');
      const panelVisible = await panel.isVisible().catch(() => false);
      if (!panelVisible) {
        await trigger.click();
      }
      await expect(panel).toBeVisible({ timeout: 10_000 });

      await pickOtherActiveCompanyOption();

      const applyButton = page.getByTestId('company-switch-apply');
      await expect.poll(async () => await applyButton.isEnabled(), { timeout: 15_000 }).toBe(true);
      await expect(page.getByTestId('company-switch-hint')).toHaveCount(0);

      const switchOk = waitForGrpcWebUnaryOk(page, '/auth.User/SwitchCompanyScope', {
        timeoutMs: 20_000,
      });
      // Consume rejection if clickApplyButton throws and this wait is abandoned on retry.
      void switchOk.catch(() => undefined);
      await clickApplyButton();
      await switchOk;
      return;
    } catch (err) {
      lastErr = err;
      // Close only when the panel is actually visible; presence alone would re-open it.
      await page
        .evaluate(() => {
          const trigger = document.querySelector(
            '[data-testid="company-switch-trigger"]'
          ) as HTMLElement | null;
          const panel = document.querySelector(
            '[data-testid="company-switch-panel"]'
          ) as HTMLElement | null;
          if (!trigger || !panel) return;
          const style = window.getComputedStyle(panel);
          if (!style || style.visibility === 'hidden' || style.display === 'none' || style.opacity === '0') {
            return;
          }
          const r = panel.getBoundingClientRect();
          if (r.width <= 0 || r.height <= 0) return;
          trigger.click();
        })
        .catch(() => undefined);
      await page.waitForTimeout(250 * attempt);
    }
  }
  throw lastErr;
}

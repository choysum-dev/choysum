// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { expect, page } from '@choysum/e2e';
import { waitForGrpcWebUnaryOk } from './grpcweb.ts';

async function readActiveCompanyIdFromAuth(): Promise<string> {
  return page.evaluate(() => {
    const raw = localStorage.getItem('choysum.auth') || sessionStorage.getItem('choysum.auth');
    if (!raw) return '';
    try {
      const data = JSON.parse(raw);
      return String(data?.identity?.metadata?.activeCompanyId || '').trim();
    } catch {
      return '';
    }
  });
}

/**
 * Pick a non-selected company option in the open active-company el-select.
 * Uses HTMLElement.click() so Element Plus Vue handlers run (MouseEvent dispatch is flaky).
 */
async function pickOtherActiveCompanyOption(): Promise<void> {
  // Scope to the company panel: other page dropdowns must not skip opening this select.
  const dropdownOpen = await page.evaluate(() => {
    const panel = document.querySelector('[data-testid="company-switch-panel"]');
    if (!panel) return false;
    const select = panel.querySelector('[data-testid="company-active-select"]');
    if (!select) return false;
    if (select.getAttribute('aria-expanded') === 'true') return true;
    if (select.querySelector('[aria-expanded="true"]')) return true;
    const nodes = Array.from(panel.querySelectorAll('.el-select-dropdown')) as HTMLElement[];
    return nodes.some(el => {
      const style = window.getComputedStyle(el);
      if (!style || style.visibility === 'hidden' || style.display === 'none' || style.opacity === '0') {
        return false;
      }
      const r = el.getBoundingClientRect();
      if (r.width <= 0 || r.height <= 0) return false;
      return !!el.querySelector('.el-select-dropdown__item, [role="option"]');
    });
  });
  if (!dropdownOpen) {
    await page.getByTestId('company-active-select').click();
  }

  await expect
    .poll(
      async () =>
        page.evaluate(() => {
          const panel = document.querySelector('[data-testid="company-switch-panel"]');
          if (!panel) return 0;
          const opts = Array.from(
            panel.querySelectorAll(
              '.el-select-dropdown__item, .el-select-dropdown li, [role="option"]'
            )
          ) as HTMLElement[];
          return opts.length;
        }),
      { timeout: 10_000 }
    )
    .toBeGreaterThanOrEqual(2);

  // Never fall back to the last option: it may be the already-selected company and
  // then Apply is a no-op while waitForGrpcWebUnaryOk burns a full timeout.
  const clicked = await page.evaluate(() => {
    const panel = document.querySelector('[data-testid="company-switch-panel"]');
    if (!panel) return false;
    const opts = Array.from(
      panel.querySelectorAll(
        '.el-select-dropdown__item, .el-select-dropdown li, [role="option"]'
      )
    ) as HTMLElement[];
    const target = opts.find(
      o => !o.classList.contains('is-selected') && o.getAttribute('aria-selected') !== 'true'
    );
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
    const beforeCompanyId = await readActiveCompanyIdFromAuth();
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
      try {
        await switchOk;
      } catch (waitErr) {
        // Apply is non-idempotent: if CDP missed the response but scope already changed,
        // treat as success so a retry cannot switch back to the original company.
        const afterCompanyId = await readActiveCompanyIdFromAuth();
        // beforeCompanyId may be empty if auth storage was briefly unreadable.
        if (afterCompanyId && afterCompanyId !== beforeCompanyId) {
          return;
        }
        throw waitErr;
      }
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

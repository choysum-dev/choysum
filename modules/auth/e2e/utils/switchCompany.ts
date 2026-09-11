// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { expect, page } from '@choysum/e2e';
import { waitForGrpcWebUnaryOk } from './grpcweb.ts';

/**
 * Decode activeCompanyId from the persisted access token JWT.
 * Auth persist paths omit identity, so localStorage.identity is unreliable.
 * Runs in the browser page (atob is available there).
 */
async function readActiveCompanyIdFromAuth(): Promise<string> {
  return page.evaluate(() => {
    const raw = localStorage.getItem('choysum.auth') || sessionStorage.getItem('choysum.auth');
    if (!raw) return '';
    try {
      const data = JSON.parse(raw);
      const token = String(data?.tokens?.accessToken || '').trim();
      if (!token) return '';
      const parts = token.split('.');
      if (parts.length < 2) return '';
      const b64 = parts[1].replace(/-/g, '+').replace(/_/g, '/');
      const pad = b64.length % 4 === 0 ? '' : '='.repeat(4 - (b64.length % 4));
      const json = decodeURIComponent(
        atob(b64 + pad)
          .split('')
          .map(c => '%' + ('00' + c.charCodeAt(0).toString(16)).slice(-2))
          .join('')
      );
      const payload = JSON.parse(json);
      const meta = payload?.meta && typeof payload.meta === 'object' ? payload.meta : {};
      return String(meta.activeCompanyId || '').trim();
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
  // Scope to the company panel: select uses teleported=false, so options live under the panel.
  // Other page dropdowns must not skip opening this select.
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
  // then Apply is a no-op while waits burn a full timeout.
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
 * Open the company switcher, select another company, and wait until activeCompanyId changes.
 *
 * Retries the full open→select→apply path: Element Plus often swallows the first
 * apply click under CDP, and panel-open refreshToken can race with a thin click path.
 *
 * Success is JWT activeCompanyId change (tokens are persisted; identity is not).
 */
export async function switchCompanyViaUI(): Promise<void> {
  let lastErr: unknown;
  for (let attempt = 1; attempt <= 3; attempt++) {
    const beforeCompanyId = await readActiveCompanyIdFromAuth();
    try {
      const trigger = page.getByTestId('company-switch-trigger');
      await expect(trigger).toBeVisible();

      // Presence alone is not enough: Element Plus may keep the panel mounted while hidden.
      const panel = page.getByTestId('company-switch-panel');
      const panelVisible = await panel.isVisible().catch(() => false);
      // If already open, an in-flight panel-open RefreshTokens may still complete later and
      // overwrite SwitchCompanyScope. Close + reopen so we can wait on a fresh refresh.
      if (panelVisible) {
        await trigger.click();
        await expect
          .poll(async () => !(await panel.isVisible().catch(() => false)), { timeout: 10_000 })
          .toBe(true);
      }

      // Arm before open: panel watcher always force-refreshes tokens.
      const refreshWait = waitForGrpcWebUnaryOk(page, '/auth.User/RefreshTokens', {
        timeoutMs: 20_000,
      }).catch(() => undefined);
      await trigger.click();
      await expect(panel).toBeVisible({ timeout: 10_000 });
      // Settle refresh before select/apply so an in-flight RefreshTokens cannot
      // overwrite a later SwitchCompanyScope TokenPair.
      await refreshWait;

      await pickOtherActiveCompanyOption();

      const applyButton = page.getByTestId('company-switch-apply');
      await expect.poll(async () => await applyButton.isEnabled(), { timeout: 15_000 }).toBe(true);
      await expect(page.getByTestId('company-switch-hint')).toHaveCount(0);

      await clickApplyButton();

      await expect
        .poll(
          async () => {
            const after = await readActiveCompanyIdFromAuth();
            return Boolean(after && after !== beforeCompanyId);
          },
          { timeout: 20_000 }
        )
        .toBe(true);
      return;
    } catch (err) {
      lastErr = err;
      // If scope already changed (e.g. apply raced on a prior attempt), stop.
      const afterCompanyId = await readActiveCompanyIdFromAuth();
      if (afterCompanyId && afterCompanyId !== beforeCompanyId) {
        return;
      }
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

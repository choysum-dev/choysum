// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { expect, page } from '@choysum/e2e';
import { waitForGrpcWebUnaryOk } from './grpcweb.ts';

/**
 * Decode activeCompanyId from the persisted access token JWT.
 * Auth persist paths omit identity, so localStorage.identity is unreliable.
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
      const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/';
      const clean = (b64 + pad).replace(/[^A-Za-z0-9+/=]/g, '');
      const bytes: number[] = [];
      for (let i = 0; i < clean.length; i += 4) {
        const a = chars.indexOf(clean[i]);
        const b = chars.indexOf(clean[i + 1]);
        const c = chars.indexOf(clean[i + 2]);
        const d = chars.indexOf(clean[i + 3]);
        bytes.push((a << 2) | (b >> 4));
        if (clean[i + 2] !== '=' && c >= 0) bytes.push(((b & 15) << 4) | (c >> 2));
        if (clean[i + 3] !== '=' && d >= 0) bytes.push(((c & 3) << 6) | d);
      }
      const u8 = new Uint8Array(bytes);
      let json = '';
      if (typeof TextDecoder !== 'undefined') {
        json = new TextDecoder('utf-8').decode(u8);
      } else {
        for (let i = 0; i < u8.length; i++) json += String.fromCharCode(u8[i]);
      }
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
 * Success is JWT activeCompanyId change (tokens are persisted). CDP observation of
 * SwitchCompanyScope is best-effort only — identity is not in persist paths.
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
      // Arm before open: panel watcher always force-refreshes tokens.
      let refreshWait: Promise<unknown> = Promise.resolve();
      if (!panelVisible) {
        refreshWait = waitForGrpcWebUnaryOk(page, '/auth.User/RefreshTokens', {
          timeoutMs: 20_000,
        }).catch(() => undefined);
        await trigger.click();
      }
      await expect(panel).toBeVisible({ timeout: 10_000 });
      // Settle refresh before select/apply so an in-flight RefreshTokens cannot
      // overwrite a later SwitchCompanyScope TokenPair.
      await refreshWait;

      await pickOtherActiveCompanyOption();

      const applyButton = page.getByTestId('company-switch-apply');
      await expect.poll(async () => await applyButton.isEnabled(), { timeout: 15_000 }).toBe(true);
      await expect(page.getByTestId('company-switch-hint')).toHaveCount(0);

      // Best-effort network observe; do not require it (CDP can miss under load).
      const switchOk = waitForGrpcWebUnaryOk(page, '/auth.User/SwitchCompanyScope', {
        timeoutMs: 20_000,
      });
      void switchOk.catch(() => undefined);
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
      // If scope already changed (e.g. CDP/apply race on a prior attempt), stop.
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

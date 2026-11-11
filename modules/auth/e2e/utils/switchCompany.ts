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
 * Click another company's name in the open switcher list (immediate loginto).
 */
async function clickOtherCompanyLoginto(): Promise<void> {
  await expect(page.getByTestId('company-switch-panel')).toBeVisible({ timeout: 10_000 });

  let activeCompanyId = '';
  let lastReadErr: unknown;
  const deadline = Date.now() + 5_000;
  while (!activeCompanyId && Date.now() < deadline) {
    try {
      const first = await readActiveCompanyIdFromAuth();
      if (first) {
        await page.waitForTimeout(100);
        const second = await readActiveCompanyIdFromAuth();
        if (second && second === first) {
          activeCompanyId = first;
          break;
        }
      }
    } catch (err) {
      lastReadErr = err;
    }
    await page.waitForTimeout(100);
  }
  if (!activeCompanyId) {
    throw new Error(
      `company switch: active company id unavailable; refusing to pick a row blindly${
        lastReadErr ? ` (last token read failed: ${String(lastReadErr)})` : ''
      }`,
    );
  }

  // Wait for a non-active loginto control (name click stays enabled while hierarchy loads).
  await expect
    .poll(
      async () =>
        page.evaluate((active: string) => {
          const items = Array.from(
            document.querySelectorAll('[data-testid="company-switch-item"]'),
          ) as HTMLElement[];
          for (const el of items) {
            const id = String(el.getAttribute('data-company-id') || '').trim();
            if (!id || id === active) continue;
            const btn = el.querySelector(
              '[data-testid="company-switch-loginto"]',
            ) as HTMLButtonElement | null;
            if (btn && !btn.disabled) return true;
          }
          return false;
        }, activeCompanyId),
      { timeout: 15_000 },
    )
    .toBe(true);

  // Retry the click: allowlist refresh can re-render between the readiness poll and evaluate.
  let otherCompanyId = '';
  await expect
    .poll(
      async () => {
        otherCompanyId = await page.evaluate((active: string) => {
          const items = Array.from(
            document.querySelectorAll('[data-testid="company-switch-item"]'),
          ) as HTMLElement[];
          for (const el of items) {
            const id = String(el.getAttribute('data-company-id') || '').trim();
            if (!id || id === active) continue;
            const btn = el.querySelector(
              '[data-testid="company-switch-loginto"]',
            ) as HTMLButtonElement | null;
            if (!btn || btn.disabled) continue;
            btn.click();
            return id;
          }
          return '';
        }, activeCompanyId);
        return otherCompanyId;
      },
      { timeout: 10_000 },
    )
    .not.toBe('');
}

/**
 * Open the company switcher, enter another company, and wait until activeCompanyId changes.
 *
 * Retries the full open→loginto path when the panel or company list is still settling.
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

      // Presence alone is not enough: the panel may stay mounted while hidden.
      const panel = page.getByTestId('company-switch-panel');
      const panelVisible = await panel.isVisible().catch(() => false);
      if (panelVisible) {
        await trigger.click();
        await expect
          .poll(async () => !(await panel.isVisible().catch(() => false)), { timeout: 10_000 })
          .toBe(true);
      }

      // Panel open Browses the user allowlist (no RefreshTokens); loginto calls SwitchCompanyScope.
      const switchWait = waitForGrpcWebUnaryOk(page, '/auth.User/SwitchCompanyScope', {
        timeoutMs: 20_000,
      }).catch(() => undefined);
      await trigger.click();
      await expect(panel).toBeVisible({ timeout: 10_000 });

      await clickOtherCompanyLoginto();
      await switchWait;

      await expect
        .poll(
          async () => {
            const after = await readActiveCompanyIdFromAuth();
            return Boolean(after && after !== beforeCompanyId);
          },
          { timeout: 20_000 },
        )
        .toBe(true);
      return;
    } catch (err) {
      lastErr = err;
      // If scope already changed (e.g. loginto raced on a prior attempt), stop.
      const afterCompanyId = await readActiveCompanyIdFromAuth();
      if (afterCompanyId && afterCompanyId !== beforeCompanyId) {
        return;
      }
      // Close only when the panel is actually visible; presence alone would re-open it.
      await page
        .evaluate(() => {
          const trigger = document.querySelector(
            '[data-testid="company-switch-trigger"]',
          ) as HTMLElement | null;
          const panel = document.querySelector(
            '[data-testid="company-switch-panel"]',
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

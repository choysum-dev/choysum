// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { expect, page } from '@choysum/e2e';
import { pickAlternativeCompanyOptionValue } from '../../web/components/layout/switch_company_option_pick.ts';
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
 * Pick a non-selected company option in the open active-company native select.
 */
async function pickOtherActiveCompanyOption(): Promise<void> {
  const panelSelect = '[data-testid="company-switch-panel"] [data-testid="company-active-select"]';
  await expect(page.getByTestId('company-active-select')).toBeVisible({ timeout: 10_000 });

  // Companies load async after the panel opens; wait until a non-current option exists.
  await expect
    .poll(
      async () =>
        page.evaluate((sel: string) => {
          const select = document.querySelector(sel) as HTMLSelectElement | null;
          if (!select) return 0;
          return Array.from(select.options)
            .map(opt => String(opt.value || '').trim())
            .filter(Boolean).length;
        }, panelSelect),
      { timeout: 10_000 }
    )
    .toBeGreaterThanOrEqual(2);

  const activeCompanyId = await readActiveCompanyIdFromAuth();
  if (!activeCompanyId) {
    throw new Error('company switch: active company id unavailable; refusing to pick an option blindly');
  }
  const selectState = await page.evaluate((sel: string) => {
    const select = document.querySelector(sel) as HTMLSelectElement | null;
    if (!select) return { current: '', values: [] as string[] };
    return {
      current: String(select.value || '').trim(),
      values: Array.from(select.options)
        .map(opt => String(opt.value || '').trim())
        .filter(Boolean),
    };
  }, panelSelect);
  const otherValue = pickAlternativeCompanyOptionValue(
    selectState.values,
    selectState.current,
    activeCompanyId
  );
  expect(otherValue, 'company switch: no selectable alternative company option').not.toBe('');

  const applied = await page.evaluate(
    ({ sel, other }: { sel: string; other: string }) => {
      const select = document.querySelector(sel) as HTMLSelectElement | null;
      if (!select) return '';
      if (String(select.value || '').trim() !== other) {
        select.value = other;
        select.dispatchEvent(new Event('input', { bubbles: true }));
        select.dispatchEvent(new Event('change', { bubbles: true }));
      }
      return select.value === other ? other : '';
    },
    { sel: panelSelect, other: otherValue }
  );
  expect(applied, 'company switch: failed to apply alternative company option').toBe(otherValue);

  await expect
    .poll(
      async () =>
        page.evaluate((sel: string) => {
          const select = document.querySelector(sel) as HTMLSelectElement | null;
          return String(select?.value || '').trim();
        }, panelSelect),
      { timeout: 5_000 }
    )
    .toBe(otherValue);
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
 * Retries the full open→select→apply path: panel-open refreshToken can race with a thin click path.
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

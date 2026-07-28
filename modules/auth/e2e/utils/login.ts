// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { expect, type Page } from '@playwright/test';
import { waitForGrpcWebUnaryOk } from './grpcweb.ts';

/**
 * Log in as the auth e2e fixture admin (`e2e-admin` / `e2e-admin`).
 *
 * Hardens against a known race: Login.vue `ensureAuthReady` / nprogress can still
 * be settling when Playwright fills and clicks, so the submit is ignored and the
 * suite stays on `/web/login?redirect=...`.
 */
export async function loginAsE2EAdmin(page: Page, baseURL: string): Promise<void> {
  const runOnce = async () => {
    await page.goto(`${baseURL}/web/auth/users`, { waitUntil: 'domcontentloaded' });

    const username = page.getByPlaceholder(/username/i);
    await expect(username).toBeVisible({ timeout: 10_000 });

    // Route guard / auth init briefly marks the document busy (see CI flake logs).
    await page
      .waitForFunction(() => !document.documentElement.classList.contains('nprogress-busy'), undefined, {
        timeout: 15_000,
      })
      .catch(() => undefined);

    const submit = page.locator('button[type="submit"]');
    await expect(submit).toBeEnabled({ timeout: 10_000 });

    await username.fill('e2e-admin');
    await page.getByPlaceholder(/password/i).fill('e2e-admin');

    const loginOk = waitForGrpcWebUnaryOk(page, '/auth.User/Login', { timeoutMs: 20_000 });
    await submit.click();
    await loginOk;

    await expect(page).toHaveURL(/\/web\/auth\/users/, { timeout: 15_000 });
  };

  try {
    await runOnce();
  } catch {
    // One retry absorbs residual init races without masking persistent failures.
    await runOnce();
  }
}

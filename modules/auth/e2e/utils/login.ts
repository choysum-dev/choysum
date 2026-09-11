// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { expect, type Page } from '@choysum/e2e';
import { waitForGrpcWebUnaryOk } from './grpcweb.ts';

/**
 * Log in as the auth e2e fixture admin (`e2e-admin` / `e2e-admin`).
 *
 * Hardens against known races:
 * - Login.vue `ensureAuthReady` / nprogress can still be settling when the runner
 *   fills and clicks, so the submit is ignored and the suite stays on
 *   `/web/login?redirect=...`.
 * - A successful Login can navigate away before CDP fetches the response body
 *   (`bodyGone`), leaving `waitForResponse` hanging until timeout even though
 *   auth already landed on `/web/auth/users`.
 */
export async function loginAsE2EAdmin(page: Page, baseURL: string): Promise<void> {
  const runOnce = async () => {
    // Drop prior test auth from the shared browser profile (workers=1).
    await page.goto(`${baseURL}/web/login`, { waitUntil: 'domcontentloaded' });
    await page.evaluate(() => {
      try {
        localStorage.clear();
      } catch {
        // ignore
      }
      try {
        sessionStorage.clear();
      } catch {
        // ignore
      }
    });

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

    // Heavier module closures (partner+) can still be settling SQLite writers at first Login.
    const loginOk = waitForGrpcWebUnaryOk(page, '/auth.User/Login', { timeoutMs: 45_000 });
    await submit.click();
    // Prefer Login OK, but accept post-login URL when CDP body is dropped by navigation.
    try {
      await Promise.race([loginOk, page.waitForURL(/\/web\/auth\/users/, { timeout: 45_000 })]);
    } catch (err) {
      try {
        await expect(page).toHaveURL(/\/web\/auth\/users/, { timeout: 15_000 });
      } catch {
        throw err;
      }
    } finally {
      void loginOk.catch(() => undefined);
    }

    await expect(page).toHaveURL(/\/web\/auth\/users/, { timeout: 30_000 });
  };

  let lastErr: unknown;
  for (let attempt = 1; attempt <= 3; attempt++) {
    try {
      await runOnce();
      return;
    } catch (err) {
      lastErr = err;
      if (attempt < 3) {
        // Give SQLite writers / auth init a beat before the next attempt.
        await page.waitForTimeout(250 * attempt);
      }
    }
  }
  throw lastErr;
}

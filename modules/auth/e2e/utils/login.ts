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
 * - Under CI load the first submit may be ignored; re-click while still on login
 *   and the button is not loading instead of burning a full waitForResponse timeout.
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

    // Route guard / auth init briefly marks the document busy; the submit button
    // also shows Element Plus loading while loginImpl awaits initInFlight.
    await page
      .waitForFunction(
        () => {
          if (document.documentElement.classList.contains('nprogress-busy')) return false;
          const btn = document.querySelector('button[type="submit"]');
          if (!btn) return false;
          if (btn.classList.contains('is-loading')) return false;
          return !('disabled' in btn && btn.disabled);
        },
        undefined,
        { timeout: 30_000 }
      )
      .catch(() => undefined);

    const submit = page.locator('button[type="submit"]');
    await expect(submit).toBeEnabled({ timeout: 10_000 });

    const fillCredentials = async () => {
      await username.fill('e2e-admin');
      await page.getByPlaceholder(/password/i).fill('e2e-admin');
    };
    await fillCredentials();

    // Heavier module closures (partner+) can still be settling SQLite writers at first Login.
    const loginTimeoutMs = 60_000;
    const loginOk = waitForGrpcWebUnaryOk(page, '/auth.User/Login', { timeoutMs: loginTimeoutMs });
    await submit.click();

    let loginSucceeded = false;
    let urlSucceeded = false;
    let loginSettled = false;
    let urlSettled = false;
    let lastErr: unknown;

    // Always-fulfill so the re-click loop is not aborted by the first timeout.
    const loginWait = loginOk.then(
      () => {
        loginSucceeded = true;
        loginSettled = true;
      },
      (err: unknown) => {
        lastErr = err;
        loginSettled = true;
      }
    );
    const urlWait = page.waitForURL(/\/web\/auth\/users/, { timeout: loginTimeoutMs }).then(
      () => {
        urlSucceeded = true;
        urlSettled = true;
      },
      (err: unknown) => {
        lastErr = err;
        urlSettled = true;
      }
    );

    const deadline = Date.now() + loginTimeoutMs;
    while (!(loginSucceeded || urlSucceeded) && Date.now() < deadline) {
      if (loginSettled && urlSettled) break;
      // Omit already-settled waiters so a failed Login does not spin the loop
      // at microtask speed instead of waiting on the 4s re-click backoff.
      const pendingWaiters: Promise<unknown>[] = [page.waitForTimeout(4_000)];
      if (!loginSettled) pendingWaiters.push(loginWait);
      if (!urlSettled) pendingWaiters.push(urlWait);
      await Promise.race(pendingWaiters);
      if (loginSucceeded || urlSucceeded) break;

      const href = String(await page.url());
      if (!/\/web\/login/.test(href)) continue;

      // First submit was ignored (auth init / nprogress). Re-click only when the
      // button is idle so we do not stack parallel loginImpl calls mid-flight.
      const canRetry = await page.evaluate(() => {
        const btn = document.querySelector('button[type="submit"]');
        if (!btn || btn.classList.contains('is-loading')) return false;
        return !('disabled' in btn && btn.disabled);
      });
      if (!canRetry) continue;

      await fillCredentials().catch(() => undefined);
      await submit.click().catch(() => undefined);
    }

    if (!(loginSucceeded || urlSucceeded)) {
      try {
        await expect(page).toHaveURL(/\/web\/auth\/users/, { timeout: 15_000 });
      } catch {
        throw lastErr ?? new Error('loginAsE2EAdmin: login did not complete');
      }
    }

    void loginOk.catch(() => undefined);
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

// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { test, expect, page, runtime, randomUUID } from '@choysum/e2e';
import { waitForGrpcWebUnaryOk } from './utils/grpcweb.ts';
import { drainPageErrorBuffer, filterDeniedSignals, installPageErrorBuffer } from './utils/errorBuffer.ts';

async function readAuthAccessToken(): Promise<string> {
  return page.evaluate(() => {
    const raw = localStorage.getItem('choysum.auth') || sessionStorage.getItem('choysum.auth');
    if (!raw) return '';
    try {
      const data = JSON.parse(raw);
      return String(data?.tokens?.accessToken || '');
    } catch {
      return '';
    }
  });
}

async function runRegisterOnce(baseURL: string): Promise<void> {
  await page.goto(`${baseURL}/web/register`, { waitUntil: 'domcontentloaded' });
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
  await page.goto(`${baseURL}/web/register`, { waitUntil: 'domcontentloaded' });
  await installPageErrorBuffer(page);

  const suffix = `${Date.now()}-${randomUUID()}`;
  const username = `e2e-reg-${suffix}`;
  const email = `${username}@example.com`;
  const password = `e2e-pass-${suffix}`;

  await expect(page.getByPlaceholder(/Enter username|请输入用户名|username/i)).toBeVisible({
    timeout: 15_000,
  });

  await page.getByPlaceholder(/Enter username|请输入用户名|username/i).fill(username);
  await page.getByPlaceholder(/Enter email address|请输入邮箱|email/i).fill(email);
  await page.getByPlaceholder(/^Enter password$|^请输入密码$/).fill(password);
  await page.getByPlaceholder(/Re-enter password|请再次输入密码/i).fill(password);

  // Kit checkbox is Reka CheckboxRoot (button), not a native <input type="checkbox">.
  const terms = page.locator(
    '[data-testid="register-terms"] #fld-AgreeTerms, [data-testid="register-terms"] [data-slot="checkbox"]',
  );
  await expect(terms.first()).toBeVisible({ timeout: 10_000 });
  await terms.first().click();
  await expect
    .poll(
      async () =>
        page.evaluate(() => {
          const el = document.querySelector(
            '[data-testid="register-terms"] #fld-AgreeTerms, [data-testid="register-terms"] [data-slot="checkbox"]',
          ) as HTMLElement | null;
          if (!el) return false;
          const aria = el.getAttribute('aria-checked');
          const data = el.getAttribute('data-state');
          return aria === 'true' || data === 'checked';
        }),
      { timeout: 10_000 },
    )
    .toBe(true);

  const submit = page.getByRole('button', { name: /Create Account|创建账户/ });
  await expect(submit).toBeEnabled({ timeout: 10_000 });

  // Arm boot RPC waiters before submit.
  const browseOk = waitForGrpcWebUnaryOk(page, '/auth.User/Browse', { timeoutMs: 30_000 });
  const permOk = waitForGrpcWebUnaryOk(page, '/auth.User/GetPermissionState', { timeoutMs: 30_000 });

  await submit.click();

  await expect.poll(async () => await readAuthAccessToken(), { timeout: 30_000 }).not.toBe('');
  await Promise.all([browseOk, permOk]);
  await page.waitForTimeout(3_000);

  const deniedSignals = filterDeniedSignals(await drainPageErrorBuffer(page));
  expect(deniedSignals).toEqual([]);
}

test('auth: register new user → auto login → no permission_denied on boot RPCs', async () => {
  test.setTimeout(120_000);

  const baseURL = runtime.baseURL;
  try {
    await runRegisterOnce(baseURL);
  } catch {
    // One retry absorbs intermittent sqlite "database is locked" under WAL.
    await runRegisterOnce(baseURL);
  }
});

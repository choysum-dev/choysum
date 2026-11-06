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

async function runSetupOnce(baseURL: string): Promise<void> {
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
  const optionsOk = waitForGrpcWebUnaryOk(page, '/base.Company/GetInitialSetupOptions', { timeoutMs: 60_000 });
  await page.goto(`${baseURL}/web/setup`, { waitUntil: 'domcontentloaded' });
  await installPageErrorBuffer(page);
  await optionsOk;

  // Prefer placeholders (same as register.spec) — getByText regex is flaky across locales.
  await expect(page.getByPlaceholder(/Enter username|请输入用户名|username/i)).toBeVisible({
    timeout: 15_000,
  });
  await expect(page.getByPlaceholder(/Re-enter password|请再次输入密码/i)).toBeVisible({
    timeout: 10_000,
  });

  const suffix = `${Date.now()}-${randomUUID()}`;
  const username = `e2e-setup-${suffix}`.slice(0, 32);
  const password = `e2e-pass-${suffix}`;

  await page.getByPlaceholder(/Enter username|请输入用户名|username/i).fill(username);
  await page.getByPlaceholder(/^Enter password$|^请输入密码$/).fill(password);
  await page.getByPlaceholder(/Re-enter password|请再次输入密码/i).fill(password);

  const next = page.getByRole('button', { name: /^(Next|下一步)$/ });
  await next.click();

  const orgName = page.getByPlaceholder(/Enter organization name|请输入组织名称/);
  await expect(orgName).toBeVisible({ timeout: 10_000 });
  await orgName.fill(`E2E Org ${suffix}`.slice(0, 60));
  await next.click();

  await expect(page.locator('.setup-review-step')).toBeVisible({ timeout: 10_000 });
  await expect
    .poll(
      async () =>
        page.evaluate(
          (name: string) =>
            ((document.querySelector('.setup-review-step') as HTMLElement | null)?.innerText || '').includes(name),
          username,
        ),
      { timeout: 10_000 },
    )
    .toBe(true);

  const submit = page.getByRole('button', { name: /Complete Setup|完成设置/ });
  await expect(submit).toBeEnabled({ timeout: 10_000 });

  const setupOk = waitForGrpcWebUnaryOk(page, '/auth.User/CompleteInitialSetup', { timeoutMs: 60_000 });
  const browseOk = waitForGrpcWebUnaryOk(page, '/auth.User/Browse', { timeoutMs: 60_000 });
  const permOk = waitForGrpcWebUnaryOk(page, '/auth.User/GetPermissionState', { timeoutMs: 60_000 });

  await submit.click();
  await setupOk;
  await browseOk;
  await permOk;

  await expect.poll(async () => await readAuthAccessToken(), { timeout: 30_000 }).not.toBe('');
  await expect
    .poll(async () => page.url(), { timeout: 20_000 })
    .not.toMatch(/\/web\/(setup|login)(\/|$|\?)/);

  const denied = filterDeniedSignals(await drainPageErrorBuffer(page));
  expect(denied).toEqual([]);
}

/**
 * SetupPending → /web/setup (admin → organization → review) → CompleteInitialSetup → Login.
 * Runs only under scenario `setup-pending` (no setup.completed_at fixture).
 */
test('auth setup: complete initial setup then land authenticated', async () => {
  test.skip(runtime.scenario !== 'setup-pending', 'requires setup-pending scenario (no Ready marker)');
  test.setTimeout(180_000);

  const baseURL = runtime.baseURL;
  try {
    await runSetupOnce(baseURL);
  } catch (err) {
    const msg = err instanceof Error ? err.message : String(err ?? '');
    // Only retry transient sqlite lock races; rethrow assertion/setup regressions.
    if (!/database is locked/i.test(msg)) {
      throw err;
    }
    // CompleteInitialSetup may already have succeeded; a full retry would hit
    // ALREADY_COMPLETE. Skip retry once a session token exists.
    if (await readAuthAccessToken()) {
      throw err;
    }
    await runSetupOnce(baseURL);
  }
});

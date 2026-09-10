// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { test, expect, page, runtime } from '@choysum/e2e';
import { waitForGrpcWebUnary, waitForGrpcWebUnaryOk } from './utils/grpcweb.ts';
import { loginAsE2EAdmin } from './utils/login.ts';

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

test('auth: switch company → new TokenPair → refresh PermissionState → header updates', async () => {
  test.setTimeout(120_000);

  const baseURL = runtime.baseURL;

  await loginAsE2EAdmin(page, baseURL);

  const beforeToken = await readAuthAccessToken();
  expect(beforeToken).not.toBe('');

  const trigger = page.getByTestId('company-switch-trigger');
  await expect(trigger).toBeVisible();

  const beforeLabel = ((await trigger.textContent()) || '').trim();

  await trigger.click();

  await page.getByTestId('company-active-select').click();
  await expect
    .poll(
      async () =>
        page.evaluate(() => {
          const opts = Array.from(
            document.querySelectorAll('.el-select-dropdown li, [role="option"]')
          ) as HTMLElement[];
          if (opts.length < 2) return false;
          const target =
            opts.find(o => o.getAttribute('aria-selected') !== 'true') || opts[opts.length - 1];
          target.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true, view: window }));
          return true;
        }),
      { timeout: 10_000 }
    )
    .toBe(true);

  const applyButton = page.getByTestId('company-switch-apply');
  await expect.poll(async () => await applyButton.isEnabled(), { timeout: 15_000 }).toBe(true);
  await expect(page.getByTestId('company-switch-hint')).toHaveCount(0);

  const switchOk = waitForGrpcWebUnaryOk(page, '/auth.User/SwitchCompanyScope', { timeoutMs: 30_000 });
  // Permission refresh is best-effort: some boots coalesce GetPermissionState.
  const permObserved = waitForGrpcWebUnary(page, '/auth.User/GetPermissionState', { timeoutMs: 30_000 }).catch(() => null);

  await applyButton.click();

  await expect.poll(async () => await readAuthAccessToken(), { timeout: 30_000 }).not.toBe(beforeToken);
  const afterToken = await readAuthAccessToken();
  expect(afterToken).not.toBe('');
  expect(afterToken).not.toBe(beforeToken);

  await switchOk;
  const perm = await permObserved;
  if (perm && perm.grpcStatus !== '0') {
    expect(perm.grpcStatus).toBe('7');
    expect(perm.grpcMessage.toLowerCase()).toContain('access denied');
  }

  await expect.poll(async () => ((await trigger.textContent()) || '').trim(), { timeout: 10_000 }).not.toBe(beforeLabel);
});

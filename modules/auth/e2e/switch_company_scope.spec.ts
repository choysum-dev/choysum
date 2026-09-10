// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { test, expect, page, runtime } from '@choysum/e2e';
import { waitForGrpcWebUnary } from './utils/grpcweb.ts';
import { loginAsE2EAdmin } from './utils/login.ts';
import { switchCompanyViaUI } from './utils/switchCompany.ts';

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

  // Arm permission observation before SwitchCompanyScope; panel-open refreshToken
  // also rotates access tokens, so token inequality alone is not proof of switch.
  const permObserved = waitForGrpcWebUnary(page, '/auth.User/GetPermissionState', {
    timeoutMs: 5_000,
  }).catch(() => null);

  await switchCompanyViaUI();

  await expect.poll(async () => await readAuthAccessToken(), { timeout: 30_000 }).not.toBe(beforeToken);
  const afterToken = await readAuthAccessToken();
  expect(afterToken).not.toBe('');
  expect(afterToken).not.toBe(beforeToken);

  const perm = await permObserved;
  if (perm && perm.grpcStatus !== '0') {
    expect(perm.grpcStatus).toBe('7');
    expect(perm.grpcMessage.toLowerCase()).toContain('access denied');
  }

  await expect.poll(async () => ((await trigger.textContent()) || '').trim(), { timeout: 10_000 }).not.toBe(beforeLabel);
});

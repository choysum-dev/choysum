// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { test, expect, page, runtime } from '@choysum/e2e';
import { loginAsE2EAdmin } from './utils/login';

/**
 * Smoke test for auth module:
 * - Uses injected runtime from the e2e host
 * - Attempts login with the fixture user (e2e-admin / e2e-admin)
 * - Verifies redirection away from the login page
 */

test('auth smoke: login with fixture user and navigate to /web/auth/users', async () => {
  const baseURL = runtime.baseURL;

  await loginAsE2EAdmin(page, baseURL);

  // Verify we're no longer on the login page
  await expect(page.getByText('User Login')).toHaveCount(0);

  // Basic sanity: page should render successfully (no crash)
  await expect(page.locator('body')).toBeVisible();
});

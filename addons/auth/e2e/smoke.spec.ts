// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { test, expect } from '@playwright/test';
import fs from 'node:fs';

/**
 * Smoke test for auth module:
 * - Reads runtime.json from CHOYSUM_E2E_RUNTIME_JSON
 * - Attempts login with the fixture user (e2e-admin / e2e-admin)
 * - Verifies redirection to /web/auth/users
 */

type RuntimeInfo = {
  baseURL: string;
  specsDir: string;
  module: string;
  scenario: string;
  fixtures: string[];
};

function readRuntimeInfo(): RuntimeInfo {
  const runtimePath = process.env.CHOYSUM_E2E_RUNTIME_JSON;
  if (!runtimePath) {
    throw new Error('CHOYSUM_E2E_RUNTIME_JSON env var not set');
  }
  const raw = fs.readFileSync(runtimePath, 'utf-8');
  return JSON.parse(raw) as RuntimeInfo;
}

test('auth smoke: login with fixture user and navigate to /web/auth/users', async ({ page }) => {
  const runtime = readRuntimeInfo();
  const baseURL = runtime.baseURL;

  // Navigate to a protected route; auth guard should redirect to login
  await page.goto(`${baseURL}/web/auth/users`, { waitUntil: 'domcontentloaded' });

  // Wait for login form to appear
  await page.waitForSelector('input[placeholder*="username"]', { timeout: 10000 });

  // Fill login credentials (from fixture: e2e-admin / e2e-admin)
  await page.getByPlaceholder(/username/i).fill('e2e-admin');
  await page.getByPlaceholder(/password/i).fill('e2e-admin');

  // Submit login form
  await page.locator('button[type="submit"]').click();

  // After login, should land on /web/auth/users
  await expect(page).toHaveURL(/\/web\/auth\/users/, { timeout: 15000 });

  // Verify we're no longer on the login page
  await expect(page.getByText('User Login')).toHaveCount(0);

  // Basic sanity: page should render successfully (no crash)
  await expect(page.locator('body')).toBeVisible();
});

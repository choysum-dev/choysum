// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { test, expect } from '@playwright/test';
import fs from 'node:fs';
import { randomUUID } from 'node:crypto';
import { waitForGrpcWebUnaryOk } from './utils/grpcweb';

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

async function readAuthAccessToken(page: any): Promise<string> {
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

test('auth: register new user → auto login → no permission_denied on boot RPCs', async ({ page }) => {
  test.setTimeout(120_000);

  const deniedSignals: string[] = [];

  page.on('pageerror', err => {
    const msg = err?.message || String(err);
    if (/permission_denied|access denied/i.test(msg)) {
      deniedSignals.push(`[pageerror] ${msg}`);
    }
  });

  page.on('console', msg => {
    if (msg.type() !== 'error') return;
    const text = msg.text();
    if (/permission_denied|access denied|\/auth\.User\/(Browse|GetPermissionState)/i.test(text)) {
      deniedSignals.push(`[console.error] ${text}`);
    }
  });

  page.on('requestfailed', req => {
    const failure = req.failure();
    const text = `${req.method()} ${req.url()} ${failure?.errorText || ''}`;
    if (/permission_denied|access denied|\/auth\.User\/(Browse|GetPermissionState)/i.test(text)) {
      deniedSignals.push(`[requestfailed] ${text}`);
    }
  });

  const runtime = readRuntimeInfo();
  const baseURL = runtime.baseURL;

  const suffix = `${Date.now()}-${randomUUID()}`;
  const username = `e2e-reg-${suffix}`;
  const email = `${username}@example.com`;
  const password = `e2e-pass-${suffix}`;

  await page.goto(`${baseURL}/web/register`, { waitUntil: 'domcontentloaded' });

  await page.waitForSelector('input[placeholder*="username"]', { timeout: 15_000 });

  await page.getByPlaceholder(/username/i).fill(username);
  await page.getByPlaceholder(/email address/i).fill(email);
  await page.getByPlaceholder(/^Enter password$/).fill(password);
  await page.getByPlaceholder(/Re-enter password/i).fill(password);

  // Agree terms
  const agreeLabel = page.locator('label.el-checkbox', { hasText: 'I have read and agree to' });
  await expect(agreeLabel).toBeVisible();
  await agreeLabel.locator('.el-checkbox__inner').click();
  await expect(agreeLabel.locator('input.el-checkbox__original')).toBeChecked();

  // Submit
  const submit = page.getByRole('button', { name: /Create Account/ });
  await expect(submit).toBeEnabled();
  await submit.click();

  // Hard assertions: ensure boot chain actually called these RPCs successfully.
  const browseOk = waitForGrpcWebUnaryOk(page, '/auth.User/Browse', { timeoutMs: 30_000 });
  const permOk = waitForGrpcWebUnaryOk(page, '/auth.User/GetPermissionState', { timeoutMs: 30_000 });

  // Auto-login + redirect should leave us with a token.
  await expect.poll(async () => await readAuthAccessToken(page), { timeout: 30_000 }).not.toBe('');

  await Promise.all([browseOk, permOk]);

  // Give boot sequence time to call Browse/GetPermissionState.
  await page.waitForTimeout(3_000);

  expect(deniedSignals).toEqual([]);
});

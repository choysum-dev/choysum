// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { test, expect } from '@playwright/test';
import fs from 'node:fs';
import { loginAsE2EAdmin } from '../../auth/e2e/utils/login.ts';

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

test('partner import: list page exposes import entry', async ({ page }) => {
  const runtime = readRuntimeInfo();
  await loginAsE2EAdmin(page, runtime.baseURL);

  await page.goto(`${runtime.baseURL}/web/partner/partners`);
  await expect(page.getByRole('button', { name: /Import CSV/i })).toBeVisible();
});

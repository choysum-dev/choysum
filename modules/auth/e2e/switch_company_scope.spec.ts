// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { test, expect } from '@playwright/test';
import fs from 'node:fs';
import { waitForGrpcWebUnary, waitForGrpcWebUnaryOk } from './utils/grpcweb.ts';
import { loginAsE2EAdmin } from './utils/login.ts';

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

test('auth: switch company → new TokenPair → refresh PermissionState → header updates', async ({ page }) => {
  test.setTimeout(120_000);

  page.on('pageerror', err => {
    // surfaced in Playwright output
    console.log(`[pageerror] ${err?.message || String(err)}`);
  });
  page.on('console', msg => {
    if (msg.type() === 'error') {
      console.log(`[console.error] ${msg.text()}`);
    }
  });
  page.on('requestfailed', req => {
    const failure = req.failure();
    console.log(`[requestfailed] ${req.method()} ${req.url()} ${failure?.errorText || ''}`);
  });

  const runtime = readRuntimeInfo();
  const baseURL = runtime.baseURL;

  await loginAsE2EAdmin(page, baseURL);

  // Baseline token
  const beforeToken = await readAuthAccessToken(page);
  expect(beforeToken).not.toBe('');

  // Open switcher
  const trigger = page.getByTestId('company-switch-trigger');
  await expect(trigger).toBeVisible();

  const beforeLabel = ((await trigger.textContent()) || '').trim();

  await trigger.click();

  // Choose a different company than current, so the switch actually triggers.
  await page.getByTestId('company-active-select').click();
  const options = page.getByRole('option');
  await expect.poll(async () => await options.count(), { timeout: 10_000 }).toBeGreaterThan(1);
  const count = await options.count();
  expect(count, 'need at least 2 companies in selector').toBeGreaterThan(1);
  let picked = false;
  for (let i = 0; i < count; i++) {
    const opt = options.nth(i);
    const selected = await opt.getAttribute('aria-selected');
    if (selected === 'true') continue;
    await opt.click();
    picked = true;
    break;
  }
  expect(picked, 'expected an unselected company option to click').toBe(true);

  // Apply (poll so a late open-sync refresh cannot race past a one-shot toBeEnabled).
  const applyButton = page.getByTestId('company-switch-apply');
  await expect.poll(async () => await applyButton.isEnabled(), { timeout: 15_000 }).toBe(true);
  await expect(page.getByTestId('company-switch-hint')).toHaveCount(0);

  // Hard assertions: switching should call the RPC(s) successfully.
  const switchOk = waitForGrpcWebUnaryOk(page, '/auth.User/SwitchCompanyScope', { timeoutMs: 30_000 });
  // GetPermissionState refresh is fail-soft in authStore.switchCompanyScope; assert the call is observed
  // and allow transient access-denied (grpc-status=7) without failing this UI flow test.
  const permObserved = waitForGrpcWebUnary(page, '/auth.User/GetPermissionState', { timeoutMs: 30_000 });

  await applyButton.click();

  // TokenPair should change
  await expect.poll(async () => await readAuthAccessToken(page), { timeout: 30_000 }).not.toBe(beforeToken);
  const afterToken = await readAuthAccessToken(page);
  expect(afterToken).not.toBe('');
  expect(afterToken).not.toBe(beforeToken);

  const [, perm] = await Promise.all([switchOk, permObserved]);
  if (perm.grpcStatus !== '0') {
    expect(perm.grpcStatus).toBe('7');
    expect(perm.grpcMessage.toLowerCase()).toContain('access denied');
  }

  // UX change: header label should change.
  await expect.poll(async () => ((await trigger.textContent()) || '').trim(), { timeout: 10_000 }).not.toBe(beforeLabel);
});

/**
 * Read active + an alternate allowed company id from the live auth store.
 */
async function readCompanyIdsFromAuthStore(page: any): Promise<{ active: string; other: string }> {
  return page.evaluate(() => {
    const app = (document.querySelector('#app') as any)?.__vue_app__;
    const pinia = app?.config?.globalProperties?.$pinia || null;
    const store = pinia?._s?.get?.('choysum.auth') || null;
    const meta = store?.identity?.metadata ?? {};
    const active = String(meta.activeCompanyId ?? '').trim();
    const allowed = [
      ...(Array.isArray(meta.allowedCompanyIds) ? meta.allowedCompanyIds : []),
      ...(Array.isArray(meta.enabledCompanyIds) ? meta.enabledCompanyIds : []),
    ]
      .map((x: any) => String(x ?? '').trim())
      .filter(Boolean);
    const other = allowed.find((id: string) => id && id !== active) || '';
    return { active, other };
  });
}

/**
 * Simulate a token-refresh updating JWT company metadata while the switcher is open.
 */
async function patchAuthIdentityActiveCompany(page: any, nextActiveCompanyId: string): Promise<boolean> {
  return page.evaluate(activeId => {
    const app = (document.querySelector('#app') as any)?.__vue_app__;
    const pinia = app?.config?.globalProperties?.$pinia || null;
    const store = pinia?._s?.get?.('choysum.auth');
    if (!store?.identity) return false;

    const prev = store.identity;
    const meta = { ...((prev as any).metadata ?? {}) };
    meta.activeCompanyId = activeId;
    store.identity = { ...prev, metadata: meta };
    return String(store.identity?.metadata?.activeCompanyId ?? '') === activeId;
  }, nextActiveCompanyId);
}

test('auth: open company-switch panel drafts survive JWT metadata refresh', async ({ page }) => {
  test.setTimeout(120_000);

  const runtime = readRuntimeInfo();
  const baseURL = runtime.baseURL;

  await loginAsE2EAdmin(page, baseURL);

  const { other: otherCompanyId } = await readCompanyIdsFromAuthStore(page);
  test.skip(!otherCompanyId, 'Need two companies in the auth allowlist');

  const trigger = page.getByTestId('company-switch-trigger');
  await expect(trigger).toBeVisible();
  await trigger.click();

  const panel = page.getByTestId('company-switch-panel');
  await expect(panel).toBeVisible();

  // Wait until select options are seeded (same readiness as the happy-path switcher test).
  await page.getByTestId('company-active-select').click();
  const options = page.getByRole('option');
  await expect.poll(async () => await options.count(), { timeout: 10_000 }).toBeGreaterThan(1);
  await page.keyboard.press('Escape');
  await expect(panel).toBeVisible();

  const applyButton = page.getByTestId('company-switch-apply');
  // Fresh open + no user edit → Apply disabled ("No changes to apply").
  await expect(applyButton).toBeDisabled();

  const patched = await patchAuthIdentityActiveCompany(page, otherCompanyId);
  expect(patched, 'expected pinia auth identity metadata patch').toBe(true);

  // Drafts must stay on the open-panel seed; JWT active moved → dirty → Apply enabled.
  await expect.poll(async () => await applyButton.isEnabled(), { timeout: 10_000 }).toBe(true);
  await expect(page.getByTestId('company-switch-hint')).toHaveCount(0);
  await expect(panel).toBeVisible();
});

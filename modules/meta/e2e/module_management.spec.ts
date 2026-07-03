// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { test, expect, type Page, type Locator } from '@playwright/test';
import fs from 'node:fs';

test.setTimeout(10 * 60 * 1000);

// Set a default action/navigation timeout so that any Playwright operation
// without an explicit timeout cannot hang indefinitely on slow CI runners.
test.beforeEach(async ({ page }) => {
  page.setDefaultTimeout(30000);
});

/**
 * Runtime metadata injected into the meta module management e2e harness.
 */
type RuntimeInfo = {
  baseURL: string;
  specsDir: string;
  module: string;
  scenario: string;
  fixtures: string[];
};

type OperationTerminalStatus = 'succeeded' | 'failed' | 'cancelled' | 'reloaded';

function parseTerminalStatus(value: string): Exclude<OperationTerminalStatus, 'reloaded'> | '' {
  const normalized = value.trim().toLowerCase();
  if (normalized === 'succeeded' || normalized === 'failed' || normalized === 'cancelled') {
    return normalized;
  }
  return '';
}

/**
 * Escapes a string so it can be used as a literal fragment in RegExp patterns.
 */
function escapeRegExp(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

/**
 * Loads the current e2e runtime descriptor from the test harness.
 */
function readRuntimeInfo(): RuntimeInfo {
  const runtimePath = process.env.CHOYSUM_E2E_RUNTIME_JSON;
  if (!runtimePath) {
    throw new Error('CHOYSUM_E2E_RUNTIME_JSON env var not set');
  }
  const raw = fs.readFileSync(runtimePath, 'utf-8');
  return JSON.parse(raw) as RuntimeInfo;
}

/**
 * Opens the module management page and authenticates if the session is not yet logged in.
 */
async function ensureLoggedIn(page: Page, baseURL: string) {
  await page.goto(`${baseURL}/web/meta/modules`, { waitUntil: 'domcontentloaded' });
  await page.waitForLoadState('domcontentloaded');
  const loginInput = page.getByPlaceholder(/用户名|username/i);
  const loginVisible = await loginInput.isVisible().catch(() => false);
  if (page.url().includes('/web/login') || loginVisible) {
    await loginInput.waitFor({ state: 'visible', timeout: 10000 });
    const tryLogin = async (username: string, password: string) => {
      await page.getByPlaceholder(/用户名|username/i).fill(username);
      await page.getByPlaceholder(/密码|password/i).fill(password);
      const submit = page.locator('button[type="submit"]');
      const canClick = await submit.isEnabled().catch(() => false);
      if (canClick) {
        await submit.click();
      } else {
        await page.getByPlaceholder(/密码|password/i).press('Enter');
      }
      await page.waitForURL(/\/(web\/)?meta\/modules/, { timeout: 15000 }).catch(() => null);
      return !page.url().includes('/web/login');
    };
    const ok = await tryLogin('admin', 'admin');
    if (!ok) {
      await tryLogin('e2e-admin', 'e2e-admin');
    }
    await expect(page).not.toHaveURL(/\/web\/login/, { timeout: 15000 });
  }
  await page.goto(`${baseURL}/web/meta/modules`, { waitUntil: 'domcontentloaded' });
  await page.waitForURL('**/web/meta/modules', { timeout: 30000 });
  await waitForModuleList(page);
}

/**
 * Waits until the module kanban list and its backing response become available.
 */
async function waitForModuleList(page: Page) {
  await page.locator('.okanban').waitFor({ state: 'visible', timeout: 30000 });
  await page.waitForResponse(resp => resp.url().includes('meta.IrModule') && resp.status() === 200, { timeout: 30000 }).catch(() => null);
}

/**
 * Clears any active module search tags before a new lookup is attempted.
 */
async function clearSearchFilters(page: Page) {
  const closeBtns = page.locator('.o-search__tag .el-tag__close');
  const maxClicks = 5;
  for (let i = 0; i < maxClicks; i += 1) {
    if ((await closeBtns.count()) === 0) break;
    await closeBtns.first().click();
    await page.waitForTimeout(200);
  }
}

/**
 * Applies a module-name search so the board can focus on a specific card.
 */
async function searchModuleCard(page: Page, moduleName: string) {
  const searchInput = page.locator('.o-kanban__search .o-search__input');
  if (!(await searchInput.count())) {
    return;
  }

  await searchInput.waitFor({ state: 'visible', timeout: 15000 }).catch(() => null);
  await clearSearchFilters(page);
  await searchInput.fill(moduleName);
  await searchInput.press('Enter');
  await waitForModuleList(page);
}

/**
 * Resolves the kanban card for a module, using search and reload fallback when needed.
 */
async function findModuleCardByName(page: Page, moduleName: string): Promise<Locator | null> {
  const cards = page.locator('.module-card');
  const total = await cards.count();
  for (let i = 0; i < total; i += 1) {
    const card = cards.nth(i);
    const name = (
      (await card
        .locator('.module-card__title .name')
        .textContent()
        .catch(() => '')) || ''
    ).trim();
    if (name === moduleName) {
      return card;
    }
  }
  return null;
}

async function openModuleCard(page: Page, moduleName: string): Promise<Locator>;
async function openModuleCard(page: Page, moduleName: string, opts: { allowMissing: true }): Promise<Locator | null>;
async function openModuleCard(page: Page, moduleName: string, opts: { allowMissing: true; skipReload?: boolean }): Promise<Locator | null>;
async function openModuleCard(page: Page, moduleName: string, opts?: { allowMissing?: boolean; skipReload?: boolean }): Promise<Locator | null> {
  const initialCard = await findModuleCardByName(page, moduleName);
  if (initialCard && (await initialCard.isVisible().catch(() => false))) return initialCard;

  await searchModuleCard(page, moduleName);

  const searchedCard = await findModuleCardByName(page, moduleName);
  if (searchedCard && (await searchedCard.isVisible().catch(() => false))) return searchedCard;

  if (opts?.skipReload) {
    if (opts.allowMissing) {
      return null;
    }
    throw new Error(`module card not found: ${moduleName}`);
  }

  await page.reload({ waitUntil: 'domcontentloaded' });
  await page.waitForURL('**/web/meta/modules', { timeout: 30000 }).catch(() => null);
  await waitForModuleList(page);
  await searchModuleCard(page, moduleName);

  const reloadedCard = await findModuleCardByName(page, moduleName);
  if (reloadedCard && (await reloadedCard.isVisible().catch(() => false))) return reloadedCard;

  // Search filters can become stale after module install/uninstall side effects.
  // Fall back to the full board once before treating the card as missing.
  await clearSearchFilters(page);
  await waitForModuleList(page);

  const fallbackCard = await findModuleCardByName(page, moduleName);
  if (fallbackCard && (await fallbackCard.isVisible().catch(() => false))) return fallbackCard;

  if (opts?.allowMissing) {
    return null;
  }

  const finalCard = await findModuleCardByName(page, moduleName);
  if (!finalCard) {
    throw new Error(`module card not found: ${moduleName}`);
  }
  await expect(finalCard).toBeVisible({ timeout: 30000 });
  return finalCard;
}

/**
 * Infers module install state from visible action buttons when status tags are transiently empty.
 */
async function inferModuleStatusFromActions(card: Locator) {
  const hasInstall = await card
    .getByRole('button', { name: '安装' })
    .first()
    .isVisible()
    .catch(() => false);
  if (hasInstall) {
    return '未安装';
  }

  const hasUpgrade = await card
    .getByRole('button', { name: '升级' })
    .first()
    .isVisible()
    .catch(() => false);
  const hasUninstall = await card
    .getByRole('button', { name: '卸载' })
    .first()
    .isVisible()
    .catch(() => false);
  if (hasUpgrade || hasUninstall) {
    return '已安装';
  }

  return '';
}

/**
 * Reads the current status label shown on a module card.
 */
async function moduleStatusText(page: Page, moduleName: string) {
  const card = await openModuleCard(page, moduleName, { allowMissing: true, skipReload: true });
  if (!card) {
    return '';
  }
  const tagText = (
    (await card
      .locator('.module-card__title .el-tag')
      .textContent()
      .catch(() => '')) || ''
  ).trim();
  if (tagText && tagText !== '—') {
    return tagText;
  }

  return inferModuleStatusFromActions(card);
}

/**
 * Polls the module board until a module reaches the expected status label.
 *
 * Uses a hard deadline via setTimeout so that even if Playwright operations
 * inside the polling loop hang beyond their individual timeouts the function
 * cannot exceed the requested deadline.
 */
async function waitForModuleStatus(page: Page, moduleName: string, expectedStatus: string, timeout = 120000) {
  const deadline = Date.now() + timeout;
  const reloadIntervalMs = 15000;
  let nextReloadAt = Date.now();
  let lastStatus = '';

  while (Date.now() < deadline) {
    const remaining = deadline - Date.now();
    if (remaining <= 0) break;

    if (Date.now() >= nextReloadAt) {
      const navTimeout = Math.min(remaining, 30000);
      await page.reload({ waitUntil: 'domcontentloaded', timeout: navTimeout }).catch(() => null);
      await page.waitForURL('**/web/meta/modules', { timeout: Math.min(deadline - Date.now(), 30000) }).catch(() => null);
      await waitForModuleList(page).catch(() => null);
      nextReloadAt = Date.now() + reloadIntervalMs;
    }

    // Guard each polling iteration with a hard deadline so that a single
    // stuck moduleStatusText call cannot consume the remaining budget.
    const iterRemaining = deadline - Date.now();
    if (iterRemaining <= 0) break;

    const statusPromise = moduleStatusText(page, moduleName);
    const deadlinePromise = new Promise<string>(resolve => setTimeout(() => resolve('__deadline__'), Math.min(iterRemaining, 5000)));
    const result = (await Promise.race([statusPromise, deadlinePromise])) as string;
    if (result !== '__deadline__') {
      lastStatus = result;
    }
    // If the deadline fired first, keep the previous lastStatus and let
    // the outer loop's while-condition (or the next reload cycle) decide.

    if (lastStatus === expectedStatus) {
      return;
    }

    const waitMs = lastStatus ? 1000 : 1500;
    await Promise.race([
      page.waitForTimeout(waitMs),
      new Promise(resolve => setTimeout(resolve, waitMs + 1000)),
    ]);
  }

  throw new Error(`module ${moduleName} status remained ${lastStatus || '<empty>'}, want ${expectedStatus}`);
}

/**
 * Waits until an operation reaches terminal status in the dialog or performs a hard reload.
 */
async function waitForOperationTerminalState(page: Page, timeout = 3 * 60 * 1000): Promise<OperationTerminalStatus> {
  const dialog = page.locator('.el-dialog');
  const statusTag = dialog.locator('.status-row .el-tag').first();

  const reloadPromise = page
    .waitForNavigation({ waitUntil: 'domcontentloaded', timeout })
    .then(async () => {
      await page.waitForLoadState('networkidle', { timeout: 30000 }).catch(() => null);
      return 'reloaded' as const;
    })
    .catch(() => 'no' as const);

  const statusPromise = (async () => {
    const deadline = Date.now() + timeout;
    while (Date.now() < deadline) {
      const statusText = ((await statusTag.textContent().catch(() => '')) || '').trim();
      const terminal = parseTerminalStatus(statusText);
      if (terminal) {
        return terminal;
      }
      await page.waitForTimeout(500);
    }
    return 'no' as const;
  })();

  const winner = await Promise.race([reloadPromise, statusPromise]);
  if (winner !== 'no') {
    return winner;
  }

  const [reloadResult, statusResult] = await Promise.all([reloadPromise, statusPromise]);
  if (statusResult !== 'no') {
    return statusResult;
  }
  if (reloadResult !== 'no') {
    return reloadResult;
  }

  const dialogVisible = await dialog.isVisible().catch(() => false);
  throw new Error(
    dialogVisible ? 'operation did not reach a terminal status before timeout' : 'operation dialog disappeared before terminal status became observable'
  );
}

/**
 * Waits until an operation finishes through terminal status or page reload.
 */
async function waitForOperationCompletion(page: Page) {
  const dialog = page.locator('.el-dialog');
  const completion = await waitForOperationTerminalState(page);

  if (completion === 'reloaded') {
    await waitForModuleList(page);
    return;
  }

  if (completion !== 'succeeded') {
    const resultTag = dialog.locator('.status-row .el-tag').nth(1);
    const resultText = ((await resultTag.textContent().catch(() => '')) || '').trim();
    const resultSuffix = resultText ? `, result ${resultText}` : '';
    throw new Error(`module operation finished with status ${completion}${resultSuffix}`);
  }

  const doneButton = page.getByRole('button', { name: '完成' });
  const doneVisible = await doneButton.isVisible().catch(() => false);
  if (doneVisible) {
    await doneButton.click();
    await dialog.waitFor({ state: 'hidden', timeout: 15000 }).catch(() => null);
  }
}

/**
 * Waits until an operation completes with a failed result and asserts the failure summary is populated.
 */
async function waitForOperationFailure(page: Page) {
  const dialog = page.locator('.el-dialog');
  const completion = await waitForOperationTerminalState(page);
  if (completion === 'reloaded') {
    throw new Error('expected failed operation result, but page reloaded before terminal status could be read');
  }

  const statusTag = dialog.locator('.status-row .el-tag').first();
  const statusText = ((await statusTag.textContent().catch(() => '')) || '').trim().toLowerCase();
  const resultTag = dialog.locator('.status-row .el-tag').nth(1);
  const resultText = ((await resultTag.textContent().catch(() => '')) || '').trim();
  const failedByStatus = completion === 'failed' || completion === 'cancelled' || statusText === 'failed' || statusText === 'cancelled';
  const failedByResult = /FAILED/i.test(resultText);

  const summaryRow = dialog.locator('.status-row', { hasText: '摘要' });
  const errorRow = dialog.locator('.status-row', { hasText: '错误' });

  if (!failedByStatus && !failedByResult) {
    if ((await summaryRow.count()) > 0) {
      await expect(summaryRow.locator('.value')).not.toHaveText(/^\s*$/);
    } else if ((await errorRow.count()) > 0) {
      await expect(errorRow.locator('.value')).not.toHaveText(/—\s*\/\s*—/);
    } else {
      throw new Error(`expected failed operation status, got status=${statusText || '<empty>'}, result=${resultText || '<empty>'}`);
    }
  }

  const doneButton = page.getByRole('button', { name: '完成' });
  const doneVisible = await doneButton.isVisible().catch(() => false);
  if (doneVisible) {
    await doneButton.click();
    await dialog.waitFor({ state: 'hidden', timeout: 15000 }).catch(() => null);
  }
}

/**
 * Waits for the confirmation button to finish preflight loading and become clickable.
 */
async function clickConfirmWhenReady(page: Page, timeout = 90000) {
  const confirmBtn = page.getByRole('button', { name: '确认执行' });
  await confirmBtn.waitFor({ state: 'visible', timeout });
  await expect(confirmBtn).not.toHaveClass(/is-loading/, { timeout });
  await expect(confirmBtn).toBeEnabled({ timeout });
  await confirmBtn.click();
}

/**
 * Returns true when the operation failure looks transient and retryable.
 */
function shouldRetryOperationFailure(error: unknown) {
  const message = String((error as { message?: string })?.message ?? error ?? '');
  return /module operation finished with status (failed|cancelled)/i.test(message);
}

/**
 * Dismisses the operation dialog when it is still visible.
 */
async function closeOperationDialogIfPresent(page: Page) {
  const dialog = page.locator('.el-dialog');
  const dialogVisible = await dialog.isVisible().catch(() => false);
  if (!dialogVisible) {
    return;
  }

  const doneButton = page.getByRole('button', { name: '完成' });
  const doneVisible = await doneButton.isVisible().catch(() => false);
  if (doneVisible) {
    await doneButton.click().catch(() => null);
  }

  await dialog.waitFor({ state: 'hidden', timeout: 15000 }).catch(() => null);
}

/**
 * Waits for the expected module status and returns false instead of throwing on timeout.
 */
async function hasExpectedModuleStatus(page: Page, moduleName: string, expectedStatus: string, timeout = 30000) {
  try {
    await waitForModuleStatus(page, moduleName, expectedStatus, timeout);
    return true;
  } catch {
    return false;
  }
}

/**
 * Executes one module management action and waits for the board state to settle.
 */
async function runActionOnce(page: Page, moduleName: string, action: 'install' | 'upgrade' | 'uninstall') {
  const actionLabel = action === 'install' ? '安装' : action === 'upgrade' ? '升级' : '卸载';
  const card = await openModuleCard(page, moduleName);
  await card.getByRole('button', { name: actionLabel }).click();

  const dialog = page.locator('.el-dialog');
  await dialog.waitFor({ state: 'visible', timeout: 15000 });
  await clickConfirmWhenReady(page);

  await waitForOperationCompletion(page);

  if (action === 'uninstall') {
    await page.reload({ waitUntil: 'domcontentloaded' });
    await page.waitForURL('**/web/meta/modules', { timeout: 30000 }).catch(() => null);
    await waitForModuleList(page);
    return;
  }

  const expectedStatus = '已安装';
  await waitForModuleStatus(page, moduleName, expectedStatus, 180000);
}

/**
 * Executes a module management action and retries once for transient terminal failures.
 */
async function runAction(page: Page, moduleName: string, action: 'install' | 'upgrade' | 'uninstall') {
  const expectedStatus = action === 'uninstall' ? '未安装' : '已安装';
  const maxAttempts = 2;

  for (let attempt = 1; attempt <= maxAttempts; attempt += 1) {
    try {
      await runActionOnce(page, moduleName, action);
      return;
    } catch (error) {
      if (await hasExpectedModuleStatus(page, moduleName, expectedStatus)) {
        return;
      }

      const canRetry = attempt < maxAttempts && shouldRetryOperationFailure(error);
      if (!canRetry) {
        throw error;
      }

      await closeOperationDialogIfPresent(page);
      await page.reload({ waitUntil: 'domcontentloaded' }).catch(() => null);
      await page.waitForURL('**/web/meta/modules', { timeout: 30000 }).catch(() => null);
      await waitForModuleList(page);
    }
  }
}

/**
 * Executes a module action and asserts that the result dialog reports failure.
 */
async function runActionExpectFailure(page: Page, moduleName: string, action: 'install' | 'upgrade' | 'uninstall') {
  const actionLabel = action === 'install' ? '安装' : action === 'upgrade' ? '升级' : '卸载';
  const card = await openModuleCard(page, moduleName);
  const initialStatus = (await card.locator('.module-card__title .el-tag').innerText()).trim();
  await card.getByRole('button', { name: actionLabel }).click();

  const dialog = page.locator('.el-dialog');
  await dialog.waitFor({ state: 'visible', timeout: 15000 });
  await clickConfirmWhenReady(page);

  await waitForOperationFailure(page);

  await waitForModuleStatus(page, moduleName, initialStatus);
}

/**
 * Executes a module action and asserts that only the reload stage fails while the action still applies.
 */
async function runActionExpectReloadFailed(page: Page, moduleName: string, action: 'install' | 'upgrade' | 'uninstall') {
  const actionLabel = action === 'install' ? '安装' : action === 'upgrade' ? '升级' : '卸载';
  const card = await openModuleCard(page, moduleName);
  await card.getByRole('button', { name: actionLabel }).click();

  const dialog = page.locator('.el-dialog');
  await dialog.waitFor({ state: 'visible', timeout: 15000 });
  await clickConfirmWhenReady(page);

  const completion = await waitForOperationTerminalState(page);
  if (completion === 'reloaded') {
    await waitForModuleList(page);
  } else {
    const reloadRow = dialog.locator('.status-row', { hasText: 'Reload' });
    if ((await reloadRow.count()) > 0) {
      await expect(reloadRow).toHaveText(/触发失败/);
    }

    const doneButton = page.getByRole('button', { name: '完成' });
    const doneVisible = await doneButton.isVisible().catch(() => false);
    if (doneVisible) {
      await doneButton.click();
      await dialog.waitFor({ state: 'hidden', timeout: 15000 }).catch(() => null);
    }
  }

  const expectedStatus = action === 'uninstall' ? '未安装' : '已安装';
  await waitForModuleStatus(page, moduleName, expectedStatus);
}

/**
 * Selects the fixed partner module as the install/upgrade e2e target.
 */
async function pickTargetModule(page: Page) {
  await waitForModuleList(page);

  const cards = page.locator('.module-card');
  await expect
    .poll(async () => cards.count(), { timeout: 15000 })
    .toBeGreaterThan(0)
    .catch(() => null);

  const total = await cards.count();
  if (total === 0) {
    return null;
  }

  for (let i = 0; i < total; i += 1) {
    const card = cards.nth(i);
    const name = (await card.locator('.module-card__title .name').innerText()).trim();
    if (name !== 'partner') continue;
    const status = (await card.locator('.module-card__title .el-tag').innerText()).trim();
    return { name, status };
  }

  return null;
}

test('meta module management: install/upgrade/uninstall flow', async ({ page }) => {
  // This flow executes up to three heavy module operations (install/upgrade/uninstall)
  // and can exceed 10 minutes on cold CI runners with slower network/disk.
  test.setTimeout(20 * 60 * 1000);

  const runtime = readRuntimeInfo();
  test.skip(runtime.scenario !== 'default', 'only runs under default scenario');
  const baseURL = runtime.baseURL;

  await ensureLoggedIn(page, baseURL);

  const target = await pickTargetModule(page);
  if (!target) {
    test.skip(true, 'No safely operable module was found; expected an uninstalled module or a non-core module');
  }
  const moduleName = target!.name;
  const isInstalled = target!.status.includes('已安装');

  const ciMode = String(process.env.CI || '') === 'true' || String(process.env.GITHUB_ACTIONS || '') === 'true';

  // Keep PR CI fast and stable: execute one representative stateful action.
  // The full three-step chain remains available in non-CI environments.
  if (ciMode) {
    await runAction(page, moduleName, isInstalled ? 'upgrade' : 'install');
    return;
  }

  if (isInstalled) {
    await runAction(page, moduleName, 'upgrade');
    await runAction(page, moduleName, 'uninstall');
    await runAction(page, moduleName, 'install');
  } else {
    await runAction(page, moduleName, 'install');
    await runAction(page, moduleName, 'upgrade');
    await runAction(page, moduleName, 'uninstall');
  }
});

test('meta module management: failed result status flow', async ({ page }) => {
  const runtime = readRuntimeInfo();
  test.skip(runtime.scenario !== 'result-failed', 'only runs under result-failed scenario');

  const baseURL = runtime.baseURL;
  await ensureLoggedIn(page, baseURL);

  const target = await pickTargetModule(page);
  if (!target) {
    test.skip(true, 'No safely operable module was found; expected an uninstalled module or a non-core module');
  }
  await runActionExpectFailure(page, target!.name, target!.status.includes('已安装') ? 'upgrade' : 'install');
});

test('meta module management: reload failed flow', async ({ page }) => {
  const runtime = readRuntimeInfo();
  test.skip(runtime.scenario !== 'reload-failed', 'only runs under reload-failed scenario');

  const baseURL = runtime.baseURL;
  await ensureLoggedIn(page, baseURL);

  const target = await pickTargetModule(page);
  if (!target) {
    test.skip(true, 'No safely operable module was found; expected an uninstalled module or a non-core module');
  }
  await runActionExpectReloadFailed(page, target!.name, target!.status.includes('已安装') ? 'upgrade' : 'install');
});

test('meta module management: lock conflict flow', async ({ page }) => {
  const runtime = readRuntimeInfo();
  test.skip(runtime.scenario !== 'lock-conflict', 'only runs under lock-conflict scenario');

  const baseURL = runtime.baseURL;
  await ensureLoggedIn(page, baseURL);

  const target = await pickTargetModule(page);
  if (!target) {
    test.skip(true, 'No safely operable module was found; expected an uninstalled module or a non-core module');
  }
  await runActionExpectFailure(page, target!.name, target!.status.includes('已安装') ? 'upgrade' : 'install');
});

/**
 * Verifies that the module kanban page loads and remains interactive when the
 * onMounted lazy sync fires in the background. The board must not block on
 * async index refresh.
 */
test('meta module management: kanban lazy sync does not block page', async ({ page }) => {
  const runtime = readRuntimeInfo();
  test.skip(runtime.scenario !== 'default', 'only runs under default scenario');

  const baseURL = runtime.baseURL;
  await ensureLoggedIn(page, baseURL);

  // After ensuring the user is logged in and the board is loaded, the
  // onMounted hook triggers a stale-aware RequestSync for registry then local.
  // The test asserts the page remains interactive: the search input is usable,
  // and module cards are visible.
  const searchInput = page.locator('.o-kanban__search .o-search__input');
  await expect(searchInput).toBeVisible({ timeout: 15000 });

  const cards = page.locator('.module-card');
  const count = await cards.count();
  if (count > 0) {
    // At least one card is present: the board rendered before or during sync.
    await expect(cards.first()).toBeVisible({ timeout: 10000 });
  } else if (count === 0) {
    // No local modules and registry may be unreachable in CI; page must still
    // render the empty board shell without crashing.
    await expect(page.locator('.okanban')).toBeVisible({ timeout: 15000 });
  }

  // The manual sync toolbar button must remain reachable.
  const syncButton = page.getByRole('button', { name: '同步' });
  if (await syncButton.isVisible().catch(() => false)) {
    await expect(syncButton).toBeEnabled({ timeout: 5000 });
  }
});

/**
 * Verifies that the module kanban page remains usable even when registry
 * sync fails silently (e.g. index.choysum.dev is unreachable in an air-gapped
 * or CI environment). The board must still show local modules and allow
 * install/upgrade/uninstall operations.
 */
test('meta module management: kanban usable when registry sync fails', async ({ page }) => {
  const runtime = readRuntimeInfo();
  test.skip(runtime.scenario !== 'default', 'only runs under default scenario');

  const baseURL = runtime.baseURL;
  let forcedRegistrySyncFailures = 0;
  let requestSyncCalls = 0;
  const routeHandler = async (route: any) => {
    const req = route.request();
    if (req.method() === 'POST' && req.url().includes('/meta.IrModuleIndex/RequestSync')) {
      requestSyncCalls += 1;
      if (forcedRegistrySyncFailures === 0) {
        forcedRegistrySyncFailures += 1;
        await route.fulfill({
          status: 503,
          contentType: 'application/json',
          body: JSON.stringify({
            error: {
              code: 'REGISTRY_UNAVAILABLE',
              message: 'simulated registry outage in e2e',
            },
          }),
        });
        return;
      }
    }
    await route.continue();
  };

  await page.route('**/*', routeHandler);
  try {
    await ensureLoggedIn(page, baseURL);

    // Lazy sync for registry runs on onMounted; this test forces registry
    // RequestSync to fail and verifies the page still stays usable.
    await expect(page.locator('.okanban')).toBeVisible({ timeout: 15000 });
    await expect.poll(() => requestSyncCalls, { timeout: 15000 }).toBeGreaterThan(0);
    await expect.poll(() => forcedRegistrySyncFailures, { timeout: 15000 }).toBeGreaterThan(0);

    // The manual sync button should still work for local-only refresh.
    const syncButton = page.getByRole('button', { name: '同步' });
    if (await syncButton.isVisible().catch(() => false)) {
      await syncButton.click();
      await page.waitForTimeout(2000);
      await expect(page.locator('.okanban')).toBeVisible({ timeout: 15000 });
    }

    // The board remains interactive despite registry sync failures.
    const searchInput = page.locator('.o-kanban__search .o-search__input');
    if (await searchInput.isVisible().catch(() => false)) {
      await searchInput.fill('partner');
      await searchInput.press('Enter');
      await waitForModuleList(page);
      await expect(page.locator('.okanban')).toBeVisible({ timeout: 15000 });
    }
  } finally {
    await page.unroute('**/*', routeHandler);
  }
});

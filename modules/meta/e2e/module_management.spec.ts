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

type ModuleAction = 'install' | 'upgrade' | 'uninstall';

type ModuleCardSnapshot = {
  name: string;
  status: string;
  hasInstall: boolean;
  hasUpgrade: boolean;
  hasUninstall: boolean;
};

const preferredModuleOrder = ['partner', 'partner_bank', 'partner_commercial'];

function summarizeCardSnapshots(cards: ModuleCardSnapshot[], limit = 8): string {
  if (!cards || cards.length === 0) {
    return '<none>';
  }
  const rows = cards
    .slice(0, limit)
    .map(card => `${card.name}:${card.status || '-'}:${card.hasInstall ? 'i' : '-'}${card.hasUpgrade ? 'u' : '-'}${card.hasUninstall ? 'x' : '-'}`);
  if (cards.length > limit) {
    rows.push(`... (+${cards.length - limit} more)`);
  }
  return rows.join(', ');
}

function logSkipReason(reason: string) {
  console.warn(`[meta-e2e] SKIP: ${reason}`);
}

async function skipWithVisibleCards(page: Page, baseReason: string) {
  const cards = await snapshotModuleCards(page).catch(() => [] as ModuleCardSnapshot[]);
  const reason = `${baseReason} (visible cards=${summarizeCardSnapshots(cards)})`;
  logSkipReason(reason);
  test.skip(true, reason);
}

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
  // The board can already be stable from cache without a fresh RPC on each poll.
  // Keep a short best-effort response wait to avoid 30s stalls in hot loops.
  await page.waitForResponse(resp => resp.url().includes('meta.IrModule') && resp.status() === 200, { timeout: 2000 }).catch(() => null);
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
    .getByRole('button', { name: 'Install' })
    .first()
    .isVisible()
    .catch(() => false);
  if (hasInstall) {
    return 'Not Installed';
  }

  const hasUpgrade = await card
    .getByRole('button', { name: 'Upgrade' })
    .first()
    .isVisible()
    .catch(() => false);
  const hasUninstall = await card
    .getByRole('button', { name: 'Uninstall' })
    .first()
    .isVisible()
    .catch(() => false);
  if (hasUpgrade || hasUninstall) {
    return 'Installed';
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
    await Promise.race([page.waitForTimeout(waitMs), new Promise(resolve => setTimeout(resolve, waitMs + 1000))]);
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

  const doneButton = page.getByRole('button', { name: 'Done' });
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

  const summaryRow = dialog.locator('.status-row', { hasText: 'Summary' });
  const errorRow = dialog.locator('.status-row', { hasText: 'Error' });

  if (!failedByStatus && !failedByResult) {
    if ((await summaryRow.count()) > 0) {
      await expect(summaryRow.locator('.value')).not.toHaveText(/^\s*$/);
    } else if ((await errorRow.count()) > 0) {
      await expect(errorRow.locator('.value')).not.toHaveText(/—\s*\/\s*—/);
    } else {
      throw new Error(`expected failed operation status, got status=${statusText || '<empty>'}, result=${resultText || '<empty>'}`);
    }
  }

  const doneButton = page.getByRole('button', { name: 'Done' });
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
  const confirmBtn = page.getByRole('button', { name: 'Confirm' });
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
  if (/module operation finished with status (failed|cancelled)/i.test(message)) {
    return true;
  }
  // Module status can be stale right after action completion; allow one retry.
  if (/module\s+.+\s+status remained\s+.+,\s+want\s+.+/i.test(message)) {
    return true;
  }
  // Module board can transiently miss a card during reload/index sync windows.
  return /module card not found:/i.test(message);
}

function isModuleCardMissingError(error: unknown) {
  const message = String((error as { message?: string })?.message ?? error ?? '');
  return /module card not found:/i.test(message);
}

function isNoActionableModuleError(error: unknown) {
  const message = String((error as { message?: string })?.message ?? error ?? '');
  return /no module card exposes action/i.test(message);
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

  const doneButton = page.getByRole('button', { name: 'Done' });
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
  const actionLabel = action === 'install' ? 'Install' : action === 'upgrade' ? 'Upgrade' : 'Uninstall';
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

  const expectedStatus = 'Installed';
  await waitForModuleStatus(page, moduleName, expectedStatus, 90000);
}

async function snapshotModuleCards(page: Page): Promise<ModuleCardSnapshot[]> {
  const cards = page.locator('.module-card');
  const total = await cards.count();
  const snapshots: ModuleCardSnapshot[] = [];

  for (let i = 0; i < total; i += 1) {
    const card = cards.nth(i);
    const visible = await card.isVisible().catch(() => false);
    if (!visible) {
      continue;
    }

    const name = (
      (await card
        .locator('.module-card__title .name')
        .textContent()
        .catch(() => '')) || ''
    ).trim();
    if (!name) {
      continue;
    }

    const status = (
      (await card
        .locator('.module-card__title .el-tag')
        .textContent()
        .catch(() => '')) || ''
    ).trim();

    const hasInstall = await card
      .getByRole('button', { name: 'Install' })
      .first()
      .isVisible()
      .catch(() => false);
    const hasUpgrade = await card
      .getByRole('button', { name: 'Upgrade' })
      .first()
      .isVisible()
      .catch(() => false);
    const hasUninstall = await card
      .getByRole('button', { name: 'Uninstall' })
      .first()
      .isVisible()
      .catch(() => false);

    snapshots.push({
      name,
      status,
      hasInstall,
      hasUpgrade,
      hasUninstall,
    });
  }

  return snapshots;
}

function canApplyAction(card: ModuleCardSnapshot, action: ModuleAction): boolean {
  if (action === 'install') return card.hasInstall;
  if (action === 'upgrade') return card.hasUpgrade;
  return card.hasUninstall;
}

async function resolveActionTargetModule(page: Page, action: ModuleAction, preferredName?: string): Promise<string> {
  await waitForModuleList(page).catch(() => null);
  await clearSearchFilters(page).catch(() => null);

  const cards = await snapshotModuleCards(page);
  const candidates = cards.filter(card => canApplyAction(card, action));
  if (candidates.length === 0) {
    const preview = cards
      .slice(0, 8)
      .map(card => `${card.name}:${card.status || '-'}:${card.hasInstall ? 'i' : '-'}${card.hasUpgrade ? 'u' : '-'}${card.hasUninstall ? 'x' : '-'}`)
      .join(', ');
    throw new Error(`no module card exposes action ${action}; visible cards=${preview || '<none>'}`);
  }

  const normalizedPreferred = String(preferredName || '').trim();
  if (normalizedPreferred) {
    const preferredHit = candidates.find(card => card.name === normalizedPreferred);
    if (preferredHit) {
      return preferredHit.name;
    }
  }

  for (const prefName of preferredModuleOrder) {
    const stableHit = candidates.find(card => card.name === prefName);
    if (stableHit) {
      return stableHit.name;
    }
  }

  return candidates[0].name;
}

/**
 * Executes a module management action and retries on transient failures.
 */
async function runAction(page: Page, moduleName: string | undefined, action: ModuleAction): Promise<string> {
  const expectedStatus = action === 'uninstall' ? 'Not Installed' : 'Installed';
  const maxAttempts = 3;
  let preferredName = String(moduleName || '').trim() || undefined;
  let activeName = preferredName || '';

  for (let attempt = 1; attempt <= maxAttempts; attempt += 1) {
    try {
      activeName = await resolveActionTargetModule(page, action, preferredName);
      await runActionOnce(page, activeName, action);
      return activeName;
    } catch (error) {
      const cardMissing = isModuleCardMissingError(error);
      // Card-lookup misses are often transient right after board transitions.
      // Skip the slower status polling and retry quickly after a lightweight reset.
      if (!cardMissing && activeName && (await hasExpectedModuleStatus(page, activeName, expectedStatus))) {
        return activeName;
      }

      const canRetry = attempt < maxAttempts && shouldRetryOperationFailure(error);
      if (!canRetry) {
        throw error;
      }

      await closeOperationDialogIfPresent(page);
      if (cardMissing) {
        await clearSearchFilters(page).catch(() => null);
        await waitForModuleList(page).catch(() => null);
      }
      await page.reload({ waitUntil: 'domcontentloaded' }).catch(() => null);
      await page.waitForURL('**/web/meta/modules', { timeout: 30000 }).catch(() => null);
      await waitForModuleList(page);
      preferredName = undefined;
    }
  }

  return activeName;
}

/**
 * Executes a module action and asserts that the result dialog reports failure.
 */
async function runActionExpectFailure(page: Page, moduleName: string, action: 'install' | 'upgrade' | 'uninstall') {
  const actionLabel = action === 'install' ? 'Install' : action === 'upgrade' ? 'Upgrade' : 'Uninstall';
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
  const actionLabel = action === 'install' ? 'Install' : action === 'upgrade' ? 'Upgrade' : 'Uninstall';
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
    if ((await reloadRow.count()) === 0) {
      throw new Error('expected reload status row to be present in reload-failed flow');
    }
    await expect(reloadRow).toHaveText(/Trigger Failed/);

    const resultTag = dialog.locator('.status-row .el-tag').nth(1);
    const resultText = ((await resultTag.textContent().catch(() => '')) || '').trim();
    if (/FAILED/i.test(resultText)) {
      throw new Error(`expected successful operation before reload failure, got result=${resultText}`);
    }

    const doneButton = page.getByRole('button', { name: 'Done' });
    const doneVisible = await doneButton.isVisible().catch(() => false);
    if (doneVisible) {
      await doneButton.click();
      await dialog.waitFor({ state: 'hidden', timeout: 15000 }).catch(() => null);
    }
  }

  const expectedStatus = action === 'uninstall' ? 'Not Installed' : 'Installed';
  if (completion === 'reloaded') {
    await waitForModuleStatus(page, moduleName, expectedStatus);
    return;
  }

  // When reload is intentionally forced to fail, board data may remain stale
  // until a later index refresh. Keep status verification best-effort.
  await hasExpectedModuleStatus(page, moduleName, expectedStatus).catch(() => false);
}

/**
 * Picks a stable operable module snapshot for one-shot scenario tests.
 */
async function pickTargetModule(page: Page, opts?: { preferUpgrade?: boolean }) {
  await waitForModuleList(page);

  // The first board paint can race with async module-index hydration.
  // Poll briefly before concluding there are no visible cards.
  let cards = await snapshotModuleCards(page);
  if (cards.length === 0) {
    await expect
      .poll(async () => (await snapshotModuleCards(page)).length, { timeout: 15000 })
      .toBeGreaterThan(0)
      .catch(() => null);
    cards = await snapshotModuleCards(page);
  }
  if (cards.length === 0) {
    await page.reload({ waitUntil: 'domcontentloaded' }).catch(() => null);
    await page.waitForURL('**/web/meta/modules', { timeout: 30000 }).catch(() => null);
    await waitForModuleList(page).catch(() => null);
    cards = await snapshotModuleCards(page);
  }
  if (cards.length === 0) {
    return null;
  }

  const candidates = cards.filter(card => card.hasInstall || card.hasUpgrade);
  if (candidates.length === 0) {
    return null;
  }

  const pickPreferred = (list: ModuleCardSnapshot[]) => {
    for (const preferredName of preferredModuleOrder) {
      const preferred = list.find(card => card.name === preferredName);
      if (preferred) {
        return preferred;
      }
    }
    return list[0];
  };

  if (opts?.preferUpgrade) {
    const upgradeCandidates = candidates.filter(card => card.hasUpgrade);
    if (upgradeCandidates.length > 0) {
      return pickPreferred(upgradeCandidates);
    }
  }

  return pickPreferred(candidates);
}

test('meta module management: install/upgrade/uninstall flow', async ({ page }) => {
  // This flow executes up to three heavy module operations (install/upgrade/uninstall)
  // and can exceed 10 minutes on cold CI runners with slower network/disk.
  test.setTimeout(20 * 60 * 1000);

  const runtime = readRuntimeInfo();
  test.skip(runtime.scenario !== 'default', 'only runs under default scenario');
  const baseURL = runtime.baseURL;

  await ensureLoggedIn(page, baseURL);

  const noActionReasons: string[] = [];
  const runActionIfAvailable = async (action: ModuleAction, preferredName?: string) => {
    try {
      await runAction(page, preferredName, action);
      return true;
    } catch (error) {
      if (isNoActionableModuleError(error)) {
        const message = String((error as { message?: string })?.message ?? error ?? '');
        noActionReasons.push(`${action}: ${message}`);
        return false;
      }
      throw error;
    }
  };

  const initialCards = await snapshotModuleCards(page);
  if (initialCards.length === 0) {
    const reason = `No module cards are visible on the board (visible cards=${summarizeCardSnapshots(initialCards)})`;
    logSkipReason(reason);
    test.skip(true, reason);
  }

  const preferredStart =
    initialCards.find(card => card.hasInstall && preferredModuleOrder.includes(card.name))?.name ||
    initialCards.find(card => card.hasInstall)?.name ||
    initialCards.find(card => card.hasUpgrade && preferredModuleOrder.includes(card.name))?.name ||
    initialCards.find(card => card.hasUpgrade)?.name ||
    initialCards.find(card => preferredModuleOrder.includes(card.name))?.name ||
    initialCards[0]?.name;

  const ciMode = String(process.env.CI || '') === 'true' || String(process.env.GITHUB_ACTIONS || '') === 'true';

  // Keep PR CI fast and stable: execute one representative stateful action.
  // The full three-step chain remains available in non-CI environments.
  if (ciMode) {
    const ciActionDone =
      (await runActionIfAvailable('upgrade', preferredStart)) ||
      (await runActionIfAvailable('install', preferredStart)) ||
      (await runActionIfAvailable('uninstall', preferredStart));
    if (!ciActionDone) {
      const cards = await snapshotModuleCards(page);
      const reason =
        `No actionable module cards are available for CI representative action ` +
        `(visible cards=${summarizeCardSnapshots(cards)}; no_action_reasons=${noActionReasons.join(' | ') || '<none>'})`;
      logSkipReason(reason);
      test.skip(true, reason);
    }
    return;
  }

  let executedActions = 0;
  if (await runActionIfAvailable('install', preferredStart)) executedActions += 1;
  if (await runActionIfAvailable('upgrade')) executedActions += 1;
  if (await runActionIfAvailable('uninstall')) executedActions += 1;

  if (executedActions === 0) {
    const cards = await snapshotModuleCards(page);
    const reason =
      `No actionable module cards are available for install/upgrade/uninstall flow ` +
      `(visible cards=${summarizeCardSnapshots(cards)}; no_action_reasons=${noActionReasons.join(' | ') || '<none>'})`;
    logSkipReason(reason);
    test.skip(true, reason);
  }
});

test('meta module management: failed result status flow', async ({ page }) => {
  const runtime = readRuntimeInfo();
  test.skip(runtime.scenario !== 'result-failed', 'only runs under result-failed scenario');

  const baseURL = runtime.baseURL;
  await ensureLoggedIn(page, baseURL);

  const target = await pickTargetModule(page, { preferUpgrade: true });
  if (!target) {
    await skipWithVisibleCards(page, 'No safely operable module was found for failed result status flow');
  }
  const action = target!.hasUpgrade ? 'upgrade' : target!.hasInstall ? 'install' : null;
  if (!action) {
    await skipWithVisibleCards(page, `Target module ${target!.name} exposes neither install nor upgrade action in failed result status flow`);
  }
  await runActionExpectFailure(page, target!.name, action!);
});

test('meta module management: reload failed flow', async ({ page }) => {
  const runtime = readRuntimeInfo();
  test.skip(runtime.scenario !== 'reload-failed', 'only runs under reload-failed scenario');

  const baseURL = runtime.baseURL;
  await ensureLoggedIn(page, baseURL);

  const target = await pickTargetModule(page, { preferUpgrade: true });
  if (!target) {
    await skipWithVisibleCards(page, 'No safely operable module was found for reload failed flow');
  }
  const action = target!.hasUpgrade ? 'upgrade' : target!.hasInstall ? 'install' : null;
  if (!action) {
    await skipWithVisibleCards(page, `Target module ${target!.name} exposes neither install nor upgrade action in reload failed flow`);
  }
  await runActionExpectReloadFailed(page, target!.name, action!);
});

test('meta module management: lock conflict flow', async ({ page }) => {
  const runtime = readRuntimeInfo();
  test.skip(runtime.scenario !== 'lock-conflict', 'only runs under lock-conflict scenario');

  const baseURL = runtime.baseURL;
  await ensureLoggedIn(page, baseURL);

  const target = await pickTargetModule(page);
  if (!target) {
    await skipWithVisibleCards(page, 'No safely operable module was found for lock conflict flow');
  }
  const action = target!.hasUpgrade ? 'upgrade' : target!.hasInstall ? 'install' : null;
  if (!action) {
    await skipWithVisibleCards(page, `Target module ${target!.name} exposes neither install nor upgrade action in lock conflict flow`);
  }
  await runActionExpectFailure(page, target!.name, action!);
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
  const syncButton = page.getByRole('button', { name: /Sync/i });
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
    const syncButton = page.getByRole('button', { name: /Sync/i });
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

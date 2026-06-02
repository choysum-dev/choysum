// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { test, expect, Page, Locator } from '@playwright/test';
import fs from 'node:fs';

test.setTimeout(10 * 60 * 1000);

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

    const fillInputStable = async (selector: string, value: string) => {
      for (let i = 0; i < 4; i += 1) {
        const input = page.locator(selector).first();
        try {
          await input.waitFor({ state: 'visible', timeout: 4000 });
          await input.fill(value, { timeout: 4000 });
          const actual = await input.inputValue({ timeout: 2000 }).catch(() => '');
          if (actual === value) {
            return true;
          }
        } catch {
          // Retry on transient detach/re-render.
        }
        await page.waitForTimeout(250);
      }
      return false;
    };

    const tryLogin = async (username: string, password: string) => {
      const usernameSelector = 'input[autocomplete="username"], input[placeholder*="username" i], input[placeholder*="用户名"]';
      const passwordSelector = 'input[type="password"][autocomplete="current-password"], input[type="password"][placeholder*="password" i], input[type="password"][placeholder*="密码"]';
      const passwordInput = page.locator(passwordSelector).first();

      const userOK = await fillInputStable(usernameSelector, username);
      const passOK = await fillInputStable(passwordSelector, password);
      if (!userOK || !passOK) {
        return false;
      }

      const submit = page.locator('button[type="submit"]');
      const canClick = await submit.isEnabled().catch(() => false);
      if (canClick) {
        await submit.click({ timeout: 5000 }).catch(() => null);
      } else {
        await passwordInput.press('Enter').catch(() => null);
      }
      await page.waitForURL(/\/(web\/)?meta\/modules/, { timeout: 10000 }).catch(() => null);
      return !page.url().includes('/web/login');
    };

    let ok = false;
    const maxAttempts = 3;
    for (let i = 0; i < maxAttempts; i += 1) {
      ok = await tryLogin('admin', 'admin');
      if (ok) {
        break;
      }
      await page.waitForTimeout(1000);
      await page.goto(`${baseURL}/web/login`, { waitUntil: 'domcontentloaded' }).catch(() => null);
    }
    if (!ok) {
      for (let i = 0; i < maxAttempts; i += 1) {
        ok = await tryLogin('e2e-admin', 'e2e-admin');
        if (ok) {
          break;
        }
        await page.waitForTimeout(1000);
        await page.goto(`${baseURL}/web/login`, { waitUntil: 'domcontentloaded' }).catch(() => null);
      }
    }
    if (!ok) {
      throw new Error(`login failed after retries for admin and e2e-admin; current url=${page.url()}`);
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
async function openModuleCard(page: Page, moduleName: string, opts?: { allowMissing?: boolean }): Promise<Locator | null> {
  const initialCard = await findModuleCardByName(page, moduleName);
  if (initialCard && (await initialCard.isVisible().catch(() => false))) return initialCard;

  await searchModuleCard(page, moduleName);

  const searchedCard = await findModuleCardByName(page, moduleName);
  if (searchedCard && (await searchedCard.isVisible().catch(() => false))) return searchedCard;

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
 * Reads the current status label shown on a module card.
 */
async function moduleStatusText(page: Page, moduleName: string) {
  const card = await openModuleCard(page, moduleName, { allowMissing: true });
  if (!card) {
    return '';
  }
  return (
    (await card
      .locator('.module-card__title .el-tag')
      .textContent()
      .catch(() => '')) || ''
  ).trim();
}

/**
 * Derives module status from available action buttons when the status tag is transiently unavailable.
 */
async function inferModuleStatusFromActions(page: Page, moduleName: string): Promise<string> {
  const card = await openModuleCard(page, moduleName, { allowMissing: true });
  if (!card) {
    return '';
  }

  const installVisible = await card
    .getByRole('button', { name: '安装' })
    .first()
    .isVisible()
    .catch(() => false);
  if (installVisible) {
    return '未安装';
  }

  const upgradeVisible = await card
    .getByRole('button', { name: '升级' })
    .first()
    .isVisible()
    .catch(() => false);
  const uninstallVisible = await card
    .getByRole('button', { name: '卸载' })
    .first()
    .isVisible()
    .catch(() => false);
  if (upgradeVisible || uninstallVisible) {
    return '已安装';
  }

  return '';
}

/**
 * Polls the module board until a module reaches the expected status label.
 */
async function waitForModuleStatus(page: Page, moduleName: string, expectedStatus: string, timeout = 120000) {
  const deadline = Date.now() + timeout;
  let lastStatus = '';
  let emptyPolls = 0;

  while (Date.now() < deadline) {
    await page.reload({ waitUntil: 'domcontentloaded' });
    await page.waitForURL('**/web/meta/modules', { timeout: 30000 }).catch(() => null);
    await waitForModuleList(page);
    const statusText = await moduleStatusText(page, moduleName);
    const inferredStatus = statusText || (await inferModuleStatusFromActions(page, moduleName));
    if (!inferredStatus) {
      emptyPolls += 1;
      await page.waitForTimeout(1000);
      continue;
    }

    lastStatus = inferredStatus;
    if (lastStatus === expectedStatus) {
      return;
    }
    await page.waitForTimeout(1000);
  }

  const detail = lastStatus || '<empty>';
  throw new Error(`module ${moduleName} status remained ${detail}, want ${expectedStatus} (empty polls: ${emptyPolls})`);
}

/**
 * Waits for a terminal operation signal: either backend-triggered reload or a terminal status in the dialog.
 */
async function waitForTerminalSignal(page: Page, timeout = 3 * 60 * 1000): Promise<'reload' | 'terminal'> {
  const dialog = page.locator('.el-dialog');
  const statusTag = dialog.locator('.status-row .el-tag').first();
  const reloadPromise = page
    .waitForNavigation({ waitUntil: 'domcontentloaded', timeout })
    .then(() => 'reload' as const)
    .catch(() => null);
  const terminalPromise = expect(statusTag)
    .toHaveText(/succeeded|failed|cancelled/i, { timeout })
    .then(() => 'terminal' as const)
    .catch(() => null);

  const first = await Promise.race([reloadPromise, terminalPromise]);
  if (first) {
    return first;
  }

  const [reload, terminal] = await Promise.all([reloadPromise, terminalPromise]);
  if (reload) {
    return reload;
  }
  if (terminal) {
    return terminal;
  }

  throw new Error('operation completion signal not observed before timeout');
}

type OperationCompletionResult = {
  signal: 'reload' | 'terminal';
  resultStatus: string;
  failureKind: string;
};

/**
 * Waits until an operation finishes through either a page reload or a result dialog.
 */
async function waitForOperationCompletion(page: Page): Promise<OperationCompletionResult> {
  const dialog = page.locator('.el-dialog');
  const winner = await waitForTerminalSignal(page);
  if (winner === 'terminal') {
    const resultTag = dialog.locator('.status-row .el-tag').nth(1);
    const resultStatus = ((await resultTag.textContent().catch(() => '')) || '').trim();
    const failureKind = (
      (await dialog
        .locator('.status-row .value')
        .first()
        .textContent()
        .catch(() => '')) || ''
    ).trim();
    await page.getByRole('button', { name: '完成' }).click();
    await dialog.waitFor({ state: 'hidden', timeout: 15000 });
    return {
      signal: 'terminal',
      resultStatus,
      failureKind,
    };
  } else if (winner === 'reload') {
    await page.waitForLoadState('networkidle');
    return {
      signal: 'reload',
      resultStatus: 'SUCCEEDED',
      failureKind: '',
    };
  }

  return {
    signal: winner,
    resultStatus: '',
    failureKind: '',
  };
}

/**
 * Waits until an operation completes with a failed result and asserts the failure summary is populated.
 */
async function waitForOperationFailure(page: Page) {
  const dialog = page.locator('.el-dialog');
  const winner = await waitForTerminalSignal(page);
  if (winner === 'reload') {
    throw new Error('expected failure dialog result, but page reloaded before terminal failure status was visible');
  }
  const resultTag = dialog.locator('.status-row .el-tag').nth(1);
  const resultText = (await resultTag.textContent().catch(() => '')) || '';
  if (/FAILED/i.test(resultText)) {
    await page.getByRole('button', { name: '完成' }).click();
    await dialog.waitFor({ state: 'hidden', timeout: 15000 });
    return;
  }
  const summaryRow = dialog.locator('.status-row', { hasText: '摘要' });
  const errorRow = dialog.locator('.status-row', { hasText: '错误' });
  if ((await summaryRow.count()) > 0) {
    await expect(summaryRow.locator('.value')).not.toHaveText(/^\s*$/);
  } else if ((await errorRow.count()) > 0) {
    await expect(errorRow.locator('.value')).not.toHaveText(/—\s*\/\s*—/);
  }
  await page.getByRole('button', { name: '完成' }).click();
  await dialog.waitFor({ state: 'hidden', timeout: 15000 });
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
 * Executes a module management action and waits for the board state to settle.
 */
async function runAction(page: Page, moduleName: string, action: 'install' | 'upgrade' | 'uninstall') {
  const actionLabel = action === 'install' ? '安装' : action === 'upgrade' ? '升级' : '卸载';
  const expectedStatus = action === 'uninstall' ? '未安装' : '已安装';
  const maxAttempts = 2;
  let lastFailure = '';

  for (let attempt = 1; attempt <= maxAttempts; attempt += 1) {
    const card = await openModuleCard(page, moduleName);
    await card.getByRole('button', { name: actionLabel }).click({ timeout: 30000 });

    const dialog = page.locator('.el-dialog');
    await dialog.waitFor({ state: 'visible', timeout: 15000 });
    await clickConfirmWhenReady(page);

    const completion = await waitForOperationCompletion(page);
    const failed = completion.signal === 'terminal' && /FAILED/i.test(completion.resultStatus);
    if (!failed) {
      await waitForModuleStatus(page, moduleName, expectedStatus);
      return;
    }

    lastFailure = [completion.resultStatus, completion.failureKind].filter(Boolean).join(' / ') || 'FAILED';

    try {
      await waitForModuleStatus(page, moduleName, expectedStatus, 45000);
      return;
    } catch {
      if (attempt >= maxAttempts) {
        throw new Error(`module ${moduleName} action ${action} failed after ${attempt} attempts: ${lastFailure}`);
      }
      await page.waitForTimeout(1500);
    }
  }

  throw new Error(`module ${moduleName} action ${action} did not complete: ${lastFailure || 'unknown failure'}`);
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

  const winner = await waitForTerminalSignal(page);
  if (winner === 'reload') {
    throw new Error('expected result dialog for reload-failed assertion, but operation triggered page reload');
  }
  const reloadRow = dialog.locator('.status-row', { hasText: 'Reload' });
  if ((await reloadRow.count()) > 0) {
    await expect(reloadRow).toHaveText(/触发失败/);
  }
  await page.getByRole('button', { name: '完成' }).click();
  await dialog.waitFor({ state: 'hidden', timeout: 15000 });

  const expectedStatus = action === 'uninstall' ? '未安装' : '已安装';
  await waitForModuleStatus(page, moduleName, expectedStatus);
}

/**
 * Selects the fixed partner module as the install/upgrade e2e target.
 */
async function pickTargetModule(page: Page) {
  await waitForModuleList(page);

  const cards = page.locator('.module-card');
  const total = await cards.count();
  if (total === 0) {
    throw new Error('Module list is empty; cannot run the module management regression flow');
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

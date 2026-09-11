// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { test, expect, page, runtime } from '@choysum/e2e';
import { loginAsE2EAdmin } from '../../auth/e2e/utils/login.ts';

/**
 * Partner list Import entry: title-row IO menu opens the import dialog.
 */
test('partner import: list page exposes import entry', async () => {
  const baseURL = runtime.baseURL;

  await loginAsE2EAdmin(page, baseURL);

  await page.goto(`${baseURL}/web/partner/partners`);
  const trigger = page.getByTestId('page-io-menu-trigger');
  await expect(trigger).toBeVisible({ timeout: 30_000 });
  await trigger.click();
  const importItem = page.getByTestId('page-io-menu-import');
  await expect(importItem).toBeVisible();
  await importItem.click();
  await expect(page.getByRole('dialog')).toBeVisible();
});

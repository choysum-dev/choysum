// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  beginNavigationLoading,
  getLoadingRegistryDebugState,
  resetLoadingRegistryForTests,
  runWithLoading,
} from '@/core/web/client/loadingRegistry';
import { settleNavChromeIfNoSurface } from './useSettleNavChromeOnActivate';

test('settleNavChromeIfNoSurface drops an idle navigation ticket', () => {
  resetLoadingRegistryForTests();
  beginNavigationLoading();
  expect(getLoadingRegistryDebugState().navTicketActive).toBe(true);
  settleNavChromeIfNoSurface();
  expect(getLoadingRegistryDebugState().navTicketActive).toBe(false);
  resetLoadingRegistryForTests();
});

test('settleNavChromeIfNoSurface keeps the ticket while a surface is busy', async () => {
  resetLoadingRegistryForTests();
  beginNavigationLoading();
  await runWithLoading({ layer: 'surface', target: 'keep-alive-land', mode: 'block' }, async () => {
    settleNavChromeIfNoSurface();
    expect(getLoadingRegistryDebugState().navTicketActive).toBe(true);
  });
  resetLoadingRegistryForTests();
});

// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// density backfill from main before merge

import { exposeBrowserI18nOnWindow } from './browser_bridge';

test('browser_bridge smoke: exports exposeBrowserI18nOnWindow', () => {
  expect(typeof exposeBrowserI18nOnWindow).toBe('function');
});

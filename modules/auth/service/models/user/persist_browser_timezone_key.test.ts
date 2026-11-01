// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { PERSIST_BROWSER_TIMEZONE_KEY } from './persist_browser_timezone_key';

test('PERSIST_BROWSER_TIMEZONE_KEY is the table-local auth AppSetting key', () => {
  expect(PERSIST_BROWSER_TIMEZONE_KEY).toBe('persist_browser_timezone');
});

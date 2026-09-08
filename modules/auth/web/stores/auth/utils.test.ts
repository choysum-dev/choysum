// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { hashPasswordClient } from './utils';

test('hashPasswordClient: returns client hash when window is present', async () => {
  const got = await hashPasswordClient('plain-secret', 'admin');
  // QuickJS FE host installs minimal DOM (window), so VueUse isClient is true.
  expect(got.indexOf('$CH$') === 0).toBe(true);
  expect(got.length).toBeGreaterThan(4);
});

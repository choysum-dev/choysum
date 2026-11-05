// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { hashPasswordClient } from './utils';

test('hashPasswordClient: returns client hash when window is present', async () => {
  const got = await hashPasswordClient('plain-secret', 'admin');
  // QuickJS FE host installs minimal DOM (window), so VueUse isClient is true.
  expect(got.indexOf('$CH$') === 0).toBe(true);
  expect(got.length).toBeGreaterThan(4);
});

// Same vectors as internal/cli/initialsetup ClientHash: install-driven setup and
// browser Login must produce identical hashes.
test('hashPasswordClient: lowercase hex SHA-256 of "<password>:<username>" in UTF-8', async () => {
  expect(await hashPasswordClient('secret', 'admin')).toBe(
    '$CH$3a0f774e9e3410187269e9d493fe067af71fb547aedfa8a8d798cd29f340393b'
  );
  expect(await hashPasswordClient('密码', '管理员')).toBe(
    '$CH$80158961a09b8473f752dbfb39352e9891e6ccc08f3295f5840578c6f1760bd3'
  );
});

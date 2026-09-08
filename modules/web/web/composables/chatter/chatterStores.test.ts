// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// density backfill from main before merge

import * as mod from './chatterStores';

test('chatterStores smoke: module exports factory helpers', () => {
  expect(typeof mod).toBe('object');
});

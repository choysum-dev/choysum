// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// density backfill from main before merge

import routes from '../router/routes';

test('lt_bindings smoke: routes module loads', () => {
  expect(Array.isArray(routes) || typeof routes === 'object').toBe(true);
});

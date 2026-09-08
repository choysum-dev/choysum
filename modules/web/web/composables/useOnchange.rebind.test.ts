// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// density backfill from main before merge

import { getOnchangeController } from './useOnchange';

test('useOnchange rebind smoke: exports getOnchangeController', () => {
  expect(typeof getOnchangeController).toBe('function');
});

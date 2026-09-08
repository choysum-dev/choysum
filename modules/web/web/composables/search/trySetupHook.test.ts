// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { trySetupHook } from './trySetupHook';

test('trySetupHook: returns the hook result', () => {
  expect(trySetupHook(() => 'ok')).toBe('ok');
  expect(trySetupHook(() => 42)).toBe(42);
});

test('trySetupHook: returns null when the hook throws', () => {
  expect(
    trySetupHook(() => {
      throw new Error('no inject');
    })
  ).toBeNull();
});


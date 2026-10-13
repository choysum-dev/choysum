// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { runShellMenuToggle } from './runShellMenuToggle';

test('runShellMenuToggle prefers layoutStore when present', () => {
  const calls: string[] = [];
  runShellMenuToggle(
    { toggleSidebar: () => calls.push('layout') },
    () => calls.push('kit'),
  );
  expect(calls).toEqual(['layout']);
});

test('runShellMenuToggle falls back to kit toggle when layoutStore is missing', () => {
  const calls: string[] = [];
  runShellMenuToggle(null, () => calls.push('kit'));
  runShellMenuToggle(undefined, () => calls.push('kit2'));
  expect(calls).toEqual(['kit', 'kit2']);
});

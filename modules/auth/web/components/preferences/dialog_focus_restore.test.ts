// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { restoreDialogFocus } from './dialog_focus_restore';

test('restoreDialogFocus: focuses an attached element', () => {
  let focused = false;
  const el = { focus: () => { focused = true; } };
  expect(restoreDialogFocus(el, { contains: (_node: Node | null) => true })).toBe(true);
  expect(focused).toBe(true);
});

test('restoreDialogFocus: skips detached or missing triggers', () => {
  let focused = false;
  const el = { focus: () => { focused = true; } };
  expect(restoreDialogFocus(el, { contains: (_node: Node | null) => false })).toBe(false);
  expect(restoreDialogFocus(null, { contains: (_node: Node | null) => true })).toBe(false);
  expect(focused).toBe(false);
});

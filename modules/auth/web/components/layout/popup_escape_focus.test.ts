// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { dismissPopupOnEscape } from './popup_escape_focus';

test('dismissPopupOnEscape: no-op when closed', () => {
  let closed = false;
  let focused = false;
  expect(
    dismissPopupOnEscape(
      false,
      () => {
        closed = true;
      },
      { focus: () => { focused = true; } },
    ),
  ).toBe(false);
  expect(closed).toBe(false);
  expect(focused).toBe(false);
});

test('dismissPopupOnEscape: closes and focuses trigger', () => {
  let closed = false;
  let focused = false;
  expect(
    dismissPopupOnEscape(
      true,
      () => {
        closed = true;
      },
      { focus: () => { focused = true; } },
    ),
  ).toBe(true);
  expect(closed).toBe(true);
  expect(focused).toBe(true);
});

test('dismissPopupOnEscape: tolerates a missing trigger', () => {
  let closed = false;
  expect(dismissPopupOnEscape(true, () => { closed = true; }, null)).toBe(true);
  expect(closed).toBe(true);
});

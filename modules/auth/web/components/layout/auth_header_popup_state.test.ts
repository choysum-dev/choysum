// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { shouldResetAuthHeaderPopups } from './auth_header_popup_state';

test('shouldResetAuthHeaderPopups: only on authenticated → logged-out transition', () => {
  expect(shouldResetAuthHeaderPopups(true, false)).toBe(true);
  expect(shouldResetAuthHeaderPopups(false, false)).toBe(false);
  expect(shouldResetAuthHeaderPopups(false, true)).toBe(false);
  expect(shouldResetAuthHeaderPopups(true, true)).toBe(false);
});

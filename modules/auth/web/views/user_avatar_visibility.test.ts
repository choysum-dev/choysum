// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { isUserAvatarFieldVisible } from './user_avatar_visibility';

test('isUserAvatarFieldVisible: requires caps.document', () => {
  expect(isUserAvatarFieldVisible(null)).toBe(false);
  expect(isUserAvatarFieldVisible(undefined)).toBe(false);
  expect(isUserAvatarFieldVisible({ document: false })).toBe(false);
  expect(isUserAvatarFieldVisible({ document: true })).toBe(true);
});

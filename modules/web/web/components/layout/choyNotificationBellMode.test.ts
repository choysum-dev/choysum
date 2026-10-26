// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { isChoyNotificationInboxMode } from './choyNotificationBellMode';

test('isChoyNotificationInboxMode: omitted count uses inbox engine', () => {
  expect(isChoyNotificationInboxMode(undefined)).toBe(true);
});

test('isChoyNotificationInboxMode: explicit count uses chrome badge', () => {
  expect(isChoyNotificationInboxMode(0)).toBe(false);
  expect(isChoyNotificationInboxMode(3)).toBe(false);
});

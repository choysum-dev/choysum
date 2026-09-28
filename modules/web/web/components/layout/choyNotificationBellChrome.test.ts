// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  choyNotificationAriaLabel,
  choyNotificationBadgeText,
  choyNotificationUnreadCount,
} from './choyNotificationBellChrome';

test('choyNotificationUnreadCount: clamps non-positive and non-finite to 0', () => {
  expect(choyNotificationUnreadCount(undefined)).toBe(0);
  expect(choyNotificationUnreadCount(0)).toBe(0);
  expect(choyNotificationUnreadCount(-2)).toBe(0);
  expect(choyNotificationUnreadCount(Number.NaN)).toBe(0);
});

test('choyNotificationUnreadCount: ceils positive counts', () => {
  expect(choyNotificationUnreadCount(1)).toBe(1);
  expect(choyNotificationUnreadCount(2.2)).toBe(3);
});

test('choyNotificationBadgeText: empty, numeric, and 99+', () => {
  expect(choyNotificationBadgeText(0)).toBe('');
  expect(choyNotificationBadgeText(3)).toBe('3');
  expect(choyNotificationBadgeText(99)).toBe('99');
  expect(choyNotificationBadgeText(100)).toBe('99+');
  // Defensive: raw negatives / fractions are normalized before formatting.
  expect(choyNotificationBadgeText(-3)).toBe('');
  expect(choyNotificationBadgeText(2.2)).toBe('3');
});

test('choyNotificationAriaLabel: default and custom labels', () => {
  expect(choyNotificationAriaLabel(undefined, 0)).toBe('Notifications');
  expect(choyNotificationAriaLabel('Alerts', 0)).toBe('Alerts');
  expect(choyNotificationAriaLabel('Alerts', 4)).toBe('Alerts (4 unread)');
  expect(choyNotificationAriaLabel('  ', 2)).toBe('Notifications (2 unread)');
});

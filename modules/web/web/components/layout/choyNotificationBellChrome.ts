// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/** Normalize a chrome badge count into a display unread count. */
export function choyNotificationUnreadCount(count: unknown): number {
  const n = Number(count);
  return Number.isFinite(n) && n > 0 ? Math.max(1, Math.ceil(n)) : 0;
}

/** Badge label for unread count (empty when zero; caps at 99+). */
export function choyNotificationBadgeText(unreadCount: number): string {
  const n = choyNotificationUnreadCount(unreadCount);
  if (n > 99) return '99+';
  return n ? String(n) : '';
}

/** Accessible label for the chrome notification button. */
export function choyNotificationAriaLabel(label: string | undefined, unreadCount: number): string {
  const text = String(label || 'Notifications').trim() || 'Notifications';
  const count = choyNotificationUnreadCount(unreadCount);
  return count ? `${text} (${count} unread)` : text;
}

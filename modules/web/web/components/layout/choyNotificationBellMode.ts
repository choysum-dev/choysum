// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Inbox engine (host ONotificationBell) when `count` is omitted.
 * Chrome badge mode when `count` is set (including 0), e.g. Gallery demos.
 */
export function isChoyNotificationInboxMode(count: number | undefined): boolean {
  return count === undefined;
}

// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Inbox engine when `count` is omitted; chrome badge when `count` is passed.
 * Chrome badge mode when `count` is set (including 0), e.g. Gallery demos.
 */
export function isChoyNotificationInboxMode(count: number | undefined): boolean {
  return count === undefined;
}

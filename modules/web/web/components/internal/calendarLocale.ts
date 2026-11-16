// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * BCP47 tag for calendar widgets (Intl / @internationalized/date).
 */
export function calendarLocaleFromBcp47(bcp47: unknown): string {
  return String(bcp47 || '').trim() || 'en-US';
}
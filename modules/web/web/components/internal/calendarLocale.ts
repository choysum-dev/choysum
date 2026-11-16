// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { useI18nStore } from '@/web/web/stores/i18nStore';

/**
 * BCP47 tag for calendar widgets (Intl / @internationalized/date).
 */
export function calendarLocaleFromBcp47(bcp47: unknown): string {
  return String(bcp47 || '').trim() || 'en-US';
}

/** Calendar locale from the active i18n store, or en-US when Pinia is unavailable. */
export function resolveCalendarLocale(): string {
  try {
    return calendarLocaleFromBcp47(useI18nStore().bcp47Locale);
  } catch {
    return 'en-US';
  }
}
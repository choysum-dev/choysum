// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { useI18nStore } from '@/web/web/stores/i18nStore';

/** Reka CalendarRoot weekStartsOn: 0=Sunday … 6=Saturday. */
export type WeekStartsOn = 0 | 1 | 2 | 3 | 4 | 5 | 6;

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

/** Clamp Language.FirstDayOfWeek (or similar) to Reka's 0–6 weekStartsOn. */
export function normalizeWeekStartsOn(value: unknown, fallback: WeekStartsOn = 0): WeekStartsOn {
  // Number(null) and Number('') are 0 — treat those as missing so fallback wins.
  if (value === null || value === undefined || value === '') {
    return fallback;
  }
  const n = Number(value);
  if (Number.isInteger(n) && n >= 0 && n <= 6) {
    return n as WeekStartsOn;
  }
  return fallback;
}

/** Week start for CalendarRoot from the active Language overlay. */
export function resolveCalendarWeekStartsOn(): WeekStartsOn {
  try {
    return normalizeWeekStartsOn(useI18nStore().getFirstDayOfWeek(), 0);
  } catch {
    return 0;
  }
}

function trimOverride(override?: string | null): string {
  return String(override ?? '').trim();
}

/** Dayjs display pattern for dates: prop override, else Language.DateFormat. */
export function resolveActiveDateDisplayFormat(override?: string | null): string {
  const o = trimOverride(override);
  if (o) return o;
  try {
    return (
      String(useI18nStore().activeLanguage.dateTimeFormat?.shortDate || '').trim() || 'YYYY-MM-DD'
    );
  } catch {
    return 'YYYY-MM-DD';
  }
}

/** Dayjs display pattern for times: prop override, else Language.TimeFormat. */
export function resolveActiveTimeDisplayFormat(override?: string | null): string {
  const o = trimOverride(override);
  if (o) return o;
  try {
    return (
      String(useI18nStore().activeLanguage.dateTimeFormat?.shortTime || '').trim() || 'HH:mm:ss'
    );
  } catch {
    return 'HH:mm:ss';
  }
}

/** Dayjs display pattern for datetimes: prop override, else date + time formats. */
export function resolveActiveDateTimeDisplayFormat(override?: string | null): string {
  const o = trimOverride(override);
  if (o) return o;
  try {
    const dt = useI18nStore().activeLanguage.dateTimeFormat;
    const datePart = String(dt?.shortDate || '').trim() || 'YYYY-MM-DD';
    const timePart = String(dt?.shortTime || '').trim() || 'HH:mm:ss';
    return `${datePart} ${timePart}`;
  } catch {
    return 'YYYY-MM-DD HH:mm:ss';
  }
}

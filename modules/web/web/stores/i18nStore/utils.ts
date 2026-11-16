// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { usePreferredLanguages, isClient } from '@vueuse/core';
import dayjs from 'dayjs';
import { DateTimeFormatType } from './types';
import { bcp47FromCode } from './bcp47';
import { formatCurrencyFromConfig, formatNumberFromConfig } from './languageFormat';
import { formatUtcInTimeZone, getUserTimeZone, parseUtc } from '@/web/web/utils/datetime';

/**
 * Normalize a browser language tag to POSIX for matching against Language.Code.
 */
function toPosixCandidate(tag: string): string {
  return String(tag || '')
    .trim()
    .replace(/-/g, '_');
}

/**
 * Detect the best matching active Language.Code from navigator preferences.
 */
export function detectBestLang(allowedCodes?: readonly string[]): string {
  if (!isClient) {
    return 'en_US';
  }

  const catalog =
    allowedCodes?.length && allowedCodes.some(c => String(c || '').trim())
      ? allowedCodes.map(c => String(c).trim()).filter(Boolean)
      : ['en_US'];
  const preferredLanguages = usePreferredLanguages();

  for (const pref of preferredLanguages.value || []) {
    const posix = toPosixCandidate(pref);
    if (catalog.includes(posix)) {
      return posix;
    }
    const mainLang = posix.split('_')[0];
    const matched = catalog.find(code => code === mainLang || code.startsWith(`${mainLang}_`));
    if (matched) {
      return matched;
    }
  }

  return catalog.includes('en_US') ? 'en_US' : catalog[0] || 'en_US';
}

/**
 * True when a persisted value looks like a legacy short UI key (en / zh-CN), not POSIX.
 */
export function looksLikeLegacyUiKey(code: string | null | undefined): boolean {
  const raw = String(code || '').trim();
  if (!raw) {
    return true;
  }
  if (raw.includes('-')) {
    return true;
  }
  if (!raw.includes('_') && raw.length <= 3) {
    return true;
  }
  return false;
}

/**
 * Update the HTML document direction and related CSS variables on the client only.
 */
export function updateDocumentDirection(textDirection: string) {
  if (!isClient) return;

  document.documentElement.setAttribute('dir', textDirection);
  document.documentElement.classList.remove('rtl');

  const isRtl = textDirection === 'rtl';
  const docStyle = document.documentElement.style;

  docStyle.setProperty('--choy-direction', textDirection);
  docStyle.setProperty('--choy-start', isRtl ? 'right' : 'left');
  docStyle.setProperty('--choy-end', isRtl ? 'left' : 'right');
  docStyle.setProperty('--choy-text-align', isRtl ? 'right' : 'left');
  docStyle.setProperty('--choy-transform-direction', isRtl ? '100%' : '-100%');
  docStyle.setProperty('--choy-direction-transform-factor', isRtl ? '1' : '-1');
  docStyle.setProperty('--choy-direction-inset-start', isRtl ? 'right' : 'left');
  docStyle.setProperty('--choy-direction-inset-end', isRtl ? 'left' : 'right');
  docStyle.setProperty('--choy-direction-rotate', isRtl ? '180deg' : '0deg');
  docStyle.setProperty('--choy-direction-flip', isRtl ? '-1' : '1');
  docStyle.setProperty('--choy-border-start-width', '1px');
  docStyle.setProperty('--choy-border-end-width', '1px');
}

type RelativeUnit = Intl.RelativeTimeFormatUnit;

function pickRelativeUnit(diffMs: number): { value: number; unit: RelativeUnit } {
  const abs = Math.abs(diffMs);
  const sec = Math.round(diffMs / 1000);
  const absSec = Math.abs(sec);
  if (absSec < 60) {
    return { value: sec, unit: 'second' };
  }
  const min = Math.round(sec / 60);
  if (Math.abs(min) < 60) {
    return { value: min, unit: 'minute' };
  }
  const hour = Math.round(min / 60);
  if (Math.abs(hour) < 24) {
    return { value: hour, unit: 'hour' };
  }
  const day = Math.round(hour / 24);
  if (Math.abs(day) < 7) {
    return { value: day, unit: 'day' };
  }
  const week = Math.round(day / 7);
  if (Math.abs(week) < 5) {
    return { value: week, unit: 'week' };
  }
  const month = Math.round(day / 30);
  if (Math.abs(month) < 12) {
    return { value: month, unit: 'month' };
  }
  const year = Math.round(day / 365);
  void abs;
  return { value: year, unit: 'year' };
}

/**
 * Format date and time values.
 *
 * - `date` / `time`: no user-timezone conversion (calendar / clock literals).
 * - `datetime`: interpret input as UTC instant and show in user TZ.
 * - `relative`: Intl.RelativeTimeFormat with BCP47.
 */
export function formatDateTime(
  date: Date | string | number,
  config: any,
  options?: {
    type?: DateTimeFormatType;
    format?: string;
    isLong?: boolean;
    timeZone?: string;
    /** BCP47 tag for relative formatting (required for type:'relative'). */
    bcp47?: string;
  }
): string {
  const type = options?.type || 'date';
  const isLong = options?.isLong || false;
  const timeZone = options?.timeZone || getUserTimeZone();

  if (type === 'relative') {
    const iso =
      date instanceof Date
        ? date.toISOString()
        : typeof date === 'number'
          ? new Date(date).toISOString()
          : String(date);
    const targetMs = parseUtc(iso).tz(timeZone).valueOf();
    const nowMs = Date.now();
    const { value, unit } = pickRelativeUnit(targetMs - nowMs);
    const tag = options?.bcp47 || 'en-US';
    try {
      return new Intl.RelativeTimeFormat(tag, { numeric: 'auto' }).format(value, unit);
    } catch {
      return new Intl.RelativeTimeFormat('en-US', { numeric: 'auto' }).format(value, unit);
    }
  }

  let defaultFormat: string;

  switch (type) {
    case 'time':
      defaultFormat = isLong ? config?.longTime || 'HH:mm:ss' : config?.shortTime || 'HH:mm';
      break;
    case 'datetime':
      defaultFormat = `${isLong ? config?.longDate : config?.shortDate || 'YYYY-MM-DD'} ${isLong ? config?.longTime : config?.shortTime || 'HH:mm'}`;
      break;
    case 'date':
    default:
      defaultFormat = isLong ? config?.longDate || 'YYYY-MM-DD' : config?.shortDate || 'YYYY-MM-DD';
  }

  const format = options?.format || defaultFormat;
  if (type === 'datetime') {
    return formatUtcInTimeZone(date, format, timeZone);
  }

  return dayjs(date).format(format);
}

/**
 * Format numbers.
 * Prefer Language-driven separators/grouping when present; otherwise Intl with BCP47.
 */
export function formatNumber(value: number, bcp47: string, config: any, options?: { digits?: number }) {
  if (config && (config.thousandsSeparator != null || config.decimalSeparator != null || config.grouping)) {
    return formatNumberFromConfig(value, config, options);
  }
  try {
    const digits = options?.digits ?? config?.decimalDigits ?? 2;
    const tag = bcp47 || 'en-US';

    return new Intl.NumberFormat(tag, {
      minimumFractionDigits: digits,
      maximumFractionDigits: digits,
    }).format(value);
  } catch (e) {
    return value.toString();
  }
}

/**
 * Format currency values.
 * Prefer Language-driven separators + symbol placement when separators are present.
 */
export function formatCurrency(value: number, bcp47: string, config: any, currencyCode?: string) {
  if (config && (config.thousandsSeparator != null || config.decimalSeparator != null || config.grouping)) {
    return formatCurrencyFromConfig(value, config, currencyCode);
  }
  try {
    const code = currencyCode || config?.code || 'USD';
    const tag = bcp47 || 'en-US';

    return new Intl.NumberFormat(tag, {
      style: 'currency',
      currency: code,
      minimumFractionDigits: config?.decimalDigits ?? 2,
      maximumFractionDigits: config?.decimalDigits ?? 2,
    }).format(value);
  } catch (e) {
    return value.toString();
  }
}

/**
 * Build the date and time format config keyed by POSIX lang.
 */
export function getDateTimeFormats(lang: string, config: any) {
  const formats: Record<string, any> = {};
  const bcp47 = bcp47FromCode(lang);

  formats[lang] = {
    short: {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
    },
    long: {
      year: 'numeric',
      month: 'long',
      day: 'numeric',
      weekday: 'long',
    },
    shortTime: {
      hour: 'numeric',
      minute: 'numeric',
    },
    longTime: {
      hour: 'numeric',
      minute: 'numeric',
      second: 'numeric',
      hour12: bcp47.startsWith('en'),
    },
  };

  return formats;
}

/**
 * Build the number format config keyed by POSIX lang.
 */
export function getNumberFormats(lang: string, numberFormatConfig: any, currencyConfig: any) {
  const formats: Record<string, any> = {};

  formats[lang] = {
    currency: {
      style: 'currency',
      currency: currencyConfig?.code || 'USD',
      notation: 'standard',
    },
    decimal: {
      style: 'decimal',
      minimumFractionDigits: numberFormatConfig?.decimalDigits || 2,
      maximumFractionDigits: numberFormatConfig?.decimalDigits || 2,
    },
    percent: {
      style: 'percent',
      minimumFractionDigits: 0,
      maximumFractionDigits: 2,
    },
  };

  return formats;
}

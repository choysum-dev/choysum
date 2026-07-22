// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { usePreferredLanguages, isClient } from '@vueuse/core';
import dayjs, { type Dayjs } from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';
import { SUPPORTED_LOCALES } from './locales';
import { DateTimeFormatType, SupportedLocale } from './types';
import { formatCurrencyFromConfig, formatNumberFromConfig } from './language_format';

// Enable the relative time plugin.
dayjs.extend(relativeTime);

type DayjsWithRelativeTime = Dayjs & {
  fromNow: (withoutSuffix?: boolean) => string;
};

/**
 * Detect the best matching system UI key.
 * When `allowedKeys` is provided (active Language ∩ catalog), prefer that set.
 */
export function detectBestUiKey(allowedKeys?: readonly string[]): SupportedLocale {
  // Return the default locale on the server.
  if (!isClient) {
    return 'en';
  }

  const catalog = allowedKeys?.length
    ? allowedKeys.filter(code => code in SUPPORTED_LOCALES)
    : Object.keys(SUPPORTED_LOCALES);
  const preferredLanguages = usePreferredLanguages();

  // Try matching the user's preferred language list.
  for (const lang of preferredLanguages.value) {
    // Exact match.
    if (catalog.includes(lang)) {
      return lang as SupportedLocale;
    }

    // Match the base language part, for example 'zh-TW' against 'zh-CN'.
    const mainLang = lang.split('-')[0];
    const matchedLang = catalog.find(supported => supported.startsWith(mainLang));

    if (matchedLang) {
      return matchedLang as SupportedLocale;
    }
  }

  return (catalog.includes('en') ? 'en' : catalog[0] || 'en') as SupportedLocale;
}

/**
 * Update the HTML document direction and related CSS variables on the client only.
 */
export function updateDocumentDirection(textDirection: string) {
  if (!isClient) return;

  // Update the document direction attribute.
  document.documentElement.setAttribute('dir', textDirection);

  // Always remove the RTL class to avoid style conflicts.
  document.documentElement.classList.remove('rtl');
  // Do not add the rtl class anymore; rely on the dir attribute instead.

  // Keep these direction variables because component styles depend on them.
  const isRtl = textDirection === 'rtl';
  const docStyle = document.documentElement.style;

  // Base direction variables.
  docStyle.setProperty('--o-direction', textDirection);
  docStyle.setProperty('--o-start', isRtl ? 'right' : 'left');
  docStyle.setProperty('--o-end', isRtl ? 'left' : 'right');
  docStyle.setProperty('--o-text-align', isRtl ? 'right' : 'left');

  // Transform and positioning variables.
  docStyle.setProperty('--o-transform-direction', isRtl ? '100%' : '-100%');
  docStyle.setProperty('--o-direction-transform-factor', isRtl ? '1' : '-1');

  // Inline direction variables.
  docStyle.setProperty('--o-direction-inset-start', isRtl ? 'right' : 'left');
  docStyle.setProperty('--o-direction-inset-end', isRtl ? 'left' : 'right');

  // Animation variables.
  docStyle.setProperty('--o-direction-rotate', isRtl ? '180deg' : '0deg');
  docStyle.setProperty('--o-direction-flip', isRtl ? '-1' : '1');

  // Border direction variables.
  docStyle.setProperty('--o-border-start-width', '1px');
  docStyle.setProperty('--o-border-end-width', '1px');
}

/**
 * Format date and time values.
 */
export function formatDateTime(
  date: Date | string | number,
  config: any,
  options?: {
    type?: DateTimeFormatType;
    format?: string;
    isLong?: boolean;
  }
): string {
  const type = options?.type || 'date';
  const isLong = options?.isLong || false;

  // Select the most appropriate format for the requested type.
  if (type === 'relative') {
    return (dayjs(date) as DayjsWithRelativeTime).fromNow();
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

  return dayjs(date).format(options?.format || defaultFormat);
}

/**
 * Format numbers.
 * Prefer Language-driven separators/grouping when present; otherwise Intl (catalog incomplete).
 */
export function formatNumber(value: number, locale: string, config: any, options?: { digits?: number }) {
  if (config && (config.thousandsSeparator != null || config.decimalSeparator != null || config.grouping)) {
    return formatNumberFromConfig(value, config, options);
  }
  try {
    const digits = options?.digits ?? config?.decimalDigits ?? 2;

    return new Intl.NumberFormat(locale, {
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
export function formatCurrency(value: number, locale: string, config: any, currencyCode?: string) {
  if (config && (config.thousandsSeparator != null || config.decimalSeparator != null || config.grouping)) {
    return formatCurrencyFromConfig(value, config, currencyCode);
  }
  try {
    const code = currencyCode || config?.code || 'USD';

    return new Intl.NumberFormat(locale, {
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
 * Build the date and time format config.
 */
export function getDateTimeFormats(locale: string, config: any) {
  const formats: Record<string, any> = {};

  formats[locale] = {
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
      hour12: locale.startsWith('en'), // English locales use a 12-hour clock.
    },
  };

  return formats;
}

/**
 * Build the number format config.
 */
export function getNumberFormats(locale: string, numberFormatConfig: any, currencyConfig: any) {
  const formats: Record<string, any> = {};

  formats[locale] = {
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

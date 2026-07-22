// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { SUPPORTED_LOCALES } from './locales';

/**
 * Text direction type.
 */
export type TextDirection = 'ltr' | 'rtl';

/**
 * Date and time format type.
 */
export type DateTimeFormatType = 'date' | 'time' | 'datetime' | 'relative';

/**
 * Locale configuration.
 */
export interface LocaleConfig {
  name: string;
  textDirection: TextDirection;
  dayjsLocaleCode: string;
  elementLocaleCode: string;
  // Dynamic import hooks.
  importElement?: () => Promise<any>;
  importDayjs?: () => Promise<any>;
  importVueI18n?: () => Promise<any>;

  // Number format configuration.
  numberFormat?: {
    thousandsSeparator: string;
    decimalSeparator: string;
    grouping: number[];
    decimalDigits: number;
  };

  // Currency format configuration (catalog fallback; Language may override position/spacing).
  currencyFormat?: {
    symbol: string;
    position: 'before' | 'after';
    code: string;
    decimalDigits: number;
    spacing?: boolean;
  };

  // Date and time format configuration.
  dateTimeFormat?: {
    shortDate: string;
    longDate: string;
    shortTime: string;
    longTime: string;
    firstDayOfWeek: number;
  };
}

export type SupportedLocale = keyof typeof SUPPORTED_LOCALES;

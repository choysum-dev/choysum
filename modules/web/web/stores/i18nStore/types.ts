// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Text direction type.
 */
export type TextDirection = 'ltr' | 'rtl';

/**
 * Date and time format type.
 */
export type DateTimeFormatType = 'date' | 'time' | 'datetime' | 'relative';

/**
 * Cold-start number/date defaults when Language row formats are missing (en_US-shaped).
 */
export const EN_US_FORMAT_DEFAULTS = {
  numberFormat: {
    thousandsSeparator: ',',
    decimalSeparator: '.',
    grouping: [3, 0] as number[],
    decimalDigits: 2,
  },
  currencyFormat: {
    symbol: '$',
    position: 'before' as const,
    code: 'USD',
    decimalDigits: 2,
  },
  dateTimeFormat: {
    shortDate: 'MM/DD/YYYY',
    longDate: 'MMMM D, YYYY',
    shortTime: 'h:mm A',
    longTime: 'h:mm:ss A',
    firstDayOfWeek: 0,
  },
};

/**
 * Active Language row projection from GetActiveLanguages (PascalCase keys preferred).
 */
export type ActiveLanguageRow = {
  Code: string;
  Name?: string;
  Direction?: TextDirection;
  DecimalSeparator?: string;
  ThousandSeparator?: string;
  Grouping?: string | number[];
  DateFormat?: string;
  TimeFormat?: string;
  FirstDayOfWeek?: number;
  CurrencySymbolPosition?: 'before' | 'after';
  CurrencySymbolSpacing?: boolean;
  PersonNameFormat?: string;
};

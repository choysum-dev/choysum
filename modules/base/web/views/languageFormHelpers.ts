// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import dayjs from 'dayjs';
import { formatPersonName } from '@/core/service/utils/person_name';
import {
  formatCurrencyFromConfig,
  formatNumberFromConfig,
  parseGrouping,
} from '@/web/web/stores/i18nStore/languageFormat';

/** Sample instant used for date/time format previews on the Language form. */
export const LANGUAGE_PREVIEW_SAMPLE_DATE = new Date(Date.UTC(2026, 2, 15, 14, 30, 45));

/** Sample amount used for number / currency previews. */
export const LANGUAGE_PREVIEW_SAMPLE_AMOUNT = 1234567.89;

export const GROUPING_SELECTION = ['[3,0]', '[3,2,0]', '[3,3]', '[]'] as const;

export const FIRST_DAY_SELECTION = ['0', '1', '2', '3', '4', '5', '6'] as const;

export const PERSON_NAME_SELECTION = ['{FirstName} {LastName}', '{LastName}{FirstName}'] as const;

/** List columns for LanguageListView (plan §5.1). */
export const LANGUAGE_LIST_COLUMN_PROPS = ['Name', 'Code', 'Direction', 'IsActive', 'DateFormat'] as const;

/** Code is editable only while creating a Language row. */
export function isLanguageCodeReadonly(mode: string | null | undefined): boolean {
  return String(mode || '') !== 'create';
}

export const DATE_FORMAT_PRESETS = [
  'YYYY-MM-DD',
  'DD/MM/YYYY',
  'MM/DD/YYYY',
  'DD.MM.YYYY',
  'DD-MM-YYYY',
  'YYYY年MM月DD日',
] as const;

export const TIME_FORMAT_PRESETS = ['HH:mm:ss', 'hh:mm:ss A', 'HH:mm'] as const;

export const DECIMAL_SEPARATOR_PRESETS = ['.', ','] as const;

export const THOUSAND_SEPARATOR_PRESETS = [',', '.', ' ', "'", ''] as const;

export type LanguageFormPreviewSource = {
  DecimalSeparator?: string | null;
  ThousandSeparator?: string | null;
  Grouping?: string | null;
  DateFormat?: string | null;
  TimeFormat?: string | null;
  CurrencySymbolPosition?: 'before' | 'after' | string | null;
  CurrencySymbolSpacing?: boolean | null;
  PersonNameFormat?: string | null;
};

export function previewNumber(source: LanguageFormPreviewSource, amount = LANGUAGE_PREVIEW_SAMPLE_AMOUNT): string {
  try {
    return formatNumberFromConfig(amount, {
      decimalSeparator: source.DecimalSeparator ?? '.',
      thousandsSeparator: source.ThousandSeparator ?? ',',
      grouping: parseGrouping(source.Grouping ?? '[3,0]'),
      decimalDigits: 2,
    });
  } catch {
    return '';
  }
}

export function previewCurrency(source: LanguageFormPreviewSource, amount = LANGUAGE_PREVIEW_SAMPLE_AMOUNT): string {
  try {
    return formatCurrencyFromConfig(amount, {
      decimalSeparator: source.DecimalSeparator ?? '.',
      thousandsSeparator: source.ThousandSeparator ?? ',',
      grouping: parseGrouping(source.Grouping ?? '[3,0]'),
      decimalDigits: 2,
      symbol: '¥',
      position: source.CurrencySymbolPosition === 'after' ? 'after' : 'before',
      spacing: Boolean(source.CurrencySymbolSpacing),
    });
  } catch {
    return '';
  }
}

export function previewDate(source: LanguageFormPreviewSource, date = LANGUAGE_PREVIEW_SAMPLE_DATE): string {
  const pattern = String(source.DateFormat || '').trim() || 'YYYY-MM-DD';
  try {
    return dayjs(date).format(pattern);
  } catch {
    return '';
  }
}

export function previewTime(source: LanguageFormPreviewSource, date = LANGUAGE_PREVIEW_SAMPLE_DATE): string {
  const pattern = String(source.TimeFormat || '').trim() || 'HH:mm:ss';
  try {
    return dayjs(date).format(pattern);
  } catch {
    return '';
  }
}

export function previewPersonName(source: LanguageFormPreviewSource): string {
  return formatPersonName(
    { FirstName: 'Ada', LastName: 'Lovelace' },
    source.PersonNameFormat
  );
}

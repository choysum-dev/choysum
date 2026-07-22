// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * FE number/date formatting driven by Language format fields (+ optional Preferences.display overrides).
 */

export type LanguageFormatOverlay = {
  DecimalSeparator?: string;
  ThousandSeparator?: string;
  Grouping?: string | number[];
  DateFormat?: string;
  TimeFormat?: string;
  FirstDayOfWeek?: number;
  CurrencySymbolPosition?: 'before' | 'after';
  CurrencySymbolSpacing?: boolean;
};

export type DisplayFormatOverrides = {
  dateFormat?: string;
  timeFormat?: string;
  currency?: string;
};

export function parseGrouping(raw: unknown): number[] {
  if (Array.isArray(raw)) {
    return raw.map(n => Number(n)).filter(n => Number.isFinite(n) && n >= 0);
  }
  const text = String(raw ?? '').trim();
  if (!text) {
    return [3, 0];
  }
  try {
    const parsed = JSON.parse(text.replace(/'/g, '"'));
    if (Array.isArray(parsed)) {
      return parsed.map(n => Number(n)).filter(n => Number.isFinite(n) && n >= 0);
    }
  } catch {
    // fall through
  }
  const nums = text
    .replace(/[\[\]]/g, '')
    .split(',')
    .map(s => Number(String(s).trim()))
    .filter(n => Number.isFinite(n) && n >= 0);
  return nums.length ? nums : [3, 0];
}

export function applyGrouping(integerDigits: string, grouping: number[], thousandsSeparator: string): string {
  const sep = thousandsSeparator ?? '';
  if (!sep || !grouping.length) {
    return integerDigits;
  }
  const digits = String(integerDigits || '').replace(/^\+/, '');
  const negative = digits.startsWith('-');
  let body = negative ? digits.slice(1) : digits;
  if (!body) {
    return integerDigits;
  }

  const parts: string[] = [];
  let index = 0;
  let groupSize = grouping[0] || 3;
  while (body.length > 0) {
    if (index < grouping.length) {
      const next = grouping[index];
      if (next === 0) {
        groupSize = grouping[Math.max(0, index - 1)] || groupSize || 3;
      } else {
        groupSize = next;
        index += 1;
      }
    }
    if (!groupSize || groupSize <= 0) {
      parts.unshift(body);
      break;
    }
    if (body.length <= groupSize) {
      parts.unshift(body);
      break;
    }
    parts.unshift(body.slice(-groupSize));
    body = body.slice(0, -groupSize);
  }
  const joined = parts.join(sep);
  return negative ? `-${joined}` : joined;
}

/**
 * Resolve effective number/date config: Preferences.display sparse overrides > Language > catalog fallback.
 */
export function resolveFormatConfig(
  catalogNumber?: { thousandsSeparator?: string; decimalSeparator?: string; grouping?: number[]; decimalDigits?: number } | null,
  catalogDate?: { shortDate?: string; longDate?: string; shortTime?: string; longTime?: string; firstDayOfWeek?: number } | null,
  language?: LanguageFormatOverlay | null,
  display?: DisplayFormatOverrides | null
) {
  const decimalDigits = catalogNumber?.decimalDigits ?? 2;
  return {
    numberFormat: {
      thousandsSeparator: language?.ThousandSeparator ?? catalogNumber?.thousandsSeparator ?? ',',
      decimalSeparator: language?.DecimalSeparator ?? catalogNumber?.decimalSeparator ?? '.',
      grouping: language?.Grouping != null ? parseGrouping(language.Grouping) : catalogNumber?.grouping ?? [3, 0],
      decimalDigits,
    },
    dateTimeFormat: {
      shortDate: display?.dateFormat || language?.DateFormat || catalogDate?.shortDate || 'YYYY-MM-DD',
      longDate: language?.DateFormat || catalogDate?.longDate || catalogDate?.shortDate || 'YYYY-MM-DD',
      shortTime: display?.timeFormat || language?.TimeFormat || catalogDate?.shortTime || 'HH:mm',
      longTime: language?.TimeFormat || catalogDate?.longTime || catalogDate?.shortTime || 'HH:mm:ss',
      firstDayOfWeek: language?.FirstDayOfWeek ?? catalogDate?.firstDayOfWeek ?? 1,
    },
    currencyCodeOverride: display?.currency,
    currencySymbolPosition: language?.CurrencySymbolPosition,
    currencySymbolSpacing: language?.CurrencySymbolSpacing,
  };
}

/**
 * Format an already-quantized decimal string (e.g. Decimal.toFixed(scale)) with Language separators.
 * Prefer this over Number() for high-scale fields to avoid float noise.
 */
export function formatFixedDecimalString(
  fixed: string,
  config: { thousandsSeparator?: string; decimalSeparator?: string; grouping?: number[] }
): string {
  const text = String(fixed ?? '').trim();
  if (!text) {
    return '';
  }
  const decimalSep = config.decimalSeparator ?? '.';
  const thousandSep = config.thousandsSeparator ?? ',';
  const grouping = config.grouping?.length ? config.grouping : [3, 0];
  const negative = text.startsWith('-');
  const body = negative ? text.slice(1) : text;
  const [intRaw, fracPart] = body.split('.');
  const intPart = applyGrouping(intRaw || '0', grouping, thousandSep);
  const out = fracPart != null ? `${intPart}${decimalSep}${fracPart}` : intPart;
  return negative ? `-${out}` : out;
}

export function formatNumberFromConfig(
  value: number,
  config: { thousandsSeparator?: string; decimalSeparator?: string; grouping?: number[]; decimalDigits?: number },
  options?: { digits?: number }
): string {
  if (!Number.isFinite(value)) {
    return String(value);
  }
  const digits = options?.digits ?? config.decimalDigits ?? 2;
  const sign = value < 0 ? '-' : '';
  const abs = Math.abs(value);
  const fixed = abs.toFixed(digits);
  return formatFixedDecimalString(`${sign}${fixed}`, config);
}

/**
 * Format currency with Language-driven separators and symbol position/spacing.
 */
export function formatCurrencyFromConfig(
  value: number,
  config: {
    thousandsSeparator?: string;
    decimalSeparator?: string;
    grouping?: number[];
    decimalDigits?: number;
    symbol?: string;
    code?: string;
    position?: 'before' | 'after';
    spacing?: boolean;
  },
  currencyCode?: string
): string {
  const digits = config.decimalDigits ?? 2;
  const amount = formatNumberFromConfig(value, config, { digits });
  const symbol = config.symbol || currencyCode || config.code || '';
  if (!symbol) {
    return amount;
  }
  const space = config.spacing ? ' ' : '';
  if (config.position === 'after') {
    return `${amount}${space}${symbol}`;
  }
  return `${symbol}${space}${amount}`;
}

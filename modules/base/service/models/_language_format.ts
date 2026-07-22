// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Pure number/date format helpers driven by Language format fields.
 * Grouping follows Odoo/POSIX style lists such as "[3,0]" or [3, 0].
 */

export type LanguageFormatFields = {
  DecimalSeparator?: string | null;
  ThousandSeparator?: string | null;
  Grouping?: string | number[] | null;
  DateFormat?: string | null;
  TimeFormat?: string | null;
  FirstDayOfWeek?: number | null;
  CurrencySymbolPosition?: 'before' | 'after' | null;
  CurrencySymbolSpacing?: boolean | null;
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

/**
 * Apply thousands grouping from the right.
 * Trailing 0 means "repeat previous group size".
 */
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
        // Repeat previous non-zero size.
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
    if (index >= grouping.length) {
      // Keep repeating last group size (Odoo [3,0] behavior after consuming list).
      // no-op: groupSize already set
    }
  }
  const joined = parts.join(sep);
  return negative ? `-${joined}` : joined;
}

export function formatNumberWithLanguage(
  value: number,
  fields: LanguageFormatFields,
  options?: { digits?: number }
): string {
  if (!Number.isFinite(value)) {
    return String(value);
  }
  const digits = options?.digits ?? 2;
  const decimalSep = fields.DecimalSeparator != null && fields.DecimalSeparator !== '' ? String(fields.DecimalSeparator) : '.';
  const thousandSep = fields.ThousandSeparator != null ? String(fields.ThousandSeparator) : ',';
  const grouping = parseGrouping(fields.Grouping);

  const sign = value < 0 ? '-' : '';
  const abs = Math.abs(value);
  const fixed = abs.toFixed(digits);
  const [intPartRaw, fracPart] = fixed.split('.');
  const intPart = applyGrouping(intPartRaw || '0', grouping, thousandSep);
  if (digits <= 0) {
    return `${sign}${intPart}`;
  }
  return `${sign}${intPart}${decimalSep}${fracPart || ''}`;
}

export function formatDateWithLanguage(value: Date | string | number, fields: LanguageFormatFields, kind: 'date' | 'time' | 'datetime' = 'date'): string {
  const d = value instanceof Date ? value : new Date(value);
  if (Number.isNaN(d.getTime())) {
    return String(value);
  }
  const pad = (n: number) => String(n).padStart(2, '0');
  const yyyy = d.getFullYear();
  const mm = pad(d.getMonth() + 1);
  const dd = pad(d.getDate());
  const HH = pad(d.getHours());
  const mi = pad(d.getMinutes());
  const ss = pad(d.getSeconds());

  const dateFmt = String(fields.DateFormat || 'YYYY-MM-DD');
  const timeFmt = String(fields.TimeFormat || 'HH:mm:ss');

  const render = (fmt: string) =>
    fmt
      .replace(/YYYY/g, String(yyyy))
      .replace(/MM/g, mm)
      .replace(/DD/g, dd)
      .replace(/HH/g, HH)
      .replace(/mm/g, mi)
      .replace(/ss/g, ss);

  if (kind === 'time') {
    return render(timeFmt);
  }
  if (kind === 'datetime') {
    return `${render(dateFmt)} ${render(timeFmt)}`;
  }
  return render(dateFmt);
}

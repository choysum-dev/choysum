// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Shared chrome props and number/monetary helpers for Choy field components.
 */

export type ChoyFieldChromeProps = {
  label?: string;
  help?: string;
  required?: boolean;
  readonly?: boolean;
  disabled?: boolean;
  error?: string;
  name?: string;
  visible?: boolean;
};

/** Option row shared by selection and statusbar fields. */
export type ChoySelectionOption = {
  value: string;
  label: string;
};

export const choyFieldChromeDefaults = {
  label: '',
  help: '',
  required: false,
  readonly: false,
  disabled: false,
  error: '',
  name: '',
  visible: true,
} as const satisfies Required<ChoyFieldChromeProps>;

/** Fields are visible unless `visible` is explicitly false. */
export function resolveChoyFieldVisible(visible?: boolean): boolean {
  return visible !== false;
}

/** Clamps monetary display/commit precision to the range supported by `toFixed`. */
export function resolveChoyMonetaryPrecision(precision?: number): number {
  return precision !== undefined && Number.isFinite(precision) && precision >= 0
    ? Math.min(100, Math.floor(precision))
    : 2;
}

export type ChoyDecimalRound = {
  /** Numeric value (negative zero normalized to `0`). */
  value: number;
  /** Canonical decimal text at the requested precision. */
  text: string;
};

/**
 * Rounds a decimal digit string with half-away-from-zero ties.
 * Operates on decimal digits so values like `"1.005"` at precision 2 become `1.01`.
 */
export function roundChoyDecimal(
  raw: string,
  precision?: number,
): ChoyDecimalRound | null {
  const digits = resolveChoyMonetaryPrecision(precision);
  const text = String(raw ?? '').trim();
  if (!text || !/^-?\d+(\.\d+)?$/.test(text)) {
    return null;
  }
  const negative = text.startsWith('-');
  const body = negative ? text.slice(1) : text;
  const [intRaw, fracRaw = ''] = body.split('.');
  const intDigits = (intRaw || '0').replace(/^0+(?=\d)/, '') || '0';
  const fracPadded = fracRaw.padEnd(digits + 1, '0');
  const keepFrac = digits > 0 ? fracPadded.slice(0, digits) : '';
  const roundDigit = digits === 0 ? (fracRaw.charAt(0) || '0') : fracPadded.charAt(digits);
  let digs = `${intDigits}${keepFrac}`.split('').map((c) => c.charCodeAt(0) - 48);
  if (roundDigit >= '5') {
    let i = digs.length - 1;
    while (i >= 0) {
      digs[i]! += 1;
      if (digs[i]! < 10) {
        break;
      }
      digs[i] = 0;
      i -= 1;
    }
    if (i < 0) {
      digs.unshift(1);
    }
  }
  let outInt: string;
  let outFrac: string;
  if (digits === 0) {
    outInt = digs.join('').replace(/^0+(?=\d)/, '') || '0';
    outFrac = '';
  } else {
    while (digs.length <= digits) {
      digs.unshift(0);
    }
    outFrac = digs.slice(-digits).join('');
    outInt = digs.slice(0, -digits).join('').replace(/^0+(?=\d)/, '') || '0';
  }
  const isZero = outInt === '0' && (digits === 0 || /^0+$/.test(outFrac));
  if (isZero) {
    return {
      value: 0,
      text: digits === 0 ? '0' : `0.${'0'.repeat(digits)}`,
    };
  }
  const signed =
    digits === 0
      ? `${negative ? '-' : ''}${outInt}`
      : `${negative ? '-' : ''}${outInt}.${outFrac}`;
  const value = Number(signed);
  if (!Number.isFinite(value)) {
    return null;
  }
  return { value: Object.is(value, -0) ? 0 : value, text: signed };
}

/**
 * Formats a monetary amount for display. Empty/invalid input yields ''.
 * When `currency` is set it is appended after the number.
 */
export function formatChoyMonetary(
  value: number | string | null | undefined,
  opts?: { precision?: number; currency?: string },
): string {
  if (value === null || value === undefined) {
    return '';
  }
  const precision = resolveChoyMonetaryPrecision(opts?.precision);
  let formatted: string;
  if (typeof value === 'string') {
    const rounded = roundChoyDecimal(value.trim(), precision);
    if (!rounded) {
      return '';
    }
    formatted = rounded.text;
  } else {
    if (!Number.isFinite(value)) {
      return '';
    }
    const decimal = Object.is(value, -0) ? '0' : String(value);
    const rounded = roundChoyDecimal(decimal, precision);
    // Exponential notation (|value| >= 1e21) is not decimal-parseable; keep toFixed there.
    formatted = rounded ? rounded.text : value.toFixed(precision);
  }
  const currency = String(opts?.currency ?? '').trim();
  return currency ? `${formatted} ${currency}` : formatted;
}

/**
 * Parses user input into a number. Blank → null; invalid → null.
 * `integer` rejects fractional values; `float` / `decimal` accept them.
 */
export function parseChoyNumber(
  raw: string,
  mode: 'integer' | 'float' | 'decimal',
): number | null {
  const text = String(raw ?? '').trim();
  if (!text) {
    return null;
  }
  if (mode === 'integer') {
    if (!/^-?\d+$/.test(text)) {
      return null;
    }
    const n = Number(text);
    return Number.isSafeInteger(n) ? n : null;
  }
  if (!/^-?\d+(\.\d+)?$/.test(text)) {
    return null;
  }
  const n = Number(text);
  return Number.isFinite(n) ? n : null;
}

/**
 * Prefer a draft string that `parseChoyNumber` still accepts for `value`.
 * `String(value)` can be exponential (e.g. `1e-22`), which this parser rejects.
 */
export function resolveChoyNumberDraftText(
  value: number,
  preferredRaw: string,
  mode: 'integer' | 'float' | 'decimal',
): string {
  const normalized = String(value);
  if (parseChoyNumber(normalized, mode) === value) {
    return normalized;
  }
  const preferred = String(preferredRaw ?? '').trim();
  return preferred || normalized;
}

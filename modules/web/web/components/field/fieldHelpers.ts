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

/**
 * Option row shared by selection and statusbar fields.
 * `value` must be non-empty: Reka Select rejects `''`, and the selection model
 * uses `null` for unset (empty-string options are filtered out).
 */
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
 * Expands exponential decimal text into plain digits
 * (e.g. `1.25e-3` → `0.00125`, `1.5e1` → `15`).
 * Returns null when `raw` is not a plain coefficient×10^exp form.
 */
export function expandExponentialDecimalText(raw: string): string | null {
  const text = String(raw ?? '').trim();
  // Allow trailing-dot (`12.e0`) and leading-dot (`.5e0`) coefficients.
  const m = /^([+-]?)(?:(\d+)(?:\.(\d*))?|\.(\d+))e([+-]?\d+)$/i.exec(text);
  if (!m) {
    return null;
  }
  const neg = m[1] === '-';
  const intPart = m[2] ?? '';
  const fracPart = m[3] ?? m[4] ?? '';
  const digits = `${intPart}${fracPart}`;
  const exp = Number(m[5]);
  const point = intPart.length + exp;
  // Pathological exponents (e.g. `1e-999999999`, whose Number() is still finite 0)
  // would make `String.repeat` throw RangeError; any finite double needs ≪ 400 pad digits.
  // (`Infinity > 400` also rejects engine overflow from huge exponent literals.)
  if (point <= 0) {
    const pad = -point;
    if (pad > 400) {
      return null;
    }
    return `${neg ? '-' : ''}0.${'0'.repeat(pad)}${digits}`;
  }
  if (point >= digits.length) {
    const pad = point - digits.length;
    if (pad > 400) {
      return null;
    }
    return `${neg ? '-' : ''}${digits}${'0'.repeat(pad)}`;
  }
  return `${neg ? '-' : ''}${digits.slice(0, point)}.${digits.slice(point)}`;
}

/**
 * Expands a finite number's shortest string into plain decimal digits when it
 * uses exponential notation (e.g. `1e-7` → `0.0000001`).
 */
export function expandFiniteNumberToPlainDecimal(n: number): string {
  if (Object.is(n, -0) || n === 0) {
    return '0';
  }
  const s = String(n);
  if (!/e/i.test(s)) {
    return s;
  }
  // Engine `String(n)` exponential forms always match the expander regex.
  return expandExponentialDecimalText(s)!;
}

/**
 * Normalizes decimal / exponential text for exactness comparison.
 * Strips '+', leading int zeros, trailing frac zeros; maps ±0 to '0'.
 * Exponential forms are expanded to plain decimal when possible.
 */
export function canonicalChoyDecimal(raw: string): string | null {
  const trimmed = String(raw ?? '').trim();
  if (!trimmed) {
    return null;
  }
  let text = trimmed;
  if (/e/i.test(text)) {
    const n = Number(text);
    if (!Number.isFinite(n)) {
      return null;
    }
    // Prefer expanding the literal so MID coefficient forms stay exact; fall back
    // to String(n) for engine-specific exponential shapes (and ±0).
    text = expandExponentialDecimalText(text) ?? expandFiniteNumberToPlainDecimal(n);
  }
  if (!/^[+-]?(\d+\.?\d*|\.\d+)$/.test(text)) {
    return null;
  }
  const neg = text.startsWith('-');
  const body = text.replace(/^[+-]/, '');
  const [intRaw, fracRaw = ''] = body.includes('.')
    ? (body.split('.') as [string, string])
    : [body, ''];
  const intPart = (intRaw || '0').replace(/^0+(?=\d)/, '') || '0';
  const fracPart = fracRaw.replace(/0+$/, '');
  if (intPart === '0' && fracPart === '') {
    return '0';
  }
  const core = fracPart ? `${intPart}.${fracPart}` : intPart;
  return neg ? `-${core}` : core;
}

/**
 * Rounds a decimal digit string with half-away-from-zero ties.
 * Operates on decimal digits so values like `"1.005"` at precision 2 become `1.01`.
 */
export function roundChoyDecimal(
  raw: string,
  precision?: number,
): ChoyDecimalRound | null {
  const digits = resolveChoyMonetaryPrecision(precision);
  // Users routinely commit "12." — drop a trailing decimal point before validating.
  const text = String(raw ?? '').trim().replace(/\.$/, '');
  if (!text || !/^[+-]?(\d+(\.\d+)?|\.\d+)$/.test(text)) {
    return null;
  }
  const negative = text.startsWith('-');
  const body = text.replace(/^[+-]/, '');
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
    // `digs` is always int-digits + `digits` frac digits (and may grow on carry).
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
  // Beyond MAX_SAFE_INTEGER the numeric `value` no longer matches the exact `text`.
  if (!Number.isFinite(value) || !Number.isSafeInteger(Math.trunc(value))) {
    return null;
  }
  // Reject text that the double cannot reproduce (covers the [2^51, 2^52) hole).
  if (canonicalChoyDecimal(signed) !== canonicalChoyDecimal(String(value))) {
    return null;
  }
  // Zero (including negative zero) is handled above via `isZero`.
  return { value, text: signed };
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
    const trimmed = value.trim();
    const rounded = roundChoyDecimal(trimmed, precision);
    const numeric = Number(trimmed);
    if (rounded) {
      formatted = rounded.text;
    } else if (
      trimmed &&
      /e/i.test(trimmed) &&
      Number.isFinite(numeric) &&
      /^[+-]?(\d+\.?\d*|\.\d+)(e[+-]?\d+)?$/i.test(trimmed)
    ) {
      // Require a successful literal expansion so forms like `9007199254740993.e0`
      // (expandable) and unexpandable e-text cannot pass via String(numeric)===itself.
      const expanded = expandExponentialDecimalText(trimmed);
      if (
        expanded === null ||
        canonicalChoyDecimal(expanded) !== canonicalChoyDecimal(String(numeric))
      ) {
        return '';
      }
      // Only exponential text needs the numeric fallback: a plain decimal rejected by
      // roundChoyDecimal is unrepresentable, so returning '' avoids a silently
      // re-rounded (lossy) amount.
      formatted =
        roundChoyDecimal(String(numeric), precision)?.text ??
        numeric.toFixed(precision);
    } else {
      return '';
    }
  } else {
    if (!Number.isFinite(value)) {
      return '';
    }
    const decimal = Object.is(value, -0) ? '0' : String(value);
    const rounded = roundChoyDecimal(decimal, precision);
    if (!rounded && !/e/i.test(decimal)) {
      // Non-exponential magnitudes rejected by roundChoyDecimal must not use lossy toFixed.
      return '';
    }
    // Exponential notation (|value| >= 1e21) is not decimal-parseable; keep toFixed there.
    formatted = rounded ? rounded.text : value.toFixed(precision);
  }
  // `toFixed` on a tiny negative magnitude yields "-0.00" / "-0"; normalize it like `-0`.
  if (/^-0(\.0+)?$/.test(formatted)) {
    formatted = formatted.slice(1);
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
  // Users routinely commit "12." — drop a trailing decimal point before validating.
  const text = String(raw ?? '').trim().replace(/\.$/, '');
  if (!text) {
    return null;
  }
  if (mode === 'integer') {
    if (!/^[+-]?\d+$/.test(text)) {
      return null;
    }
    const n = Number(text);
    return Number.isSafeInteger(n) ? n : null;
  }
  if (!/^[+-]?(\d+(\.\d+)?|\.\d+)$/.test(text)) {
    return null;
  }
  const n = Number(text);
  // Match roundChoyDecimal: reject magnitudes that Number cannot represent exactly.
  if (!Number.isFinite(n) || !Number.isSafeInteger(Math.trunc(n))) {
    return null;
  }
  // Reject text that the double cannot reproduce (covers the [2^51, 2^52) hole).
  if (canonicalChoyDecimal(text) !== canonicalChoyDecimal(String(n))) {
    return null;
  }
  return n;
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
  if (preferred && parseChoyNumber(preferred, mode) === value) {
    return preferred;
  }
  return normalized;
}

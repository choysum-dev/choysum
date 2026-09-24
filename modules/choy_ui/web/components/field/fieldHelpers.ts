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
  const raw = typeof value === 'number' ? value : String(value).trim();
  if (raw === '' || (typeof raw === 'string' && Number.isNaN(Number(raw)))) {
    return '';
  }
  const num = typeof raw === 'number' ? raw : Number(raw);
  if (!Number.isFinite(num)) {
    return '';
  }
  const precision = resolveChoyMonetaryPrecision(opts?.precision);
  const formatted = num.toFixed(precision);
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

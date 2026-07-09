// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { fail } from './_normalization_bridge';

/**
 * Normalize a Direction (ltr/rtl) selection value.
 * Returns undefined for undefined, null for null/empty, or the validated value.
 */
export function normalizeDirection(value: unknown): 'ltr' | 'rtl' | null | undefined {
  if (value === undefined) return undefined;
  if (value === null || value === '') return null;
  if (value === 'ltr' || value === 'rtl') return value;
  fail('Direction must be ltr or rtl');
}

/**
 * Normalize a CurrencySymbolPosition (before/after) selection value.
 * Defaults to 'before' for empty/falsy input.
 */
export function normalizeCurrencySymbolPosition(value: unknown): 'before' | 'after' {
  if (value === undefined || value === null || value === '') return 'before';
  if (value === 'before' || value === 'after') return value;
  fail('CurrencySymbolPosition must be before or after');
}

/**
 * Normalize a CurrencySymbolSpacing boolean value.
 * Defaults to false for empty/falsy input.
 */
export function normalizeCurrencySymbolSpacing(value: unknown): boolean {
  if (value === undefined || value === null || value === '') return false;
  return Boolean(value);
}

/**
 * Normalize a RatePolicy.Mode value (exact/latest_before).
 * Defaults to 'latest_before'.
 */
export function normalizeRatePolicyMode(value: unknown): 'exact' | 'latest_before' {
  if (value === undefined || value === null || value === '') return 'latest_before';
  if (value === 'exact' || value === 'latest_before') return value;
  fail('RatePolicy.Mode must be exact or latest_before');
}

/**
 * Normalize a Rounding.Mode value (currency/none).
 * Defaults to 'currency'.
 */
export function normalizeRoundingMode(value: unknown): 'currency' | 'none' {
  if (value === undefined || value === null || value === '') return 'currency';
  if (value === 'currency' || value === 'none') return value;
  fail('Rounding.Mode must be currency or none');
}

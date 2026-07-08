// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ChoysumError, GrpcCode } from '@/core/service/error';
import { NormalizationError } from '@/core/service/utils/normalization';

/**
 * Throw a base-domain InvalidArgument error.
 */
export function fail(message: string): never {
  throw new ChoysumError({ domain: 'base', code: 'InvalidArgument', message }).withGrpcCode(GrpcCode.InvalidArgument);
}

/**
 * Normalize a required code field: trim, optionally uppercase, fail if empty.
 */
export function normalizeCodeRequired(value: any, opts?: { uppercase?: boolean }): string {
  let code = String(value ?? '').trim();
  if (opts?.uppercase !== false) {
    code = code.toUpperCase();
  }
  if (!code) fail('Code is required');
  return code;
}

/**
 * Normalize an optional code field: trim, optionally uppercase.
 * Returns undefined for undefined input, null for null/empty input.
 */
export function normalizeCodeOptional(value: any, opts?: { uppercase?: boolean }): string | null | undefined {
  if (value === undefined) return undefined;
  if (value === null) return null;
  let code = String(value ?? '').trim();
  if (opts?.uppercase !== false) {
    code = code.toUpperCase();
  }
  return code || null;
}

/**
 * Normalize a required name field: trim, fail if empty.
 */
export function normalizeName(value: any): string {
  const name = String(value ?? '').trim();
  if (!name) fail('Name is required');
  return name;
}

/**
 * Map a domain-agnostic normalization failure into base-domain InvalidArgument.
 */
export function mapNormalizationToBase<T>(fn: () => T, mapMessage: (err: NormalizationError) => string): T {
  try {
    return fn();
  } catch (err) {
    if (err instanceof NormalizationError) {
      fail(mapMessage(err));
    }
    throw err;
  }
}

/**
 * Normalize an optional string field: trim, null/undefined → null, empty → null.
 */
export function normalizeOptionalString(value: any): string | null {
  if (value === undefined || value === null) return null;
  const s = String(value).trim();
  return s || null;
}

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

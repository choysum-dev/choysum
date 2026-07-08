// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { Decimal } from '@/core/service';
import { ChoysumError, GrpcCode } from '@/core/service/error';

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
 * Validate and coerce a value to a positive Decimal.
 * Throws via fail() on missing, zero, negative, or unparseable input.
 */
export function toPositiveDecimal(value: any, fieldName: string): Decimal {
  try {
    if (value === undefined || value === null || value === '') throw new Error('required');
    const decimal = value instanceof Decimal ? value : new Decimal((value as any)?.$bigdecimal ?? value);
    if (!decimal.gt(0)) fail(`${fieldName} must be greater than 0`);
    return decimal;
  } catch (err) {
    if (err instanceof ChoysumError) throw err;
    fail(`${fieldName} must be a valid decimal`);
  }
}

/**
 * Validate and normalize a positive decimal value, returning its string representation.
 * Same semantics as toPositiveDecimal but returns a string.
 */
export function normalizePositiveDecimalString(value: any, fieldName: string): string {
  try {
    if (value == null || value === '') throw new Error('required');
    const decimal = value instanceof Decimal ? value : new Decimal((value as any)?.$bigdecimal ?? value);
    if (!decimal.gt(0)) {
      fail(`${fieldName} must be greater than 0`);
    }
    return decimal.toString();
  } catch (err: any) {
    if (err instanceof ChoysumError) throw err;
    fail(`${fieldName} must be a valid decimal`);
  }
}

/**
 * Normalize a required text field with a custom field name in the error message.
 * Trims and fails if empty.
 */
export function normalizeRequiredText(value: unknown, fieldName: string): string {
  const v = String(value ?? '').trim();
  if (!v) fail(`${fieldName} is required`);
  return v;
}

/**
 * Parse a value as a positive integer (>= 1). Throws InvalidArgument on failure.
 */
export function parsePositiveInt(value: unknown, fieldName: string): number {
  const n = Number(value);
  if (!Number.isFinite(n) || Math.floor(n) !== n || n < 1) {
    throw new ChoysumError({ domain: 'base', code: 'InvalidArgument', message: `${fieldName} must be an integer >= 1` }).withGrpcCode(GrpcCode.InvalidArgument);
  }
  return n;
}

/**
 * Parse a value as a BigInt. Throws InvalidArgument on failure.
 */
export function parseBigInt(value: unknown, fieldName: string): bigint {
  try {
    if (typeof value === 'bigint') return value;
    if (typeof value === 'number' && Number.isFinite(value)) return BigInt(Math.trunc(value));
    if (value && typeof value === 'object' && typeof (value as any).$bigint === 'string') return BigInt((value as any).$bigint);
    const text = String(value ?? '').trim();
    if (!text) throw new Error('empty');
    return BigInt(text);
  } catch {
    throw new ChoysumError({ domain: 'base', code: 'InvalidArgument', message: `${fieldName} must be a valid integer` }).withGrpcCode(GrpcCode.InvalidArgument);
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
 * Normalize a DecimalDigits value: must be a non-negative integer.
 */
export function normalizeDecimalDigits(value: any): number {
  if (value === undefined || value === null || value === '') {
    fail('DecimalDigits is required');
  }
  const n = Number(value);
  if (!Number.isFinite(n) || Math.floor(n) !== n || n < 0) {
    fail('DecimalDigits must be a non-negative integer');
  }
  return n;
}

/**
 * Validate a YYYY-MM-DD date string with format and calendar-validity checks.
 */
export function normalizeDateString(value: unknown, fieldName: string): string {
  if (value === undefined || value === null || value === '') fail(`${fieldName} is required`);
  if (value instanceof Date) fail(`${fieldName} must be YYYY-MM-DD`);
  const raw = String(value).trim();
  if (!/^\d{4}-\d{2}-\d{2}$/.test(raw)) {
    fail(`${fieldName} must be YYYY-MM-DD`);
  }
  const date = new Date(`${raw}T00:00:00.000Z`);
  if (Number.isNaN(date.getTime()) || date.toISOString().slice(0, 10) !== raw) {
    fail(`${fieldName} is invalid`);
  }
  return raw;
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

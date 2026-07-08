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

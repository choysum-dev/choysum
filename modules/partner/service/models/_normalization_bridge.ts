// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { raiseDomainError } from '@/core/service/error';
import { NormalizationError, normalizeOptionalString, normalizeRequiredText as normalizeRequiredTextCore } from '@/core/service/utils/normalization';

/**
 * Throw a partner-domain InvalidArgument error.
 */
export function fail(message: string): never {
  raiseDomainError('partner', 'InvalidArgument', message);
}

/**
 * Map a domain-agnostic normalization failure into partner-domain InvalidArgument.
 */
export function mapNormalizationToPartner<T>(fn: () => T, mapMessage: (err: NormalizationError) => string): T {
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
 * Normalize an optional text field with case-coercion support.
 *
 * Preserves the undefined/null distinction:
 * - undefined → undefined
 * - null      → null
 * - empty     → null
 */
export function normalizeOptionalText(value: unknown, opts?: { upper?: boolean; lower?: boolean }): string | null | undefined {
  if (value === undefined) return undefined;
  if (value === null) return null;
  const base = normalizeOptionalString(value);
  if (base === undefined) return null;
  if (opts?.upper) return base.toUpperCase();
  if (opts?.lower) return base.toLowerCase();
  return base;
}

/**
 * Normalize a required text field and throw a partner-domain error with the
 * field name when the value is empty.
 */
export function normalizeRequiredText(value: unknown, fieldName: string): string {
  return mapNormalizationToPartner(
    () => normalizeRequiredTextCore(value),
    () => `${fieldName} is required`
  );
}

/**
 * Normalize a non-negative integer field.
 *
 * Throws a partner-domain error when the value is negative, NaN, Infinity,
 * or has a fractional component.
 */
export function normalizeNonNegativeInt(value: unknown, fieldName: string): number | undefined {
  if (value === undefined) return undefined;
  if (value === null) return 0;
  if (typeof value !== 'number' && typeof value !== 'string') {
    fail(`${fieldName} must be a non-negative integer`);
  }
  const num = Number(value);
  if (!Number.isFinite(num) || num < 0 || Math.floor(num) !== num) {
    fail(`${fieldName} must be a non-negative integer`);
  }
  return num;
}

/**
 * Normalize a display-sequence integer.
 *
 * null/empty input defaults to 10. Allows negative integers.
 * Throws a partner-domain error for non-integer values.
 */
export function normalizeSequenceInt(value: unknown, defaultValue: number = 10): number | undefined {
  if (value === undefined) return undefined;
  if (value === null || (typeof value === 'string' && value.trim() === '')) {
    return defaultValue;
  }
  if (typeof value !== 'number' && typeof value !== 'string') {
    fail('Sequence must be an integer');
  }
  const num = Number(value);
  if (!Number.isFinite(num) || Math.floor(num) !== num) {
    fail('Sequence must be an integer');
  }
  return num;
}

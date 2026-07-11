// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { raiseDomainError } from '@/core/service/error';
import { NormalizationError, normalizeOptionalString, normalizeRequiredText as normalizeRequiredTextCore } from '@/core/service/utils/normalization';

/**
 * Throw a partner-bank-domain InvalidArgument error.
 */
export function fail(message: string): never {
  raiseDomainError('partner_bank', 'InvalidArgument', message);
}

/**
 * Map a domain-agnostic normalization failure into partner-bank-domain InvalidArgument.
 */
export function mapNormalizationToPartnerBank<T>(fn: () => T, mapMessage: (err: NormalizationError) => string): T {
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
 * Normalize an optional text field with optional case coercion.
 *
 * Delegates trimming and case coercion to core normalizeOptionalString;
 * this wrapper only adds the partner_bank-domain null / undefined contract:
 *
 * - undefined → undefined
 * - null      → null
 * - empty     → null
 * - otherwise → trimmed, optionally uppercased
 */
export function normalizeOptionalText(value: unknown, opts?: { upper?: boolean }): string | null | undefined {
  if (value === undefined) return undefined;
  if (value === null) return null;
  const base = normalizeOptionalString(value, opts);
  return base === undefined ? null : base;
}

/**
 * Normalize a required text field, rejecting blank values with a
 * partner-bank-domain InvalidArgument error.
 *
 * Supports an optional `upper` case coercion applied after the
 * required-text check.
 */
export function normalizeRequiredText(value: unknown, fieldName: string, opts?: { upper?: boolean }): string {
  return mapNormalizationToPartnerBank(
    () => {
      const normalized = normalizeRequiredTextCore(value);
      return opts?.upper ? normalized.toUpperCase() : normalized;
    },
    () => `${fieldName} is required`
  );
}

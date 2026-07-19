// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { raiseDomainError } from '@/core/service/error';
import {
  NormalizationError,
  normalizeOptionalString,
  normalizeRefId,
  normalizeRequiredText as normalizeRequiredTextCore,
} from '@/core/service/utils/normalization';
import { toDate as toDateCore } from '@/core/service/utils/datetime';
import { createTranslate } from '@/core/service/i18n';

const { _t } = createTranslate('partner_commercial');

/**
 * Throw a partner-commercial-domain InvalidArgument error.
 */
export function fail(message: string): never {
  raiseDomainError('partner_commercial', 'InvalidArgument', message);
}

/**
 * Map a domain-agnostic normalization failure into partner-commercial-domain InvalidArgument.
 */
export function mapNormalizationToPartnerCommercial<T>(fn: () => T, mapMessage: (err: NormalizationError) => string): T {
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
 * Normalize an optional relation reference, preserving undefined and null.
 *
 * Delegates to core normalizeRefId for the actual normalization; this wrapper
 * only preserves the partner_commercial-domain contract:
 *
 * - undefined → undefined  (field not provided — skip in partial updates)
 * - null      → null       (explicitly cleared)
 * - otherwise → trimmed Id string or null
 */
export function normalizeOptionalRefId(value: unknown): string | null | undefined {
  if (value === undefined) return undefined;
  if (value === null) return null;
  return normalizeRefId(value);
}

/**
 * Normalize an optional text field with case-coercion support.
 *
 * Delegates trimming and case coercion to core normalizeOptionalString;
 * this wrapper only adds the partner_commercial-domain null / undefined contract:
 *
 * - undefined → undefined
 * - null      → null
 * - empty     → null
 */
export function normalizeOptionalText(value: unknown, opts?: { lower?: boolean; upper?: boolean }): string | null | undefined {
  if (value === undefined) return undefined;
  if (value === null) return null;
  const base = normalizeOptionalString(value, opts);
  return base === undefined ? null : base;
}

/**
 * Normalize a required text field, rejecting blank values with a
 * partner-commercial-domain InvalidArgument error.
 *
 * Supports optional case coercion applied after the required-text check.
 */
export function normalizeRequiredText(value: unknown, fieldName: string, opts?: { lower?: boolean; upper?: boolean }): string {
  return mapNormalizationToPartnerCommercial(
    () => {
      let normalized = normalizeRequiredTextCore(value);
      if (opts?.lower) normalized = normalized.toLowerCase();
      if (opts?.upper) normalized = normalized.toUpperCase();
      return normalized;
    },
    () => _t('%s is required', { scope: 'service/models/_normalization_bridge' }, fieldName)
  );
}

/**
 * Parse an optional datetime field, rejecting invalid values with a
 * partner-commercial-domain InvalidArgument error.
 *
 * Delegates parsing to core toDate and only adds the domain error wrapper.
 */
export function toDateOrUndefined(value: unknown, fieldName: string): Date | undefined {
  if (value === undefined || value === null || value === '') return undefined;
  const result = toDateCore(value);
  if (result === undefined) {
    fail(_t('%s must be a valid datetime', { scope: 'service/models/_normalization_bridge' }, fieldName));
  }
  return result;
}

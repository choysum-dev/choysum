// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ChoysumError, GrpcCode } from '@/core/service/error';
import {
  NormalizationError,
  normalizeCodeOptional as normalizeCodeOptionalCore,
  normalizeCodeRequired as normalizeCodeRequiredCore,
  normalizeName as normalizeNameCore,
  normalizeNullableString as normalizeNullableStringCore,
  requireRefId as requireRefIdCore,
} from '@/core/service/utils/normalization';

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
  return mapNormalizationToBase(
    () => normalizeCodeRequiredCore(value, opts),
    () => 'Code is required'
  );
}

/**
 * Normalize an optional code field: trim, optionally uppercase.
 * Returns undefined for undefined input, null for null/empty input.
 */
export function normalizeCodeOptional(value: any, opts?: { uppercase?: boolean }): string | null | undefined {
  return normalizeCodeOptionalCore(value, opts);
}

/**
 * Normalize a required name field: trim, fail if empty.
 */
export function normalizeName(value: any): string {
  return mapNormalizationToBase(
    () => normalizeNameCore(value),
    () => 'Name is required'
  );
}

/**
 * Resolve and require a reference ID, failing with InvalidArgument if empty.
 */
export function requireRefId(value: unknown, fieldName: string): string {
  return mapNormalizationToBase(
    () => requireRefIdCore(value),
    () => `${fieldName} is required`
  );
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
export function normalizeNullableString(value: any): string | null {
  return normalizeNullableStringCore(value);
}

// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { raiseDomainError } from '@/core/service/error';
import { NormalizationError, normalizeOptionalString, normalizeRequiredText as normalizeRequiredTextCore } from '@/core/service/utils/normalization';
import { createTranslate } from '@/core/service/i18n';

const { _t } = createTranslate('partner');

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
 * Delegates trimming and case coercion to core normalizeOptionalString;
 * this wrapper only adds the partner-domain null / undefined contract:
 *
 * - undefined → undefined
 * - null      → null
 * - empty     → null
 */
export function normalizeOptionalText(value: unknown, opts?: { upper?: boolean; lower?: boolean }): string | null | undefined {
  if (value === undefined) return undefined;
  if (value === null) return null;
  const base = normalizeOptionalString(value, opts);
  return base === undefined ? null : base;
}

/**
 * Normalize a required text field and throw a partner-domain error with the
 * field name when the value is empty.
 */
export function normalizeRequiredText(value: unknown, fieldName: string): string {
  return mapNormalizationToPartner(
    () => normalizeRequiredTextCore(value),
    () => _t('%s is required', { scope: 'service/models/_normalization_bridge' }, fieldName)
  );
}

function isLangMap(value: unknown): value is Record<string, unknown> {
  return !!value && typeof value === 'object' && !Array.isArray(value);
}

/**
 * Normalize a required translated field: string or `{ lang: string }` map.
 * Empty maps / all-empty values fail the same as a missing scalar.
 * Per-lang empty strings are allowed (data-i18n D12).
 */
export function normalizeRequiredTranslatedText(
  value: unknown,
  fieldName: string
): string | Record<string, string> {
  if (isLangMap(value)) {
    const out: Record<string, string> = {};
    for (const [lang, raw] of Object.entries(value)) {
      const key = String(lang || '').trim();
      if (!key) continue;
      const normalized = normalizeOptionalText(raw);
      out[key] = normalized === undefined || normalized === null ? '' : normalized;
    }
    if (!Object.values(out).some(v => String(v || '').trim())) {
      fail(_t('%s is required', { scope: 'service/models/_normalization_bridge' }, fieldName));
    }
    return out;
  }
  return normalizeRequiredText(value, fieldName);
}

/**
 * Normalize an optional translated field: string, null, or lang map.
 */
export function normalizeOptionalTranslatedText(
  value: unknown,
  opts?: { upper?: boolean; lower?: boolean }
): string | null | undefined | Record<string, string> {
  if (value === undefined) return undefined;
  if (value === null) return null;
  if (isLangMap(value)) {
    const out: Record<string, string> = {};
    for (const [lang, raw] of Object.entries(value)) {
      const key = String(lang || '').trim();
      if (!key) continue;
      const normalized = normalizeOptionalText(raw, opts);
      if (normalized === undefined) continue;
      out[key] = normalized === null ? '' : normalized;
    }
    return out;
  }
  return normalizeOptionalText(value, opts);
}

/** True when a scalar or lang-map translated value has any non-empty text. */
export function translatedTextHasValue(value: unknown): boolean {
  if (value == null) return false;
  if (typeof value === 'string') return !!value.trim();
  if (isLangMap(value)) {
    return Object.values(value).some(v => typeof v === 'string' && !!v.trim());
  }
  return false;
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
    fail(_t('%s must be a non-negative integer', { scope: 'service/models/_normalization_bridge' }, fieldName));
  }
  const num = Number(value);
  if (!Number.isFinite(num) || num < 0 || Math.floor(num) !== num) {
    fail(_t('%s must be a non-negative integer', { scope: 'service/models/_normalization_bridge' }, fieldName));
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
    fail(_t('Sequence must be an integer', { scope: 'service/models/_normalization_bridge' }));
  }
  const num = Number(value);
  if (!Number.isFinite(num) || Math.floor(num) !== num) {
    fail(_t('Sequence must be an integer', { scope: 'service/models/_normalization_bridge' }));
  }
  return num;
}

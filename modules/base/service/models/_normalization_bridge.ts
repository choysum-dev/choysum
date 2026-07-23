// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { raiseDomainError } from '@/core/service/error';
import { createTranslate } from '@/core/service/i18n';
import {
  NormalizationError,
  normalizeCodeOptional as normalizeCodeOptionalCore,
  normalizeCodeRequired as normalizeCodeRequiredCore,
  normalizeName as normalizeNameCore,
  normalizeNullableString as normalizeNullableStringCore,
  requireRefId as requireRefIdCore,
} from '@/core/service/utils/normalization';

const { _t } = createTranslate('base');

/**
 * Throw a base-domain InvalidArgument error.
 */
export function fail(message: string): never {
  raiseDomainError('base', 'InvalidArgument', message);
}

/**
 * Normalize a required code field: trim, optionally uppercase, fail if empty.
 */
export function normalizeCodeRequired(value: any, opts?: { uppercase?: boolean }): string {
  return mapNormalizationToBase(
    () => normalizeCodeRequiredCore(value, opts),
    () => _t('Code is required', { scope: 'service/models/_normalization_bridge' })
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
    () => _t('Name is required', { scope: 'service/models/_normalization_bridge' })
  );
}

/**
 * Resolve and require a reference ID, failing with InvalidArgument if empty.
 */
export function requireRefId(value: unknown, fieldName: string): string {
  return mapNormalizationToBase(
    () => requireRefIdCore(value),
    () => _t('%s is required', { scope: 'service/models/_normalization_bridge' }, fieldName)
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

function isLangMap(value: unknown): value is Record<string, unknown> {
  return !!value && typeof value === 'object' && !Array.isArray(value);
}

/**
 * Normalize a required translated field: string or `{ lang: string }` map.
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
      const normalized = normalizeNullableString(raw);
      out[key] = normalized == null ? '' : normalized;
    }
    if (!Object.values(out).some(v => String(v || '').trim())) {
      fail(_t('%s is required', { scope: 'service/models/_normalization_bridge' }, fieldName));
    }
    return out;
  }
  return normalizeName(value);
}

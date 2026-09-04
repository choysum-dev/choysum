// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createTranslate } from '@/core/service/i18n';
import { normalizeOptionalString } from '@/core/service/utils/normalization';
import { fail } from './_base_bridge';

const { _t } = createTranslate('base');
function assertEnumMember<T extends string>(
  value: unknown,
  allowed: readonly T[],
  message: string
): T | null | undefined {
  if (value === undefined) return undefined;
  if (value === null || value === '') return null;
  const text = normalizeOptionalString(value);
  if (!text || !(allowed as readonly string[]).includes(text)) {
    fail(message);
  }
  return text as T;
}

/**
 * Validate a Direction (ltr/rtl) selection value.
 * Undefined means omitted; null/empty means cleared; invalid values fail.
 */
export function assertDirection(value: unknown): 'ltr' | 'rtl' | null | undefined {
  return assertEnumMember(value, ['ltr', 'rtl'] as const, _t('Direction must be ltr or rtl', { scope: 'service/models/_option_validators' }));
}

/**
 * Validate a CurrencySymbolPosition (before/after) selection value.
 * Does not wash empty input to a default; model defaults apply when the field is omitted.
 */
export function assertCurrencySymbolPosition(value: unknown): 'before' | 'after' | null | undefined {
  return assertEnumMember(
    value,
    ['before', 'after'] as const,
    _t('CurrencySymbolPosition must be before or after', { scope: 'service/models/_option_validators' })
  );
}

/**
 * Validate a CurrencySymbolSpacing boolean value.
 * Undefined means omitted; null/empty means cleared; non-boolean values fail.
 */
export function assertCurrencySymbolSpacing(value: unknown): boolean | null | undefined {
  if (value === undefined) return undefined;
  if (value === null || value === '') return null;
  if (typeof value === 'boolean') return value;
  fail(_t('CurrencySymbolSpacing must be a boolean', { scope: 'service/models/_option_validators' }));
}

/**
 * Validate a RatePolicy.Mode value (exact/latest_before).
 * Callers apply product defaults when the result is undefined/null.
 */
export function assertRatePolicyMode(value: unknown): 'exact' | 'latest_before' | null | undefined {
  return assertEnumMember(
    value,
    ['exact', 'latest_before'] as const,
    _t('RatePolicy.Mode must be exact or latest_before', { scope: 'service/models/_option_validators' })
  );
}

/**
 * Validate a Rounding.Mode value (currency/none).
 * Callers apply product defaults when the result is undefined/null.
 */
export function assertRoundingMode(value: unknown): 'currency' | 'none' | null | undefined {
  return assertEnumMember(
    value,
    ['currency', 'none'] as const,
    _t('Rounding.Mode must be currency or none', { scope: 'service/models/_option_validators' })
  );
}

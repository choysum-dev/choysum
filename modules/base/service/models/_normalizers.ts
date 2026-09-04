// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Re-export barrel for base domain bridge helpers and option validators.
 *
 * Generic domain bridge → ./_base_bridge
 * Enum/option validators → ./_option_validators
 */

export {
  fail,
  mapNormalizationToBase,
  assertCodeRequired,
  normalizeCodeOptional,
  assertName,
  normalizeNullableString,
  assertRequiredTranslatedText,
  assertRefId,
} from './_base_bridge';

export {
  assertCurrencySymbolPosition,
  assertCurrencySymbolSpacing,
  assertDirection,
  assertRatePolicyMode,
  assertRoundingMode,
} from './_option_validators';

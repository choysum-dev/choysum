// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Backward-compatible re-export barrel.
 *
 * Generic normalization bridge → ./_normalization_bridge
 * Enum/option validators      → ./_option_normalizers
 */

export {
  fail,
  mapNormalizationToBase,
  normalizeCodeRequired,
  normalizeCodeOptional,
  normalizeName,
  normalizeOptionalString,
  requireRefId,
} from './_normalization_bridge';

export {
  normalizeCurrencySymbolPosition,
  normalizeCurrencySymbolSpacing,
  normalizeDirection,
  normalizeRatePolicyMode,
  normalizeRoundingMode,
} from './_option_normalizers';

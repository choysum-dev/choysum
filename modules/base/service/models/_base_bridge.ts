// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createTranslate } from '@/core/service/i18n';
import { createDomainNormalizationBridge } from '@/core/service/utils/domain_normalization_bridge';
import {
  normalizeCodeOptional as normalizeCodeOptionalCore,
  normalizeCodeRequired as normalizeCodeRequiredCore,
  normalizeName as normalizeNameCore,
  normalizeNullableString as normalizeNullableStringCore,
  requireRefId as requireRefIdCore,
} from '@/core/service/utils/normalization';

const { _t } = createTranslate('base');
const bridge = createDomainNormalizationBridge('base', _t);
const scope = 'service/models/_base_bridge';

/** Throw a base-domain InvalidArgument error. */
export const fail = bridge.fail;

/** Map a domain-agnostic normalization failure into base-domain InvalidArgument. */
export const mapNormalizationToBase = bridge.mapNormalizationError;

/** Validate a required translated field: string or `{ lang: string }` map. */
export const assertRequiredTranslatedText = bridge.normalizeRequiredTranslatedText;

/** Normalize an optional code field: trim, optionally uppercase. */
export const normalizeCodeOptional = normalizeCodeOptionalCore;

/** Normalize an optional string field: trim, null/undefined → null, empty → null. */
export const normalizeNullableString = normalizeNullableStringCore;

/**
 * Validate a required code field: trim, optionally uppercase, fail if empty.
 */
export function assertCodeRequired(value: any, opts?: { uppercase?: boolean }): string {
  return mapNormalizationToBase(
    () => normalizeCodeRequiredCore(value, opts),
    () => _t('Code is required', { scope })
  );
}

/**
 * Validate a required name field: trim, fail if empty.
 */
export function assertName(value: any): string {
  return mapNormalizationToBase(
    () => normalizeNameCore(value),
    () => _t('Name is required', { scope })
  );
}

/**
 * Resolve and require a reference ID, failing with InvalidArgument if empty.
 */
export function assertRefId(value: unknown, fieldName: string): string {
  return mapNormalizationToBase(
    () => requireRefIdCore(value),
    () => _t('%s is required', { scope }, fieldName)
  );
}

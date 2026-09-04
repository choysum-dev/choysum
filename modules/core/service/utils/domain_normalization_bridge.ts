// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { raiseDomainError } from '@/core/service/error';
import type { TranslateFn } from '@/core/service/i18n';
import { toDate } from '@/core/service/utils/datetime';
import {
  NormalizationError,
  assertNonNegativeInt as assertNonNegativeIntCore,
  assertRequiredText as assertRequiredTextCore,
  assertRequiredTranslatedText as assertRequiredTranslatedTextCore,
  normalizeOptionalRefId,
  normalizeOptionalText,
  normalizeOptionalTranslatedText,
  normalizeSequenceInt as normalizeSequenceIntCore,
  translatedTextHasValue,
} from '@/core/service/utils/normalization';

const DEFAULT_SCOPE = 'service/models/_normalization_bridge';

export type DomainNormalizationBridgeOptions = {
  /** i18n scope for default required / validation messages. */
  scope?: string;
};

/**
 * Build a module-domain normalization bridge over core pure helpers.
 *
 * Pure logic stays in {@link ./normalization}; this factory only binds
 * `raiseDomainError(domain, 'InvalidArgument', …)` and `_t(...)` messages.
 */
export function createDomainNormalizationBridge(domain: string, _t: TranslateFn, options?: DomainNormalizationBridgeOptions) {
  const scope = options?.scope ?? DEFAULT_SCOPE;

  function fail(message: string): never {
    raiseDomainError(domain, 'InvalidArgument', message);
  }

  function mapNormalizationError<T>(fn: () => T, mapMessage: (err: NormalizationError) => string): T {
    try {
      return fn();
    } catch (err) {
      if (err instanceof NormalizationError) {
        fail(mapMessage(err));
      }
      throw err;
    }
  }

  function assertRequiredText(value: unknown, fieldName: string, opts?: { upper?: boolean; lower?: boolean }): string {
    return mapNormalizationError(
      () => {
        let normalized = assertRequiredTextCore(value);
        if (opts?.lower) normalized = normalized.toLowerCase();
        if (opts?.upper) normalized = normalized.toUpperCase();
        return normalized;
      },
      () => _t('%s is required', { scope }, fieldName)
    );
  }

  function assertRequiredTranslatedText(value: unknown, fieldName: string): string | Record<string, string> {
    return mapNormalizationError(
      () => assertRequiredTranslatedTextCore(value),
      () => _t('%s is required', { scope }, fieldName)
    );
  }

  function assertNonNegativeInt(value: unknown, fieldName: string): number | undefined {
    return mapNormalizationError(
      () => assertNonNegativeIntCore(value),
      () => _t('%s must be a non-negative integer', { scope }, fieldName)
    );
  }

  function normalizeSequenceInt(value: unknown, defaultValue: number = 10): number | undefined {
    return mapNormalizationError(
      () => normalizeSequenceIntCore(value, defaultValue),
      () => _t('Sequence must be an integer', { scope })
    );
  }

  function assertDateOrUndefined(value: unknown, fieldName: string): Date | undefined {
    if (value === undefined || value === null || value === '') return undefined;
    const result = toDate(value);
    if (result === undefined) {
      fail(_t('%s must be a valid datetime', { scope }, fieldName));
    }
    return result;
  }

  return {
    fail,
    mapNormalizationError,
    normalizeOptionalText,
    normalizeOptionalRefId,
    normalizeOptionalTranslatedText,
    translatedTextHasValue,
    assertRequiredText,
    assertRequiredTranslatedText,
    assertNonNegativeInt,
    normalizeSequenceInt,
    assertDateOrUndefined,
  };
}

export type DomainNormalizationBridge = ReturnType<typeof createDomainNormalizationBridge>;

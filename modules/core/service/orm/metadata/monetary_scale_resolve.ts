// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { FieldMetadata } from './field';
import { getCurrencyFieldName } from './decimal_like';
import { normalizeCurrencyRefId, readDecimalDigitsFromCurrencyValue } from './monetary_currency';
import { buildHiddenScaleAlias } from '../repository/hidden_scale_alias';
import { asObjectRecord } from '@/core/utils/object';

export function monetaryCurrencyRequiredError(fieldName: string): Error {
  return new Error(`currency required for monetary field ${fieldName}`);
}

/**
 * Resolve DecimalDigits for a monetary field from the write payload / current row.
 * Prefers a previously stamped hidden scale alias, then an inlined Currency object.
 */
export function resolveMonetaryScaleFromPayload(
  fm: FieldMetadata | undefined,
  input: unknown,
  current?: unknown
): { scale?: number; currencyId?: string; needsBrowse: boolean } {
  if (!fm || fm.type !== 'monetary') return { needsBrowse: false };
  const fieldName = String(fm.name || '').trim();
  const inputRecord = asObjectRecord(input) || {};
  const currentRecord = asObjectRecord(current) || {};

  if (fieldName) {
    const hidden = inputRecord[buildHiddenScaleAlias(fieldName)];
    if (hidden != null && Number.isInteger(Number(hidden))) {
      return { scale: Number(hidden), needsBrowse: false };
    }
  }

  const currencyField = getCurrencyFieldName(fm.column);
  if (!currencyField) {
    throw monetaryCurrencyRequiredError(fieldName || 'unknown');
  }

  const currencyVal = inputRecord[currencyField] !== undefined ? inputRecord[currencyField] : currentRecord[currencyField];
  const digitsFromObj = readDecimalDigitsFromCurrencyValue(currencyVal);
  if (digitsFromObj != null) {
    return { scale: digitsFromObj, currencyId: normalizeCurrencyRefId(currencyVal), needsBrowse: false };
  }

  const currencyId = normalizeCurrencyRefId(currencyVal);
  if (!currencyId) {
    throw monetaryCurrencyRequiredError(fieldName || currencyField);
  }
  return { currencyId, needsBrowse: true };
}

export function resolveMonetaryScaleForWrite(fm: FieldMetadata | undefined, input: unknown, current?: unknown): number {
  const resolved = resolveMonetaryScaleFromPayload(fm, input, current);
  if (typeof resolved.scale === 'number') return resolved.scale;
  throw monetaryCurrencyRequiredError(String(fm?.name || 'unknown'));
}

export function resolveMonetaryScaleFromRow(fm: FieldMetadata | undefined, fieldName: string, row: unknown): number | undefined {
  if (!fm || fm.type !== 'monetary') return undefined;
  const rowRecord = asObjectRecord(row);
  if (!rowRecord) return undefined;

  const hidden = rowRecord[buildHiddenScaleAlias(fieldName)];
  if (hidden != null && Number.isInteger(Number(hidden))) return Number(hidden);

  const currencyField = getCurrencyFieldName(fm.column);
  if (!currencyField) return undefined;

  const digitsFromObj = readDecimalDigitsFromCurrencyValue(rowRecord[currencyField]);
  if (digitsFromObj != null) return digitsFromObj;

  return undefined;
}

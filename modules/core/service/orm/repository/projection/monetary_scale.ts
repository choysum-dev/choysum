// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { Entity } from '../types';
import type { FieldMetadata, ModelMetadata } from '../../metadata';
import { getCurrencyFieldName } from '../../metadata/decimal_like';
import { normalizeCurrencyRefId, readDecimalDigitsFromCurrencyValue } from '../../metadata/monetary_currency';
import {
  monetaryCurrencyRequiredError,
  resolveMonetaryScaleFromPayload,
} from '../../metadata/monetary_scale_resolve';
import { buildHiddenScaleAlias } from '../hidden_scale_alias';
import { asObjectRecord } from '@/core/utils/object';

export {
  monetaryCurrencyRequiredError,
  resolveMonetaryScaleForWrite,
  resolveMonetaryScaleFromRow,
  resolveMonetaryScaleFromPayload,
} from '../../metadata/monetary_scale_resolve';

type CurrencyRow = { Id?: string; DecimalDigits?: number };

type CurrencyBrowseCtor = {
  BrowseMany?: (ids: string[], fields?: string[]) => Promise<CurrencyRow[]>;
};

export type MonetaryDigitsBrowser = (currencyIds: string[]) => Promise<Map<string, number>>;

export async function browseCurrencyDecimalDigits(currencyIds: string[]): Promise<Map<string, number>> {
  const out = new Map<string, number>();
  const ids = [...new Set(currencyIds.map(id => String(id || '').trim()).filter(Boolean))];
  if (!ids.length) return out;

  const { default: BaseModelCtor } = await import('../../model/model');
  const Currency = BaseModelCtor.resolveModelConstructor('base.Currency') as unknown as CurrencyBrowseCtor | undefined;
  if (!Currency || typeof Currency.BrowseMany !== 'function') {
    return out;
  }

  const rows = await Currency.BrowseMany(ids, ['Id', 'DecimalDigits']);
  for (const row of rows || []) {
    const id = String(row?.Id || '').trim();
    const digits = Number(row?.DecimalDigits);
    if (id && Number.isInteger(digits) && digits >= 0 && digits <= 18) {
      out.set(id, digits);
    }
  }
  return out;
}

/**
 * Stamp hidden decimal-scale aliases onto write payloads so sync encodeForDb / validation can quantize.
 * Throws E1 when a monetary value is present but currency cannot be resolved.
 */
export async function stampMonetaryScalesForWrite(
  meta: ModelMetadata,
  input: Entity,
  current?: Entity | null,
  browseDigits: MonetaryDigitsBrowser = browseCurrencyDecimalDigits
): Promise<Entity> {
  const inputRecord = asObjectRecord(input);
  if (!inputRecord || !meta.fields) return input;

  const pendingIds: string[] = [];
  const pendingFields: Array<{ fieldName: string; currencyId: string }> = [];

  for (const [fieldName, fm] of meta.fields) {
    if (!fm || fm.type !== 'monetary') continue;
    const writingAmount = fieldName in inputRecord && inputRecord[fieldName] != null && inputRecord[fieldName] !== '';
    const currencyField = getCurrencyFieldName(fm.column);
    const writingCurrency = !!currencyField && currencyField in inputRecord;
    if (!writingAmount && !writingCurrency) continue;

    const resolved = resolveMonetaryScaleFromPayload({ ...fm, name: fieldName } as FieldMetadata, input, current);
    if (typeof resolved.scale === 'number') {
      inputRecord[buildHiddenScaleAlias(fieldName)] = resolved.scale;
      continue;
    }
    if (resolved.needsBrowse && resolved.currencyId) {
      pendingIds.push(resolved.currencyId);
      pendingFields.push({ fieldName, currencyId: resolved.currencyId });
      continue;
    }
    if (writingAmount) {
      throw monetaryCurrencyRequiredError(fieldName);
    }
  }

  if (pendingFields.length) {
    const digitsById = await browseDigits(pendingIds);
    for (const { fieldName, currencyId } of pendingFields) {
      const digits = digitsById.get(currencyId);
      if (digits == null) {
        throw monetaryCurrencyRequiredError(fieldName);
      }
      inputRecord[buildHiddenScaleAlias(fieldName)] = digits;
    }
  }

  return input;
}

/** Companion currency field names for monetary fields in `fieldNames`. */
export function collectMonetaryCurrencyFieldCompanions(meta: ModelMetadata, fieldNames: Iterable<string>): string[] {
  const out: string[] = [];
  for (const fieldName of fieldNames) {
    const fm = meta.fields?.get(fieldName);
    if (!fm || fm.type !== 'monetary') continue;
    const currencyField = getCurrencyFieldName(fm.column);
    if (currencyField) out.push(currencyField);
  }
  return out;
}

export function readCurrencyDigitsInline(value: unknown): number | undefined {
  return readDecimalDigitsFromCurrencyValue(value);
}

export function currencyIdOf(value: unknown): string | undefined {
  return normalizeCurrencyRefId(value);
}

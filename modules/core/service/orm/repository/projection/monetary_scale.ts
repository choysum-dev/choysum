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

  const { resolveModelConstructor } = await import('../../model/model_registry');
  const Currency = resolveModelConstructor('base.Currency') as unknown as CurrencyBrowseCtor | undefined;
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
    // resolveMonetaryScaleFromPayload already throws E1 when currency cannot be resolved.
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

function collectPendingCurrencyIdsForStamp(meta: ModelMetadata, input: Entity, current?: Entity | null): string[] {
  const inputRecord = asObjectRecord(input);
  if (!inputRecord || !meta.fields) return [];
  const ids: string[] = [];
  for (const [fieldName, fm] of meta.fields) {
    if (!fm || fm.type !== 'monetary') continue;
    const writingAmount = fieldName in inputRecord && inputRecord[fieldName] != null && inputRecord[fieldName] !== '';
    const currencyField = getCurrencyFieldName(fm.column);
    const writingCurrency = !!currencyField && currencyField in inputRecord;
    if (!writingAmount && !writingCurrency) continue;
    try {
      const resolved = resolveMonetaryScaleFromPayload({ ...fm, name: fieldName } as FieldMetadata, input, current);
      if (resolved.needsBrowse && resolved.currencyId) ids.push(resolved.currencyId);
    } catch {
      // E1 is raised during the actual stamp pass.
    }
  }
  return ids;
}

/**
 * Stamp monetary scales for many write payloads with one batched Currency.BrowseMany.
 */
export async function stampMonetaryScalesForWriteMany(
  meta: ModelMetadata,
  items: Array<{ input: Entity; current?: Entity | null }>,
  browseDigits: MonetaryDigitsBrowser = browseCurrencyDecimalDigits
): Promise<Entity[]> {
  if (!items.length) return [];

  const pendingIds: string[] = [];
  for (const item of items) {
    pendingIds.push(...collectPendingCurrencyIdsForStamp(meta, item.input, item.current));
  }
  const digitsById = await browseDigits(pendingIds);
  const cachedBrowse: MonetaryDigitsBrowser = async ids => {
    const out = new Map<string, number>();
    for (const id of ids) {
      const digits = digitsById.get(id);
      if (digits != null) out.set(id, digits);
    }
    return out;
  };

  const out: Entity[] = [];
  for (const item of items) {
    out.push(await stampMonetaryScalesForWrite(meta, item.input, item.current, cachedBrowse));
  }
  return out;
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

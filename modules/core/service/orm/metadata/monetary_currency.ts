// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { ModelMetadata } from './model';
import type { FieldMetadata, ManyToOneMetadata } from './field';
import { MetadataStorage } from './storage';
import { getCurrencyFieldName } from './decimal_like';
import { asObjectRecord } from '@/core/utils/object';
import type BaseModel from '../model/model';

export const BASE_CURRENCY_FULL_NAME = 'base.Currency';

function resolveManyToOneTargetFullName(sibling: FieldMetadata): string | undefined {
  const relation = sibling.relation as ManyToOneMetadata<BaseModel> | undefined;
  const targetModel = relation?.targetModel;
  if (typeof targetModel !== 'function') return undefined;
  try {
    const ctor = targetModel();
    if (!ctor) return undefined;
    const meta = MetadataStorage.instance.getModelMetadata(ctor as never);
    const full = String(meta?.fullModelName || '').trim();
    return full || undefined;
  } catch {
    return undefined;
  }
}

function resolveRelationTargetModelName(relation: FieldMetadata['relation']): string | undefined {
  if (!relation) return undefined;
  const targetModel = (relation as { targetModel?: unknown }).targetModel;
  if (typeof targetModel === 'string') {
    const trimmed = targetModel.trim();
    return trimmed || undefined;
  }
  return undefined;
}

/** True when sibling is ManyToOne / ManyToOneRef targeting base.Currency. */
export function isCurrencyRelationField(sibling: FieldMetadata | undefined): boolean {
  if (!sibling) return false;
  if (sibling.type === 'ManyToOneRef') {
    return resolveRelationTargetModelName(sibling.relation) === BASE_CURRENCY_FULL_NAME;
  }
  if (sibling.type === 'ManyToOne') {
    return resolveManyToOneTargetFullName(sibling) === BASE_CURRENCY_FULL_NAME;
  }
  return false;
}

export function validateModelMonetaryCurrencyFields(meta: ModelMetadata): void {
  const fields = meta.fields;
  if (!fields) return;

  for (const [fieldName, fm] of fields) {
    if (!fm || fm.type !== 'monetary') continue;
    const currencyField = getCurrencyFieldName(fm.column);
    if (!currencyField) {
      throw new Error(`@Field(${fieldName}) monetary requires currencyField`);
    }
    const sibling = fields.get(currencyField);
    if (!sibling) {
      throw new Error(`@Field(${fieldName}) currencyField "${currencyField}" does not exist on the model`);
    }
    if (!isCurrencyRelationField(sibling)) {
      throw new Error(
        `@Field(${fieldName}) currencyField "${currencyField}" must be ManyToOne or ManyToOneRef targeting ${BASE_CURRENCY_FULL_NAME}`
      );
    }
  }
}

export function normalizeCurrencyRefId(value: unknown): string | undefined {
  if (value == null) return undefined;
  if (typeof value === 'string') {
    const trimmed = value.trim();
    return trimmed || undefined;
  }
  const record = asObjectRecord(value);
  if (!record) return undefined;
  const id = record.Id ?? record.id;
  if (typeof id === 'string' && id.trim()) return id.trim();
  return undefined;
}

export function readDecimalDigitsFromCurrencyValue(value: unknown): number | undefined {
  const record = asObjectRecord(value);
  if (!record) return undefined;
  const raw = record.DecimalDigits ?? record.decimalDigits;
  const n = Number(raw);
  if (!Number.isInteger(n) || n < 0 || n > 18) return undefined;
  return n;
}

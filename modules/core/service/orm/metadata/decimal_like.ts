// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { FieldMetadata, FieldType } from './field';

/** Field types that share decimal storage / Decimal codec paths. */
export function isDecimalLikeFieldType(type: FieldType | string | undefined): boolean {
  return type === 'decimal' || type === 'monetary';
}

export function isDecimalLikeField(fm: FieldMetadata | undefined | null): boolean {
  return isDecimalLikeFieldType(fm?.type);
}

export function getCurrencyFieldName(column: unknown): string | undefined {
  if (!column || typeof column !== 'object') return undefined;
  const currencyField = (column as { currencyField?: unknown }).currencyField;
  if (typeof currencyField !== 'string') return undefined;
  const trimmed = currencyField.trim();
  return trimmed || undefined;
}

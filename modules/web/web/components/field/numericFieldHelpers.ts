// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { nextTick, type Ref } from 'vue';
import { formatFixedDecimalString } from '@/web/web/stores/i18nStore';

export type NumericNumberFormat = {
  thousandsSeparator?: string;
  decimalSeparator?: string;
  grouping?: number[];
};

/**
 * When a controlled Input rejects invalid keystrokes, the browser has already
 * mutated the DOM while Vue keeps the prior modelValue. Force the input (or
 * bounce the controlled prop) back to the kept raw string.
 */
export function resyncControlledNumericRaw(
  editingRaw: Ref<string | null>,
  inputEl?: HTMLInputElement | null
): void {
  const keep = editingRaw.value ?? '';
  if (inputEl && inputEl.value !== keep) {
    inputEl.value = keep;
    return;
  }
  const prev = editingRaw.value;
  editingRaw.value = prev === null ? '' : null;
  void nextTick(() => {
    editingRaw.value = prev;
  });
}

function isHtmlInputElement(el: unknown): el is HTMLInputElement {
  // Prefer tagName: QuickJS DOM shims may not implement HTMLInputElement instanceof reliably.
  return !!el && typeof el === 'object' && String((el as { tagName?: unknown }).tagName || '').toUpperCase() === 'INPUT';
}

/** Resolve the native input element from a Vue component / element ref. */
export function resolveNumericInputEl(host: unknown): HTMLInputElement | null {
  if (!host) return null;
  if (isHtmlInputElement(host)) return host;
  const el = (host as { $el?: unknown }).$el;
  if (isHtmlInputElement(el)) return el;
  return null;
}

/** Group a canonical numeric string for display (edit stays raw). */
export function formatGroupedNumericString(
  raw: string,
  numberFormat?: NumericNumberFormat | null
): string {
  if (!raw || !numberFormat) return raw;
  try {
    return formatFixedDecimalString(raw, numberFormat);
  } catch {
    return raw;
  }
}

/** Form display shows muted empty placeholder; table/inline cells stay blank. */
export function isFormEmptyPlaceholderMode(
  renderMode: string | undefined,
  isForm: boolean | undefined
): boolean {
  if (renderMode === 'table' || renderMode === 'inline') return false;
  if (renderMode === 'form') return true;
  return isForm !== false;
}

/** List column types that compare as quantities — align to the inline end. */
const END_ALIGNED_LIST_FIELD_TYPES = new Set([
  'int',
  'integer',
  'bigint',
  'float',
  'double',
  'number',
  'decimal',
  'monetary',
]);

/** Fixed-width controls / visual blocks — center in the column. */
const CENTER_ALIGNED_LIST_FIELD_TYPES = new Set(['boolean', 'binary', 'image']);

export type ListColumnAlign = 'start' | 'center' | 'end';

function normalizeListFieldType(fieldType: unknown): string {
  return String(fieldType || '')
    .trim()
    .toLowerCase();
}

export function isEndAlignedListFieldType(fieldType: unknown): boolean {
  return END_ALIGNED_LIST_FIELD_TYPES.has(normalizeListFieldType(fieldType));
}

export function isCenterAlignedListFieldType(fieldType: unknown): boolean {
  return CENTER_ALIGNED_LIST_FIELD_TYPES.has(normalizeListFieldType(fieldType));
}

/**
 * Default list column align by field type:
 * - Number / Monetary family → `end`
 * - boolean / binary / image → `center`
 * - otherwise leave unset (content follows start)
 */
export function defaultListColumnAlignForFieldType(fieldType: unknown): ListColumnAlign | undefined {
  if (isEndAlignedListFieldType(fieldType)) return 'end';
  if (isCenterAlignedListFieldType(fieldType)) return 'center';
  return undefined;
}

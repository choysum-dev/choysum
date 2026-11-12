// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * FE display fallback for selection option labels (pairs with resolveFieldLabel D6).
 *
 * FieldsGet returns plain `{ value, label }` strings and strips labelText (D5/D15).
 * When BE `_t` misses, label is the English msgid echo — same problem field titles
 * had before resolveFieldLabel's stringText path. Re-translate via the field's
 * stringText module/scope + option label as src when a composer catalog is present.
 *
 * Already-translated labels (e.g. 浅色) miss in the catalog and pass through.
 */

import { createTermReference, type TermReference } from '@/core/service/i18n';
import { translateTerm } from '@/web/web/i18n/translate';

export type ResolveSelectionOptionLabelOptions = {
  /** FieldsGet / static meta option label (msgid or already translated). */
  label: string;
  /** Field stringText — supplies module/scope for option msgid lookup. */
  fieldStringText?: TermReference | null;
  composer?: unknown;
};

/**
 * Resolve one selection option label for display.
 */
export function resolveSelectionOptionLabel(options: ResolveSelectionOptionLabelOptions): string {
  const label = String(options.label || '').trim();
  if (!label) return '';
  const fieldRef = options.fieldStringText;
  if (!fieldRef || typeof fieldRef.module !== 'string' || typeof fieldRef.scope !== 'string') {
    return label;
  }
  const optionRef = createTermReference(fieldRef.module, label, {
    scope: fieldRef.scope,
  });
  return translateTerm(options.composer, optionRef, label);
}

// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Shared field title resolution (design D6).
 *
 * Priority:
 * 1. props.label when provided (including '' to force empty)
 * 2. FieldsGet translated string for current lang (when overlay present)
 * 3. translateTerm(composer, stringText, string)
 * 4. meta.string (msgid)
 * 5. prop leaf name
 */

import type { TermReference } from '@/core/service/i18n';
import { translateTerm } from '@/web/web/i18n/translate';

export type FieldLabelMeta = {
  string?: string;
  stringText?: TermReference;
};

export type ResolveFieldLabelOptions = {
  /** When `undefined`, resolve from meta; when string (incl. ''), use as override. */
  label?: string | null;
  prop?: string | null;
  meta?: FieldLabelMeta | null;
  composer?: unknown;
  /**
   * FieldsGet overlay: translated title for the active language.
   * Pass only when cache hit for current lang (P1+).
   */
  fieldsGetTranslatedString?: string | null;
};

function leafPropName(prop: string | null | undefined): string {
  const raw = String(prop || '').trim();
  if (!raw) return '';
  const segs = raw.split('.').filter(Boolean);
  return segs[segs.length - 1] || raw;
}

export function resolveFieldLabel(options: ResolveFieldLabelOptions): string {
  if (options.label != null) {
    return String(options.label);
  }

  const fromFieldsGet = typeof options.fieldsGetTranslatedString === 'string' ? options.fieldsGetTranslatedString.trim() : '';
  if (fromFieldsGet) {
    return fromFieldsGet;
  }

  const meta = options.meta;
  const msgid = typeof meta?.string === 'string' ? meta.string : '';
  if (meta?.stringText) {
    return translateTerm(options.composer, meta.stringText, msgid);
  }
  if (msgid) {
    return msgid;
  }

  return leafPropName(options.prop);
}

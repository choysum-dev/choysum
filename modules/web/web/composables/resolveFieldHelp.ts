// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Shared field help resolution (PR-P2-F1).
 *
 * Priority:
 * 1. FieldsGet translated help for current lang (when overlay present and not a msgid echo)
 * 2. translateTerm(composer, helpText, help)
 * 3. meta.help (msgid)
 * 4. empty string (no tip)
 */

import type { TermReference } from '@/core/service/i18n';
import { translateTerm } from '@/web/web/i18n/translate';

export type FieldHelpMeta = {
  help?: string;
  helpText?: TermReference;
};

export type ResolveFieldHelpOptions = {
  meta?: FieldHelpMeta | null;
  composer?: unknown;
  /**
   * FieldsGet overlay: translated help for the active language.
   * Pass only when cache hit for current lang.
   */
  fieldsGetTranslatedHelp?: string | null;
};

function metaMsgid(meta: FieldHelpMeta | null | undefined): string {
  const fromRef = typeof meta?.helpText?.src === 'string' ? meta.helpText.src.trim() : '';
  if (fromRef) return fromRef;
  return typeof meta?.help === 'string' ? meta.help.trim() : '';
}

export function resolveFieldHelp(options: ResolveFieldHelpOptions): string {
  const fromFieldsGet = typeof options.fieldsGetTranslatedHelp === 'string' ? options.fieldsGetTranslatedHelp.trim() : '';
  if (fromFieldsGet) {
    const msgid = metaMsgid(options.meta);
    const isMsgidEcho = !!options.meta?.helpText && !!msgid && fromFieldsGet === msgid;
    if (!isMsgidEcho) {
      return fromFieldsGet;
    }
  }

  const meta = options.meta;
  const msgid = typeof meta?.help === 'string' ? meta.help.trim() : '';
  if (meta?.helpText) {
    return String(translateTerm(options.composer, meta.helpText, msgid) || '').trim();
  }
  return msgid;
}

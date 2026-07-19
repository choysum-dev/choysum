// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  createTermReferenceKey,
  TERM_REFERENCE_NAMESPACE,
} from '@/core/service/i18n';

// Use `any` values so the result is assignable to vue-i18n `LocaleMessage`
// (`Record<string, unknown>` is not — it breaks createI18n overload resolution
// and leaves `composer.locale` typed as a plain string).
type MessageRecord = Record<string, any>;

function asRecord(value: unknown): MessageRecord | undefined {
  return value && typeof value === 'object' && !Array.isArray(value)
    ? (value as MessageRecord)
    : undefined;
}

/**
 * Preserve the Gateway's legacy nested catalog and add vue-i18n-safe flat terms.
 */
export function projectTerminologyMessages(messages: unknown): MessageRecord {
  const root = asRecord(messages) ?? {};
  const terms: Record<string, string> = {};
  const existingTerms = asRecord(root[TERM_REFERENCE_NAMESPACE]);
  if (existingTerms) {
    for (const [key, value] of Object.entries(existingTerms)) {
      if (typeof value === 'string') terms[key] = value;
    }
  }

  for (const [module, scopesValue] of Object.entries(root)) {
    if (module === TERM_REFERENCE_NAMESPACE) continue;
    const scopes = asRecord(scopesValue);
    if (!scopes) continue;
    for (const [scope, sourcesValue] of Object.entries(scopes)) {
      const sources = asRecord(sourcesValue);
      if (!sources) continue;
      for (const [src, value] of Object.entries(sources)) {
        if (typeof value !== 'string' || value === '') continue;
        const fullKey = createTermReferenceKey(module, scope, src, 'literal');
        const segment = fullKey.slice(TERM_REFERENCE_NAMESPACE.length + 1);
        terms[segment] = value;
      }
    }
  }

  return {
    ...root,
    [TERM_REFERENCE_NAMESPACE]: terms,
  };
}

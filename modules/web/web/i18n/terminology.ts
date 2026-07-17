// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  createTextDescriptorKey,
  TEXT_DESCRIPTOR_NAMESPACE,
} from '@/core/service/i18n';

type MessageRecord = Record<string, unknown>;

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
  const existingTerms = asRecord(root[TEXT_DESCRIPTOR_NAMESPACE]);
  if (existingTerms) {
    for (const [key, value] of Object.entries(existingTerms)) {
      if (typeof value === 'string') terms[key] = value;
    }
  }

  for (const [module, scopesValue] of Object.entries(root)) {
    if (module === TEXT_DESCRIPTOR_NAMESPACE) continue;
    const scopes = asRecord(scopesValue);
    if (!scopes) continue;
    for (const [scope, sourcesValue] of Object.entries(scopes)) {
      const sources = asRecord(sourcesValue);
      if (!sources) continue;
      for (const [src, value] of Object.entries(sources)) {
        if (typeof value !== 'string' || value === '') continue;
        const fullKey = createTextDescriptorKey(module, scope, src, 'literal');
        const segment = fullKey.slice(TEXT_DESCRIPTOR_NAMESPACE.length + 1);
        terms[segment] = value;
      }
    }
  }

  return {
    ...root,
    [TEXT_DESCRIPTOR_NAMESPACE]: terms,
  };
}

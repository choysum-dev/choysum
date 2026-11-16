// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { TerminologyLoadResult } from './terminologyLoader';

function hasCatalogContent(value: unknown): boolean {
  if (typeof value === 'string') {
    return value !== '';
  }
  if (!value || typeof value !== 'object') {
    return false;
  }
  return Object.values(value).some(hasCatalogContent);
}

/**
 * Whether vue-i18n should mergeLocaleMessage for a Gateway load result (D4).
 * unchanged / gatewayError / null messages → do not merge empty objects.
 */
export function shouldMergeTerminology(load: TerminologyLoadResult | null | undefined): boolean {
  if (!load || load.unchanged || load.gatewayError) {
    return false;
  }
  return Boolean(load.hash.trim()) && hasCatalogContent(load.messages);
}

type CatalogMergerOptions = {
  merge: (lang: string, messages: TerminologyLoadResult['messages']) => void;
  notify: () => void;
};

/**
 * Applies each successful Gateway catalog exactly once per lang + catalogHash.
 * Identity is recorded only after merge succeeds, so failed merges can retry.
 */
export function createTerminologyCatalogMerger({ merge, notify }: CatalogMergerOptions) {
  const mergedCatalogs = new Set<string>();

  return (load: TerminologyLoadResult | null | undefined, fallbackLang: string): boolean => {
    if (!shouldMergeTerminology(load) || !load?.messages) {
      return false;
    }

    const lang = String(load.lang || fallbackLang).trim();
    if (!lang) {
      return false;
    }
    const identity = `${lang}\0${load.hash.trim()}`;
    if (mergedCatalogs.has(identity)) {
      return false;
    }

    merge(lang, load.messages);
    mergedCatalogs.add(identity);
    notify();
    return true;
  };
}

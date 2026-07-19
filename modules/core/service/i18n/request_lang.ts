// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { RequestContextKV } from '../../rpc/context';
import { getCurrentRequestContext } from '../../rpc/context';

export type ResolveRequestLangFallbacks = {
  /** User.Language when baggage lang is absent. */
  userLanguage?: string;
  /** Company default language. */
  companyDefault?: string;
  /** Final fallback (default en_US). */
  final?: string;
};

/**
 * Resolve terminology language code (e.g. zh_CN).
 * Does NOT treat `locale` (zh-CN) as lang — §13 D12d.
 *
 * Order: baggage `lang` → userLanguage → companyDefault → en_US.
 */
export function resolveRequestLang(
  kv?: RequestContextKV,
  fallbacks?: ResolveRequestLangFallbacks,
): string {
  const ctx = kv ?? getCurrentRequestContext();
  const lang = typeof ctx.lang === 'string' ? ctx.lang.trim() : '';
  if (lang) {
    return lang;
  }
  // Intentionally ignore ctx.locale (format code ≠ terminology lang).
  const user = fallbacks?.userLanguage?.trim();
  if (user) {
    return user;
  }
  const company = fallbacks?.companyDefault?.trim();
  if (company) {
    return company;
  }
  return fallbacks?.final?.trim() || 'en_US';
}

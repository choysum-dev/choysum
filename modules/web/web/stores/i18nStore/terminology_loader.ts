// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Host Gateway terminology loader (GET /web/i18n/translations).
 */

export type TerminologyMessages = Record<string, Record<string, Record<string, string>>>;

export type WebTranslationsResponse = {
  lang: string;
  locale: string;
  hash: string;
  unchanged: boolean;
  messages: TerminologyMessages | null;
};

export type TerminologyLoadResult = WebTranslationsResponse & {
  /** Gateway failed; UI should keep msgid / existing messages. */
  gatewayError?: boolean;
};

export type FetchWebTranslationsOptions = {
  /** Override fetch (tests). */
  fetchImpl?: typeof fetch;
  /** Base URL prefix (default ''). */
  baseUrl?: string;
};

/**
 * Fetch aggregated terminology messages from the host Gateway.
 * Does not pass moduleNames — the host fills that server-side.
 */
export async function fetchWebTranslations(
  lang: string,
  hash?: string,
  options?: FetchWebTranslationsOptions
): Promise<WebTranslationsResponse> {
  const terminologyLang = String(lang || '').trim();
  if (!terminologyLang) {
    throw new Error('lang is required');
  }

  const params = new URLSearchParams({ lang: terminologyLang });
  const catalogHash = String(hash || '').trim();
  if (catalogHash) {
    params.set('hash', catalogHash);
  }

  const base = String(options?.baseUrl ?? '').replace(/\/$/, '');
  const url = `${base}/web/i18n/translations?${params.toString()}`;
  const fetchFn = options?.fetchImpl ?? globalThis.fetch;
  if (typeof fetchFn !== 'function') {
    throw new Error('fetch is not available');
  }

  const res = await fetchFn(url, {
    method: 'GET',
    credentials: 'same-origin',
    headers: { Accept: 'application/json' },
  });
  if (!res.ok) {
    throw new Error(`translations gateway HTTP ${res.status}`);
  }

  const body = (await res.json()) as Partial<WebTranslationsResponse>;
  const unchanged = body.unchanged === true;
  return {
    lang: String(body.lang || terminologyLang),
    locale: String(body.locale || ''),
    hash: String(body.hash || ''),
    unchanged,
    messages: unchanged ? null : ((body.messages as TerminologyMessages) ?? {}),
  };
}

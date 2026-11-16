// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { bcp47FromCode } from './bcp47';
import { fetchWebTranslations, type TerminologyLoadResult } from './terminologyLoader';

export type RevalidateTerminologyOptions = {
  lang: string;
  prevHash?: string;
  fetchImpl?: typeof fetchWebTranslations;
};

/**
 * Cheap Gateway hash negotiation for hotreload / tab-focus refresh.
 * Returns unchanged=true when catalogHash matches prevHash (no messages body).
 */
export async function revalidateTerminologyCatalog(
  options: RevalidateTerminologyOptions
): Promise<TerminologyLoadResult> {
  const lang = String(options.lang || '').trim();
  const prevHash = String(options.prevHash || '').trim();
  const fetchImpl = options.fetchImpl || fetchWebTranslations;
  const res = await fetchImpl(lang, prevHash || undefined);
  if (res.unchanged) {
    return {
      lang: res.lang || lang,
      bcp47: res.bcp47 || bcp47FromCode(lang),
      hash: res.hash || prevHash,
      unchanged: true,
      messages: null,
    };
  }
  return {
    lang: res.lang || lang,
    bcp47: res.bcp47 || bcp47FromCode(lang),
    hash: res.hash,
    unchanged: false,
    messages: res.messages ?? {},
  };
}

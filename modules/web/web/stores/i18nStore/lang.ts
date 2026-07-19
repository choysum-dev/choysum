// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Map UI locale (zh-CN) ↔ terminology lang (zh_CN).
 * locale is for Element/dayjs/format; lang is for Gateway / _t (D12d).
 */

const LOCALE_TO_LANG: Record<string, string> = {
  en: 'en_US',
  'en-GB': 'en_GB',
  'zh-CN': 'zh_CN',
  'zh-TW': 'zh_TW',
  ja: 'ja_JP',
  ko: 'ko_KR',
  de: 'de_DE',
  fr: 'fr_FR',
  es: 'es_ES',
  it: 'it_IT',
  pt: 'pt_PT',
  'pt-BR': 'pt_BR',
  ru: 'ru_RU',
  ar: 'ar_001',
};

const LANG_TO_LOCALE: Record<string, string> = Object.fromEntries(
  Object.entries(LOCALE_TO_LANG).map(([locale, lang]) => [lang, locale])
);

/** Convert UI locale code to terminology lang (e.g. zh-CN → zh_CN). */
export function localeToLang(locale: string): string {
  const code = String(locale || '').trim();
  if (!code) {
    return 'en_US';
  }
  if (LOCALE_TO_LANG[code]) {
    return LOCALE_TO_LANG[code];
  }
  if (code.includes('_')) {
    return code;
  }
  return code.replace(/-/g, '_');
}

/** Convert terminology lang to UI locale (e.g. zh_CN → zh-CN). */
export function langToLocale(lang: string): string {
  const code = String(lang || '').trim();
  if (!code) {
    return 'en';
  }
  if (LANG_TO_LOCALE[code]) {
    return LANG_TO_LOCALE[code];
  }
  if (code.includes('-')) {
    return code;
  }
  return code.replace(/_/g, '-');
}

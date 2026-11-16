// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { afterLangChange, type useI18nStore } from '../../stores/i18nStore';
import { DEFAULT_ACTIVE_LANGS } from '../../stores/i18nStore/activeLangs';
import type { ActiveLanguageRow } from '../../stores/i18nStore/types';
import { shellLanguageSelect } from './shellLanguageSelect';

type I18nStore = ReturnType<typeof useI18nStore>;

export function languageMenuRows(
  store: I18nStore | null | undefined,
  defaults: readonly string[] = DEFAULT_ACTIVE_LANGS,
): ActiveLanguageRow[] {
  const fromStore = store?.activeLanguages;
  if (Array.isArray(fromStore) && fromStore.length) {
    return fromStore;
  }
  return defaults.map(code => ({ Code: code, Name: code }));
}

export function languageMenuCurrentCode(store: I18nStore | null | undefined): string {
  return store?.activeLanguage.code || 'en_US';
}

export async function runLanguageMenuSelect(
  code: string,
  store: I18nStore | null | undefined,
  remount: () => Promise<void> = afterLangChange,
): Promise<void> {
  if (!store) {
    return;
  }
  await shellLanguageSelect(code, c => store.setLang(c), remount);
}
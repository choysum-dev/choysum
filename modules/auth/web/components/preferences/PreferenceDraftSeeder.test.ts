// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { seedPreferenceDraft } from './PreferenceDraftSeeder';

test('seedPreferenceDraft: fills empty LanguageId and Timezone', () => {
  const draft: Record<string, unknown> = { LanguageId: '', Timezone: '' };
  seedPreferenceDraft(
    {
      getField: path => draft[path],
      setField: (path, value) => {
        draft[path] = value;
      },
    },
    { languageId: 'lang-1', timezone: 'Asia/Shanghai' },
  );
  expect(draft.LanguageId).toBe('lang-1');
  expect(draft.Timezone).toBe('Asia/Shanghai');
});

test('seedPreferenceDraft: does not overwrite persisted LanguageId / Timezone', () => {
  const draft: Record<string, unknown> = { LanguageId: 'saved-lang', Timezone: 'UTC' };
  seedPreferenceDraft(
    {
      getField: path => draft[path],
      setField: (path, value) => {
        draft[path] = value;
      },
    },
    { languageId: 'hint-lang', timezone: 'Asia/Shanghai' },
  );
  expect(draft.LanguageId).toBe('saved-lang');
  expect(draft.Timezone).toBe('UTC');
});

test('seedPreferenceDraft: no-ops without form-root', () => {
  seedPreferenceDraft(null, { languageId: 'lang-1', timezone: 'UTC' });
});

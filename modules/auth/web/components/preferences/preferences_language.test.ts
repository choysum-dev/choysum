// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { resolveLanguageCodeFromId } from './preferences_language';

test('resolveLanguageCodeFromId: empty id skips Browse', async () => {
  let called = false;
  const code = await resolveLanguageCodeFromId('', async () => {
    called = true;
    return { Code: 'zh_CN' };
  });
  expect(code).toBe('');
  expect(called).toBe(false);
});

test('resolveLanguageCodeFromId: returns Code from Browse', async () => {
  const code = await resolveLanguageCodeFromId('lang_1', async () => ({ Code: ' en_US ' }));
  expect(code).toBe('en_US');
});

test('resolveLanguageCodeFromId: Browse errors become empty string', async () => {
  const code = await resolveLanguageCodeFromId('lang_1', async () => {
    throw new Error('boom');
  });
  expect(code).toBe('');
});

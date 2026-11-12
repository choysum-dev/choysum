// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { revalidateTerminologyCatalog } from './terminologyRevalidate';
import { shouldMergeTerminology } from './merge';

test('revalidateTerminologyCatalog: unchanged skips messages', async () => {
  const load = await revalidateTerminologyCatalog({
    lang: 'zh_CN',
    locale: 'zh-CN',
    prevHash: 'abc',
    fetchImpl: async () => ({
      lang: 'zh_CN',
      locale: 'zh-CN',
      hash: 'abc',
      unchanged: true,
      messages: { auth: { 'auth.model.User.fields': { Light: '浅色' } } },
    }),
  });
  expect(load.unchanged).toBe(true);
  expect(load.messages).toBe(null);
  expect(shouldMergeTerminology(load)).toBe(false);
});

test('revalidateTerminologyCatalog: hash change returns messages for merge', async () => {
  const messages = { auth: { 'auth.model.User.fields': { Light: '浅色' } } };
  const load = await revalidateTerminologyCatalog({
    lang: 'zh_CN',
    locale: 'zh-CN',
    prevHash: 'old',
    fetchImpl: async (lang, hash) => {
      expect(lang).toBe('zh_CN');
      expect(hash).toBe('old');
      return {
        lang: 'zh_CN',
        locale: 'zh-CN',
        hash: 'new',
        unchanged: false,
        messages,
      };
    },
  });
  expect(load.unchanged).toBe(false);
  expect(load.hash).toBe('new');
  expect(load.messages).toEqual(messages);
  expect(shouldMergeTerminology(load)).toBe(true);
});

// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { fnRecorder } from '@/web/web/__tests__/mountApp';
import { shellLanguageSelect } from './shellLanguageSelect';

test('shellLanguageSelect: setLang then remount on success', async () => {
  const setLang = fnRecorder(async () => true);
  const remount = fnRecorder(async () => undefined);
  await shellLanguageSelect('zh_CN', setLang, remount);
  expect(setLang.calls).toEqual([['zh_CN']]);
  expect(remount.calls.length).toBe(1);
});

test('shellLanguageSelect: calls setLang when code matches current', async () => {
  const setLang = fnRecorder(async () => true);
  const remount = fnRecorder(async () => undefined);
  await shellLanguageSelect('en_US', setLang, remount);
  expect(setLang.calls).toEqual([['en_US']]);
});

test('shellLanguageSelect: skips remount when setLang fails', async () => {
  const setLang = fnRecorder(async () => false);
  const remount = fnRecorder(async () => undefined);
  await shellLanguageSelect('zh_CN', setLang, remount);
  expect(setLang.calls).toEqual([['zh_CN']]);
  expect(remount.calls).toEqual([]);
});

test('shellLanguageSelect: ignores empty code', async () => {
  const setLang = fnRecorder(async () => true);
  await shellLanguageSelect('', setLang, async () => undefined);
  expect(setLang.calls).toEqual([]);
});
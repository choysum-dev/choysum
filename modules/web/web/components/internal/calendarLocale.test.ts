// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createPinia, setActivePinia } from 'pinia';
import { calendarLocaleFromBcp47 } from './calendarLocale';
import { useI18nStore } from '@/web/web/stores/i18nStore';

test('calendarLocaleFromBcp47: trims and falls back', () => {
  expect(calendarLocaleFromBcp47('  zh-CN  ')).toBe('zh-CN');
  expect(calendarLocaleFromBcp47('')).toBe('en-US');
  expect(calendarLocaleFromBcp47(null)).toBe('en-US');
});

test('calendarLocaleFromBcp47: follows store bcp47Locale', () => {
  setActivePinia(createPinia());
  const store = useI18nStore();
  store.lang = 'zh_CN';
  store.activeLanguages = [{ Code: 'zh_CN' }];
  expect(calendarLocaleFromBcp47(store.bcp47Locale)).toBe('zh-CN');
});
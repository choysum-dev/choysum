// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createPinia, setActivePinia } from 'pinia';
import { calendarLocaleFromBcp47, resolveCalendarLocale } from './calendarLocale';
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

test('resolveCalendarLocale: uses store bcp47 or en-US without pinia', () => {
  setActivePinia(undefined as any);
  expect(resolveCalendarLocale()).toBe('en-US');
  setActivePinia(createPinia());
  const store = useI18nStore();
  store.lang = 'en_US';
  store.activeLanguages = [{ Code: 'en_US' }];
  expect(resolveCalendarLocale()).toBe('en-US');
});
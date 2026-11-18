// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createPinia, setActivePinia } from 'pinia';
import {
  calendarLocaleFromBcp47,
  normalizeWeekStartsOn,
  resolveActiveDateDisplayFormat,
  resolveActiveDateTimeDisplayFormat,
  resolveActiveTimeDisplayFormat,
  resolveCalendarLocale,
  resolveCalendarWeekStartsOn,
} from './calendarLocale';
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

test('normalizeWeekStartsOn: clamps 0-6', () => {
  expect(normalizeWeekStartsOn(0)).toBe(0);
  expect(normalizeWeekStartsOn(6)).toBe(6);
  expect(normalizeWeekStartsOn(1.5)).toBe(0);
  expect(normalizeWeekStartsOn('x', 1)).toBe(1);
});

test('resolveCalendarWeekStartsOn: reads FirstDayOfWeek from active language', () => {
  setActivePinia(undefined as any);
  expect(resolveCalendarWeekStartsOn()).toBe(0);

  setActivePinia(createPinia());
  const store = useI18nStore();
  store.lang = 'zh_CN';
  store.activeLanguages = [{ Code: 'zh_CN' }];
  store.formatByCode = { zh_CN: { FirstDayOfWeek: 1 } };
  expect(resolveCalendarWeekStartsOn()).toBe(1);
  store.formatByCode = { zh_CN: { FirstDayOfWeek: 0 } };
  expect(resolveCalendarWeekStartsOn()).toBe(0);
});

test('resolveActive*DisplayFormat: override wins; else Language overlay', () => {
  setActivePinia(createPinia());
  const store = useI18nStore();
  store.lang = 'de_DE';
  store.activeLanguages = [{ Code: 'de_DE' }];
  store.formatByCode = {
    de_DE: { DateFormat: 'DD.MM.YYYY', TimeFormat: 'HH:mm:ss' },
  };
  expect(resolveActiveDateDisplayFormat('YYYY/MM/DD')).toBe('YYYY/MM/DD');
  expect(resolveActiveDateDisplayFormat()).toBe('DD.MM.YYYY');
  expect(resolveActiveTimeDisplayFormat()).toBe('HH:mm:ss');
  expect(resolveActiveDateTimeDisplayFormat()).toBe('DD.MM.YYYY HH:mm:ss');
});

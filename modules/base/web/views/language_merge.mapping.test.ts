// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import bootstrap from '../../data/bootstrap.json';
import smoke from '../../../auth/e2e/fixtures/smoke.json';
import companyChild from '../../e2e/fixtures/company_child.json';
import * as lang from '@/web/web/stores/i18nStore/lang';
import * as utils from '@/web/web/stores/i18nStore/utils';
import * as languageFormat from '@/web/web/stores/i18nStore/language_format';

test('bootstrap.json has POSIX languages and no Locale entity', () => {
  const data = bootstrap as { records: Array<{ name: string; model: string; values: Record<string, unknown> }> };
  const raw = JSON.stringify(data);

  expect(raw).not.toMatch(/base\.Locale|locale_default|DefaultLocaleId|LocaleId|language_zh[^_]|Code":\s*"zh"/);

  const byId = Object.fromEntries(data.records.map(r => [r.name, r]));
  expect(byId.language_en_us?.model).toBe('Language');
  expect(byId.language_en_us?.values.Code).toBe('en_US');
  expect(byId.language_en_us?.values.Grouping).toBe('[3,0]');
  expect(byId.language_en_us?.values.Name).toEqual({
    en_US: 'English (US)',
    zh_CN: '英语（美国）',
  });

  expect(byId.language_zh_cn?.model).toBe('Language');
  expect(byId.language_zh_cn?.values.Code).toBe('zh_CN');
  expect(byId.language_zh_cn?.values.Grouping).toBe('[3,0]');
  expect(byId.language_zh_cn?.values.DecimalSeparator).toBe('.');
  expect(byId.language_zh_cn?.values.Name).toEqual({
    en_US: 'Chinese (Simplified)',
    zh_CN: '简体中文',
  });

  const order = data.records.map(r => r.name);
  expect(order.indexOf('language_zh_cn')).toBeLessThan(order.indexOf('language_en_us'));
  expect(order.indexOf('language_en_us')).toBeLessThan(order.indexOf('currency_cny'));

  expect(byId.currency_cny?.values.Name).toEqual({
    en_US: 'Chinese Yuan',
    zh_CN: '人民币',
  });

  expect(byId.company_main?.values.LanguageId).toEqual({ ref: 'base.language_zh_cn' });
  expect(byId.company_main?.values).not.toHaveProperty('LocaleId');
});

test('auth smoke and company child fixtures use Language without LocaleId', () => {
  const smokeRaw = JSON.stringify(smoke);
  expect(smokeRaw).not.toMatch(/LocaleId|language_zh[^_]|locale_default/);
  expect(smokeRaw).toMatch(/"Language":\s*"zh_CN"/);

  const companyChildRaw = JSON.stringify(companyChild);
  expect(companyChildRaw).not.toMatch(/LocaleId|locale_default/);
  expect(companyChildRaw).toMatch(/base\.language_zh_cn/);
});

test('FE adapter symbols use UiKey names (no Locale product aliases)', () => {
  expect(Object.keys(lang)).toContain('langToUiKey');
  expect(Object.keys(lang)).toContain('uiKeyToLang');
  expect(Object.keys(lang)).not.toContain('langToLocale');
  expect(Object.keys(lang)).not.toContain('localeToLang');

  expect(Object.keys(utils)).toContain('detectBestUiKey');
  expect(Object.keys(utils)).not.toContain('detectBestLocale');

  expect(Object.keys(languageFormat)).toContain('resolveFormatConfig');
  expect(Object.keys(languageFormat)).toContain('formatNumberFromConfig');
  expect(Object.keys(languageFormat)).toContain('parseGrouping');
});

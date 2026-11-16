// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import bootstrap from '../../data/bootstrap.json';
import languages from '../../data/languages.json';
import smoke from '../../../auth/e2e/fixtures/smoke.json';
import companyChild from '../../e2e/fixtures/company_child.json';
import * as i18nStore from '@/web/web/stores/i18nStore';
import * as bcp47 from '@/web/web/stores/i18nStore/bcp47';
import * as utils from '@/web/web/stores/i18nStore/utils';
import * as languageFormat from '@/web/web/stores/i18nStore/languageFormat';

test('languages.json + bootstrap: POSIX languages and no Locale entity', () => {
  const langData = languages as { records: Array<{ name: string; model: string; values: Record<string, unknown> }> };
  const bootData = bootstrap as { records: Array<{ name: string; model: string; values: Record<string, unknown> }> };
  const raw = JSON.stringify(langData) + JSON.stringify(bootData);

  expect(raw).not.toMatch(/base\.Locale|locale_default|DefaultLocaleId|LocaleId|language_zh[^_]|Code":\s*"zh"/);

  const bootById = Object.fromEntries(bootData.records.map(r => [r.name, r]));
  expect(bootById.language_en_us?.model).toBe('Language');
  expect(bootById.language_en_us?.values.Code).toBe('en_US');
  expect(bootById.language_en_us?.values.Grouping).toBe('[3,0]');
  expect(bootById.language_en_us?.values.PersonNameFormat).toBe('{FirstName} {LastName}');
  expect(bootById.language_en_us?.values.Name).toEqual({
    en_US: 'English (US)',
    zh_CN: '英语（美国）',
  });
  expect(bootById.language_en_us?.values.IsActive).toBe(true);

  expect(bootById.language_zh_cn?.model).toBe('Language');
  expect(bootById.language_zh_cn?.values.Code).toBe('zh_CN');
  expect(bootById.language_zh_cn?.values.Grouping).toBe('[3,0]');
  expect(bootById.language_zh_cn?.values.DecimalSeparator).toBe('.');
  expect(bootById.language_zh_cn?.values.PersonNameFormat).toBe('{LastName}{FirstName}');
  expect(bootById.language_zh_cn?.values.Name).toEqual({
    en_US: 'Chinese (Simplified)',
    zh_CN: '简体中文',
  });
  expect(bootById.language_zh_cn?.values.IsActive).toBe(true);

  const order = bootData.records.map(r => r.name);
  expect(order.indexOf('language_zh_cn')).toBeLessThan(order.indexOf('language_en_us'));
  expect(order.indexOf('language_en_us')).toBeLessThan(order.indexOf('currency_cny'));

  const catalogById = Object.fromEntries(langData.records.map(r => [r.name, r]));
  expect(catalogById.language_en_us).toBeUndefined();
  expect(catalogById.language_zh_cn).toBeUndefined();
  expect(langData.records.length).toBeGreaterThanOrEqual(80);
  expect(langData.records.every(r => r.values.IsActive === false)).toBe(true);
  expect(catalogById.language_ja_jp?.values.Code).toBe('ja_JP');
  expect(catalogById.language_ja_jp?.values.PersonNameFormat).toBe('{LastName}{FirstName}');

  expect(bootById.currency_cny?.values.Name).toEqual({
    en_US: 'Chinese Yuan',
    zh_CN: '人民币',
  });
  expect(bootById.company_main?.values.LanguageId).toEqual({ ref: 'base.language_zh_cn' });
  expect(bootById.company_main?.values).not.toHaveProperty('LocaleId');
});

test('auth smoke and company child fixtures use LanguageId without LocaleId', () => {
  const smokeRaw = JSON.stringify(smoke);
  expect(smokeRaw).not.toMatch(/LocaleId|language_zh[^_]|locale_default/);
  expect(smokeRaw).toMatch(/base\.language_zh_cn/);
  expect(smokeRaw).not.toMatch(/"Language":\s*"zh_CN"/);

  const companyChildRaw = JSON.stringify(companyChild);
  expect(companyChildRaw).not.toMatch(/LocaleId|locale_default/);
  expect(companyChildRaw).toMatch(/base\.language_zh_cn/);
});

test('FE i18n store exports POSIX identity helpers (no legacy uiKey adapters)', () => {
  expect(Object.keys(i18nStore)).toContain('useI18nStore');
  expect(Object.keys(i18nStore)).toContain('afterLangChange');
  expect(Object.keys(i18nStore)).toContain('detectBestLang');
  expect(Object.keys(i18nStore)).toContain('bcp47FromCode');
  expect(Object.keys(i18nStore)).toContain('shouldNoopSetLang');
  expect(Object.keys(i18nStore)).not.toContain('langToUiKey');
  expect(Object.keys(i18nStore)).not.toContain('uiKeyToLang');
  expect(Object.keys(i18nStore)).not.toContain('setUiKey');
  expect(Object.keys(i18nStore)).not.toContain('SUPPORTED_LOCALES');

  expect(Object.keys(bcp47)).toContain('bcp47FromCode');

  expect(Object.keys(utils)).toContain('detectBestLang');
  expect(Object.keys(utils)).toContain('matchBestLang');
  expect(Object.keys(utils)).not.toContain('looksLikeLegacyUiKey');
  expect(Object.keys(utils)).not.toContain('detectBestUiKey');
  expect(Object.keys(utils)).not.toContain('detectBestLocale');

  expect(Object.keys(languageFormat)).toContain('resolveFormatConfig');
  expect(Object.keys(languageFormat)).toContain('collectActiveLanguages');
  expect(Object.keys(languageFormat)).toContain('formatNumberFromConfig');
  expect(Object.keys(languageFormat)).toContain('parseGrouping');
});

// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

function read(rel: string): string {
  return readFileSync(resolve(__dirname, rel), 'utf8');
}

describe('base language merge seeds and wiring (P0)', () => {
  it('bootstrap.json has POSIX languages and no Locale entity', () => {
    const raw = read('../../data/bootstrap.json');
    const data = JSON.parse(raw) as { records: Array<{ external_id: string; model: string; values: Record<string, unknown> }> };

    expect(raw).not.toMatch(/base\.Locale|locale_default|DefaultLocaleId|LocaleId|language_zh[^_]|Code":\s*"zh"/);

    const byId = Object.fromEntries(data.records.map(r => [r.external_id, r]));
    expect(byId.language_en_us?.model).toBe('base.Language');
    expect(byId.language_en_us?.values.Code).toBe('en_US');
    expect(byId.language_en_us?.values.Grouping).toBe('[3,0]');

    expect(byId.language_zh_cn?.model).toBe('base.Language');
    expect(byId.language_zh_cn?.values.Code).toBe('zh_CN');
    expect(byId.language_zh_cn?.values.Grouping).toBe('[3,0]');
    expect(byId.language_zh_cn?.values.DecimalSeparator).toBe('.');

    expect(byId.company_main?.values.LanguageId).toEqual({ ref: 'base.language_zh_cn' });
    expect(byId.company_main?.values).not.toHaveProperty('LocaleId');
  });

  it('auth smoke fixture drops LocaleId and points at language_zh_cn', () => {
    const raw = read('../../../auth/e2e/fixtures/smoke.json');
    expect(raw).not.toMatch(/LocaleId|language_zh[^_]|locale_default/);
    expect(raw).toMatch(/base\.language_zh_cn/);
  });

  it('routes and menus have no Locale management surface', () => {
    const routes = read('../route/routes.ts');
    const menus = read('../menu/menus.ts');
    expect(routes).not.toMatch(/localeRoutes|\/base\/locales|LocaleList|LocaleForm/);
    expect(menus).not.toMatch(/base\.menu\.locale|\/base\/locales/);
  });

  it('company and language models have no Locale FK', () => {
    const company = read('../../service/models/company.ts');
    const language = read('../../service/models/language.ts');
    expect(company).not.toMatch(/LocaleId|from ['"].*locale['"]/);
    expect(language).not.toMatch(/DefaultLocaleId|from ['"].*locale['"]/);
    expect(language).toMatch(/Grouping/);
    expect(language).toMatch(/GetActiveLanguages/);
  });

  it('FE adapter symbols use UiKey names (no Locale product aliases)', () => {
    const index = read('../../../web/web/stores/i18nStore/index.ts');
    const lang = read('../../../web/web/stores/i18nStore/lang.ts');
    const utils = read('../../../web/web/stores/i18nStore/utils.ts');
    expect(lang).toMatch(/langToUiKey|uiKeyToLang/);
    expect(utils).toMatch(/detectBestUiKey/);
    expect(index).toMatch(/setActiveUiKeys|setUiKey|DEFAULT_ACTIVE_UI_KEYS/);
    expect(`${index}\n${lang}\n${utils}`).not.toMatch(
      /\blangToLocale\b|\blocaleToLang\b|\bdetectBestLocale\b|\bsetActiveLocales\b|\bDEFAULT_ACTIVE_LOCALES\b|\bsetLocale\b/
    );
  });
});

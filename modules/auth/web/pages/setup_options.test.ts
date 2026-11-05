// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { InitialSetupOptions } from '../stores/auth';
import { clearSetupDraft, loadSetupDraft, saveSetupDraft, SETUP_DRAFT_KEY } from './setup_draft';
import { buildSetupSelections, countryDefaultCurrencies, currencyLabel, resolveSetupInitialValues } from './setup_options';

function options(): InitialSetupOptions {
  return {
    Countries: [
      { Code: 'CN', Name: 'China', DefaultCurrencyCode: 'CNY' },
      { Code: 'AQ', Name: 'Antarctica' },
    ],
    Currencies: [
      { Code: 'CNY', Name: 'Chinese Yuan' },
      { Code: 'USD', Name: '' },
    ],
    Timezones: [
      { Value: 'Asia/Shanghai', Label: 'Asia/Shanghai' },
      { Value: 'Europe/Berlin', Label: 'Europe/Berlin' },
    ],
    Languages: [
      { Code: 'en_US', Name: 'English' },
      { Code: 'zh_CN', Name: '简体中文' },
    ],
    Defaults: {
      OrganizationName: 'My Company',
      CountryCode: 'CN',
      CurrencyCode: 'CNY',
      Timezone: 'Asia/Shanghai',
      LanguageCode: 'en_US',
    },
  };
}

function memoryStorage() {
  const data = new Map<string, string>();
  return {
    data,
    getItem: (k: string) => (data.has(k) ? data.get(k)! : null),
    setItem: (k: string, v: string) => {
      data.set(k, v);
    },
    removeItem: (k: string) => {
      data.delete(k);
    },
  };
}

test('buildSetupSelections labels currencies with code and name', () => {
  const s = buildSetupSelections(options());
  expect(s.CurrencyCode).toEqual([
    { value: 'CNY', label: 'CNY · Chinese Yuan' },
    { value: 'USD', label: 'USD' },
  ]);
  expect(s.CountryCode[0]).toEqual({ value: 'CN', label: 'China' });
  expect(s.OrgLanguageCode[1]).toEqual({ value: 'zh_CN', label: '简体中文' });
});

test('currencyLabel appends the symbol unless it repeats the code', () => {
  expect(currencyLabel({ Code: 'CNY', Name: '人民币', Symbol: '¥' })).toBe('CNY · 人民币 · ¥');
  expect(currencyLabel({ Code: 'AED', Name: 'UAE Dirham', Symbol: 'AED' })).toBe('AED · UAE Dirham');
  expect(currencyLabel({ Code: 'USD', Name: '', Symbol: '$' })).toBe('USD · $');
  expect(currencyLabel({ Code: 'XXX', Name: '' })).toBe('XXX');
});

test('countryDefaultCurrencies skips countries without a default', () => {
  expect(countryDefaultCurrencies(options())).toEqual({ CN: 'CNY' });
});

test('resolveSetupInitialValues: server defaults, browser timezone, UI language', () => {
  const values = resolveSetupInitialValues({
    options: options(),
    draft: null,
    uiLanguageCode: 'zh_CN',
    browserTimezone: 'Europe/Berlin',
  });
  expect(values).toEqual({
    Username: '',
    Password: '',
    ConfirmPassword: '',
    Email: '',
    OrgName: 'My Company',
    CountryCode: 'CN',
    CurrencyCode: 'CNY',
    Timezone: 'Europe/Berlin',
    OrgLanguageCode: 'zh_CN',
  });
  const fallback = resolveSetupInitialValues({
    options: options(),
    draft: null,
    uiLanguageCode: 'fr_FR',
    browserTimezone: 'Mars/Base',
  });
  expect(fallback.Timezone).toBe('Asia/Shanghai');
  expect(fallback.OrgLanguageCode).toBe('en_US');
});

test('resolveSetupInitialValues: draft wins and unknown selections are dropped', () => {
  const values = resolveSetupInitialValues({
    options: options(),
    draft: {
      values: {
        Username: 'owner',
        OrgName: 'Acme',
        CountryCode: '',
        CurrencyCode: 'USD',
        Timezone: 'Nowhere/Zone',
        OrgLanguageCode: 'zh_CN',
      },
    },
    uiLanguageCode: 'en_US',
    browserTimezone: '',
  });
  expect(values.Username).toBe('owner');
  expect(values.OrgName).toBe('Acme');
  expect(values.CountryCode).toBe(null);
  expect(values.CurrencyCode).toBe('USD');
  expect(values.Timezone).toBe('Asia/Shanghai');
  expect(values.OrgLanguageCode).toBe('zh_CN');
});

test('setup draft round-trips whitelisted fields and never stores passwords', () => {
  const storage = memoryStorage();
  saveSetupDraft(
    { Username: 'owner', Password: 'secret1', ConfirmPassword: 'secret1', OrgName: 'Acme', Extra: 'x' },
    storage,
  );
  const raw = storage.data.get(SETUP_DRAFT_KEY) || '';
  expect(raw.includes('secret1')).toBe(false);
  expect(raw.includes('Extra')).toBe(false);
  expect(loadSetupDraft(storage)).toEqual({ values: { Username: 'owner', OrgName: 'Acme' } });
  clearSetupDraft(storage);
  expect(loadSetupDraft(storage)).toBe(null);
});

test('loadSetupDraft ignores malformed data', () => {
  const storage = memoryStorage();
  storage.setItem(SETUP_DRAFT_KEY, '{not json');
  expect(loadSetupDraft(storage)).toBe(null);
  storage.setItem(SETUP_DRAFT_KEY, JSON.stringify({ values: { Username: 5, Email: 'a@b.io' } }));
  expect(loadSetupDraft(storage)).toEqual({ values: { Email: 'a@b.io' } });
  expect(loadSetupDraft(null)).toBe(null);
});

test('resolveSetupInitialValues: company without a country yields null CountryCode', () => {
  const opts = options();
  delete opts.Defaults.CountryCode;
  const values = resolveSetupInitialValues({ options: opts, draft: null, uiLanguageCode: '', browserTimezone: '' });
  expect(values.CountryCode).toBe(null);
});

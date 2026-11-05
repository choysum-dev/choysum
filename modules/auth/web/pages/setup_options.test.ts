// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { InitialSetupOptions } from '../stores/auth';
import { clearSetupDraft, loadSetupDraft, saveSetupDraft, SETUP_DRAFT_KEY } from './setup_draft';
import {
  browserTimezone,
  buildSetupSelections,
  countryDefaultCurrencies,
  createCurrencyFollower,
  currencyLabel,
  resolveAdminLanguageCode,
  resolveSetupInitialValues,
} from './setup_options';

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
  expect('OrgLanguageCode' in s).toBe(false);
});

test('currencyLabel appends the symbol unless it repeats the code', () => {
  expect(currencyLabel({ Code: 'CNY', Name: '人民币', Symbol: '¥' })).toBe('CNY · 人民币 · ¥');
  expect(currencyLabel({ Code: 'AED', Name: 'UAE Dirham', Symbol: 'AED' })).toBe('AED · UAE Dirham');
  expect(currencyLabel({ Code: 'USD', Name: '', Symbol: '$' })).toBe('USD · $');
  expect(currencyLabel({ Code: 'XXX', Name: '' })).toBe('XXX');
});

test('resolveAdminLanguageCode: UI language when listed, else en_US, else first', () => {
  expect(resolveAdminLanguageCode(options(), 'zh_CN')).toBe('zh_CN');
  expect(resolveAdminLanguageCode(options(), 'fr_FR')).toBe('en_US');
  const zhOnly = { ...options(), Languages: [{ Code: 'zh_CN', Name: '简体中文' }] };
  expect(resolveAdminLanguageCode(zhOnly, 'fr_FR')).toBe('zh_CN');
  expect(resolveAdminLanguageCode(null, 'fr_FR')).toBe('fr_FR');
});

test('countryDefaultCurrencies skips countries without a default', () => {
  expect(countryDefaultCurrencies(options())).toEqual({ CN: 'CNY' });
});

test('resolveSetupInitialValues: server defaults and browser timezone', () => {
  const values = resolveSetupInitialValues({
    options: options(),
    draft: null,
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
  });
  const fallback = resolveSetupInitialValues({
    options: options(),
    draft: null,
    browserTimezone: 'Mars/Base',
  });
  expect(fallback.Timezone).toBe('Asia/Shanghai');
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
      },
    },
    browserTimezone: '',
  });
  expect(values.Username).toBe('owner');
  expect(values.OrgName).toBe('Acme');
  expect(values.CountryCode).toBe(null);
  expect(values.CurrencyCode).toBe('USD');
  expect(values.Timezone).toBe('Asia/Shanghai');
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
  const values = resolveSetupInitialValues({ options: opts, draft: null, browserTimezone: '' });
  expect(values.CountryCode).toBe(null);
});

function follower(touched = false) {
  const formData: Record<string, unknown> = { CountryCode: null, CurrencyCode: 'USD' };
  const setField: Array<[string, string]> = [];
  const state = { touched, touchedEvents: 0 };
  const f = createCurrencyFollower({
    formData,
    setField: (prop, value) => setField.push([prop, value]),
    countryCurrencies: () => ({ CN: 'CNY', DE: 'EUR' }),
    currencyTouched: () => state.touched,
    onTouched: () => {
      state.touchedEvents++;
      state.touched = true;
    },
  });
  return { f, formData, setField, state };
}

test('createCurrencyFollower: country fills its default currency without marking touched', () => {
  const { f, formData, setField, state } = follower();
  f.onCountryChange('CN', null);
  expect(formData.CurrencyCode).toBe('CNY');
  expect(setField).toEqual([['CurrencyCode', 'CNY']]);
  f.onCurrencyChange('CNY', 'USD');
  expect(state.touchedEvents).toBe(0);
});

test('createCurrencyFollower: unchanged, unknown or matching country leaves currency alone', () => {
  const { f, formData, setField } = follower();
  f.onCountryChange('CN', 'CN');
  f.onCountryChange('AQ', null);
  f.onCountryChange(null, 'CN');
  formData.CurrencyCode = 'EUR';
  f.onCountryChange('DE', null);
  expect(formData.CurrencyCode).toBe('EUR');
  expect(setField).toEqual([]);
});

test('createCurrencyFollower: manual currency marks touched once and stops following', () => {
  const { f, formData, setField, state } = follower();
  f.onCurrencyChange('USD', 'USD');
  expect(state.touchedEvents).toBe(0);
  f.onCurrencyChange('EUR', 'USD');
  expect(state.touchedEvents).toBe(1);
  f.onCurrencyChange('USD', 'EUR');
  expect(state.touchedEvents).toBe(1);
  f.onCountryChange('CN', null);
  expect(formData.CurrencyCode).toBe('USD');
  expect(setField).toEqual([]);
});

test('createCurrencyFollower: a manual pick that differs from the auto value marks touched', () => {
  const { f, state } = follower();
  f.onCountryChange('CN', null);
  f.onCurrencyChange('EUR', 'USD');
  expect(state.touchedEvents).toBe(1);
});

test('browserTimezone returns empty when Intl throws', () => {
  const g = globalThis as { Intl: unknown };
  const saved = g.Intl;
  g.Intl = {
    DateTimeFormat: () => {
      throw new Error('no Intl');
    },
  };
  try {
    expect(browserTimezone()).toBe('');
  } finally {
    g.Intl = saved;
  }
});

test('saveSetupDraft stores null fields as empty strings', () => {
  const storage = memoryStorage();
  saveSetupDraft({ CountryCode: null, OrgName: 'Acme' }, storage);
  expect(loadSetupDraft(storage)?.values).toEqual({ OrgName: 'Acme', CountryCode: '' });
});

test('setup draft is a no-op when sessionStorage access throws', () => {
  const desc = Object.getOwnPropertyDescriptor(globalThis, 'sessionStorage');
  Object.defineProperty(globalThis, 'sessionStorage', {
    configurable: true,
    get() {
      throw new Error('denied');
    },
  });
  try {
    saveSetupDraft({ OrgName: 'Acme' });
    expect(loadSetupDraft()).toBe(null);
    clearSetupDraft();
  } finally {
    if (desc) Object.defineProperty(globalThis, 'sessionStorage', desc);
    else delete (globalThis as { sessionStorage?: unknown }).sessionStorage;
  }
});

// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import Company from '@/base/service/models/company';
import Country from '@/base/service/models/country';
import Currency from '@/base/service/models/currency';
import { listSetupTimezones } from '@/base/service/models/_company_initial_setup';
import { ChoysumError } from '@/core/service/error';
import { withContext } from '@/core/service/api/context';
import { companyCode8, countryCode8, currencyCode3, uid } from './_helpers';

async function expectFieldError(fn: () => Promise<unknown>, field: string): Promise<void> {
  let caught: unknown;
  try {
    await fn();
  } catch (err) {
    caught = err;
  }
  expect(caught instanceof ChoysumError).toBe(true);
  const err = caught as ChoysumError;
  expect(err.domain).toBe('base');
  expect(err.metadata?.Field).toBe(field);
}

async function createTargetCompany(): Promise<string> {
  const [cny] = await Currency.Search(['Code', '=', 'CNY'] as any, { fields: ['Id'], limit: 1 } as any);
  const created = await Company.Create(
    {
      Name: { en_US: uid('SetupTargetEn'), zh_CN: uid('SetupTargetZh') },
      Code: companyCode8(),
      Timezone: 'Asia/Shanghai',
      CurrencyId: (cny as any).Id,
      IsActive: true,
    } as any,
    ['Id'] as any
  );
  return String((created as any).Id);
}

async function createInactiveCurrency(): Promise<{ id: string; code: string }> {
  const code = currencyCode3();
  const created = await Currency.Create(
    { Name: uid('SetupCur'), Code: code, DecimalDigits: 0, Rounding: '1', IsActive: false } as any,
    ['Id'] as any
  );
  return { id: String((created as any).Id), code };
}

test('GetInitialSetupOptions: fixed shape, inactive currencies listed, inactive countries hidden', async () => {
  const hiddenCountryCode = countryCode8();
  await Country.Create(
    { Name: uid('HiddenCountry'), Code: hiddenCountryCode, ZipRequired: false, StateRequired: false, IsActive: false } as any,
    ['Id'] as any
  );

  const resp = await Company.GetInitialSetupOptions({});
  expect(Object.keys(resp).sort()).toEqual(['Countries', 'Currencies', 'Defaults', 'Languages', 'Timezones']);

  const usd = resp.Currencies.find(c => c.Code === 'USD');
  expect(Boolean(usd)).toBe(true);
  expect(Object.keys(usd!).sort()).toEqual(['Code', 'Name', 'Symbol']);
  expect(resp.Currencies.length > 150).toBe(true);

  expect(resp.Countries.length >= 249).toBe(true);
  expect(resp.Countries.some(c => c.Code === hiddenCountryCode)).toBe(false);
  const us = resp.Countries.find(c => c.Code === 'US');
  expect(us?.DefaultCurrencyCode).toBe('USD');
  expect(resp.Countries.find(c => c.Code === 'AQ')?.DefaultCurrencyCode).toBeUndefined();

  expect(resp.Languages.map(l => l.Code).sort()).toEqual(['en_US', 'zh_CN']);
  expect(resp.Timezones.some(t => t.Value === 'Asia/Shanghai' && t.Label === '(UTC+08:00) Asia/Shanghai')).toBe(true);

  expect(resp.Defaults.CurrencyCode).toBe('CNY');
  expect(resp.Defaults.Timezone).toBe('Asia/Shanghai');
  expect(resp.Defaults.LanguageCode).toBe('zh_CN');
  expect(resp.Defaults.OrganizationName === '').toBe(false);
});

test('listSetupTimezones: offset labels follow DST and sort by offset then name', () => {
  const winter = listSetupTimezones(new Date('2026-01-15T12:00:00Z'));
  const summer = listSetupTimezones(new Date('2026-07-15T12:00:00Z'));
  expect(winter.find(z => z.Value === 'America/New_York')?.Label).toBe('(UTC-05:00) America/New_York');
  expect(summer.find(z => z.Value === 'America/New_York')?.Label).toBe('(UTC-04:00) America/New_York');
  expect(winter.find(z => z.Value === 'Asia/Kolkata')?.Label).toBe('(UTC+05:30) Asia/Kolkata');
  expect(winter.find(z => z.Value === 'UTC')?.Label).toBe('(UTC+00:00) UTC');
  const idx = (v: string) => winter.findIndex(z => z.Value === v);
  expect(idx('America/New_York') < idx('UTC')).toBe(true);
  expect(idx('UTC') < idx('Asia/Kolkata')).toBe(true);
  expect(idx('Asia/Kolkata') < idx('Asia/Shanghai')).toBe(true);
  expect(idx('Asia/Shanghai') < idx('Asia/Singapore')).toBe(true);
});

test('GetInitialSetupOptions: names follow request language', async () => {
  const zh = await withContext({ lang: 'zh_CN' }, () => Company.GetInitialSetupOptions({}));
  const en = await withContext({ lang: 'en_US' }, () => Company.GetInitialSetupOptions({}));
  expect(zh.Countries.find(c => c.Code === 'CN')?.Name).toBe('中国');
  expect(en.Countries.find(c => c.Code === 'CN')?.Name).toBe('China');
  expect(zh.Currencies.find(c => c.Code === 'USD')?.Name).toBe('美元');
  expect(en.Currencies.find(c => c.Code === 'USD')?.Name).toBe('US Dollar');
});

test('ApplyInitialSetup: activates currency, replaces name translations, sets org fields', async () => {
  const companyId = await createTargetCompany();
  const currency = await createInactiveCurrency();

  const req = {
    CompanyId: companyId,
    Name: '  Acme Ltd  ',
    CountryCode: 'us',
    CurrencyCode: currency.code.toLowerCase(),
    Timezone: 'America/New_York',
  };
  const before = await Company.Browse(companyId, ['Id', 'LanguageId'] as any);
  const resp = await Company.ApplyInitialSetup(req);
  expect(resp.CompanyId).toBe(companyId);

  const cur = await Currency.Browse(currency.id, ['Id', 'IsActive'] as any);
  expect((cur as any).IsActive).toBe(true);

  const zhRow = await withContext({ lang: 'zh_CN' }, () =>
    Company.Browse(companyId, ['Id', 'Name', 'Timezone', 'CurrencyId', 'LanguageId', 'CountryId'] as any)
  );
  // zh_CN translation was replaced, so zh reads fall back to en_US.
  expect(String((zhRow as any).Name)).toBe('Acme Ltd');
  expect(String((zhRow as any).Timezone)).toBe('America/New_York');
  const idOf = (v: any) => String(v?.Id ?? v ?? '');
  expect(idOf((zhRow as any).CurrencyId)).toBe(currency.id);
  const [us] = await Country.Search(['Code', '=', 'US'] as any, { fields: ['Id'], limit: 1 } as any);
  expect(idOf((zhRow as any).CountryId)).toBe(String((us as any).Id));
  expect(idOf((zhRow as any).LanguageId)).toBe(idOf((before as any).LanguageId));

  // Idempotent for identical input.
  const again = await Company.ApplyInitialSetup(req);
  expect(again.CompanyId).toBe(companyId);

  // Clearing the optional country writes null.
  await Company.ApplyInitialSetup({ ...req, CountryCode: undefined });
  const cleared = await Company.Browse(companyId, ['Id', 'CountryId'] as any);
  expect(idOf((cleared as any).CountryId)).toBe('');
});

test('ApplyInitialSetup: rejects invalid input with field metadata', async () => {
  const companyId = await createTargetCompany();
  const base = {
    CompanyId: companyId,
    Name: 'Acme',
    CurrencyCode: 'CNY',
    Timezone: 'Asia/Shanghai',
  };
  await expectFieldError(() => Company.ApplyInitialSetup({ ...base, Name: '   ' }), 'Name');
  await expectFieldError(() => Company.ApplyInitialSetup({ ...base, Name: 'x'.repeat(101) }), 'Name');
  await expectFieldError(() => Company.ApplyInitialSetup({ ...base, CurrencyCode: 'Q0Q' }), 'CurrencyCode');
  await expectFieldError(() => Company.ApplyInitialSetup({ ...base, CountryCode: 'Q0' }), 'CountryCode');
  await expectFieldError(() => Company.ApplyInitialSetup({ ...base, Timezone: 'Mars/Base' }), 'Timezone');
  await expectFieldError(() => Company.ApplyInitialSetup({ ...base, CompanyId: '' }), 'CompanyId');
});

test('ApplyInitialSetup: inactive country is rejected', async () => {
  const companyId = await createTargetCompany();
  const code = countryCode8();
  await Country.Create(
    { Name: uid('InactiveCountry'), Code: code, ZipRequired: false, StateRequired: false, IsActive: false } as any,
    ['Id'] as any
  );
  await expectFieldError(
    () =>
      Company.ApplyInitialSetup({
        CompanyId: companyId,
        Name: 'Acme',
        CountryCode: code,
        CurrencyCode: 'CNY',
        Timezone: 'Asia/Shanghai',
      }),
    'CountryCode'
  );
});

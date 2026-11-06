// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import Country from '@/base/service/models/country';
import Currency from '@/base/service/models/currency';
import bootstrap from '../../data/bootstrap.json';
import currencies from '../../data/currencies.json';
import countries from '../../data/countries.json';

type SeedRecord = { name: string; model: string; noupdate?: boolean; values: Record<string, any> };

const currencyRecords = (currencies as { records: SeedRecord[] }).records;
const countryRecords = (countries as { records: SeedRecord[] }).records;
const bootstrapRecords = (bootstrap as { records: SeedRecord[] }).records;

function expectedRounding(digits: number): string {
  return digits === 0 ? '1' : `0.${'0'.repeat(digits - 1)}1`;
}

test('seed currencies: ISO codes, xml names, rounding and inactive default', () => {
  expect(currencyRecords.length > 150).toBe(true);
  const codes = new Set<string>();
  for (const rec of currencyRecords) {
    const v = rec.values;
    expect(rec.model).toBe('Currency');
    // Upgrades must not deactivate a currency chosen during setup.
    expect(rec.noupdate).toBe(true);
    expect(/^[A-Z]{3}$/.test(v.Code)).toBe(true);
    expect(rec.name).toBe(`currency_${String(v.Code).toLowerCase()}`);
    expect(codes.has(v.Code)).toBe(false);
    codes.add(v.Code);
    expect(Number.isInteger(v.DecimalDigits) && v.DecimalDigits >= 0 && v.DecimalDigits <= 4).toBe(true);
    expect(v.Rounding).toBe(expectedRounding(v.DecimalDigits));
    expect(v.IsActive).toBe(false);
    expect(String(v.Name?.en_US || '').trim() === '').toBe(false);
    expect(String(v.Name?.zh_CN || '').trim() === '').toBe(false);
  }
  expect(codes.has('CNY')).toBe(false);
  expect(codes.has('USD')).toBe(true);
  expect(codes.has('XXX')).toBe(false);
});

test('seed countries: ISO 3166-1 alpha-2 and resolvable default currency refs', () => {
  expect(countryRecords.length).toBe(249);
  const currencyXmlIds = new Set<string>(
    [...currencyRecords, ...bootstrapRecords.filter(r => r.model === 'Currency')].map(r => `base.${r.name}`)
  );
  const codes = new Set<string>();
  for (const rec of countryRecords) {
    const v = rec.values;
    expect(rec.model).toBe('Country');
    expect(rec.noupdate).toBe(true);
    expect(/^[A-Z]{2}$/.test(v.Code)).toBe(true);
    expect(rec.name).toBe(`country_${String(v.Code).toLowerCase()}`);
    expect(codes.has(v.Code)).toBe(false);
    codes.add(v.Code);
    expect(v.IsActive).toBe(true);
    if (v.PhonePrefix !== undefined) {
      expect(/^\+\d{1,4}$/.test(v.PhonePrefix)).toBe(true);
    }
    if (v.DefaultCurrencyId !== undefined) {
      expect(currencyXmlIds.has(String(v.DefaultCurrencyId.ref))).toBe(true);
    }
  }
});

test('seed data installed: USD inactive, CNY active, CN defaults to CNY', async () => {
  const [usd] = await Currency.Search({ And: [['Code', '=', 'USD']] } as any, {
    fields: ['Id', 'IsActive', 'DecimalDigits'],
  } as any);
  expect(Boolean(usd)).toBe(true);
  expect((usd as any).IsActive).toBe(false);
  expect(Number((usd as any).DecimalDigits)).toBe(2);

  const [cny] = await Currency.Search({ And: [['Code', '=', 'CNY']] } as any, { fields: ['Id', 'IsActive'] } as any);
  expect((cny as any).IsActive).toBe(true);

  const [cn] = await Country.Search({ And: [['Code', '=', 'CN']] } as any, {
    fields: ['Id', 'DefaultCurrencyId'],
  } as any);
  const cur = (cn as any)?.DefaultCurrencyId;
  expect(String(cur?.Id ?? cur ?? '')).toBe(String((cny as any).Id));
});

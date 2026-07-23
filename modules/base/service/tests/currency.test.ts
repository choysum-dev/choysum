// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import Currency from '@/base/service/models/currency';
import Company from '@/base/service/models/company';
import ExchangeRate from '@/base/service/models/exchange_rate';
import { ChoysumError } from '@/core/service/error';
import { withContext as withModelContext, withContext } from '@/core/service/api/context';
import { resolveValidationSummary } from '@/core/service/api/validation';
import { MetadataStorage } from '@/core/service/api/metadata';

import { expectBaseInvalidArgument, uid, currencyCode3 } from './_helpers';
import { expectBaseNotFound } from './_helpers';

async function expectRepoValidationFailed(mode: 'create' | 'update', fn: () => Promise<void>): Promise<void> {
  let error: unknown;
  try {
    await fn();
  } catch (err) {
    error = err;
  }

  expect(error instanceof ChoysumError).toBe(true);
  const oe = error as ChoysumError;
  expect(oe.domain).toBe('core.repository');
  expect(oe.code).toBe('validation_failed');
  expect(oe.metadata?.mode).toBe(mode);
  const summary = resolveValidationSummary(oe);
  const codes = summary.issues.map(item => String(item?.code || ''));
  expect(codes.some(code => code === 'constraint_execution_failed' || code.startsWith('kernel_') || code.startsWith('sql_'))).toBe(true);
}

test('base.currency: code normalized to upper + trim', async () => {
  const created = await Currency.Create(
    {
      Name: uid('USD Currency'),
      Code: ' usD ',
      DecimalDigits: 2,
      Rounding: '0.01',
      IsActive: true,
    } as any,
    ['Id', 'Code'] as any
  );

  expect(String((created as any).Code)).toBe('USD');

  await expectRepoValidationFailed('update', async () => {
    await Currency.UpdateById(
      String((created as any).Id),
      {
        DecimalDigits: -1,
      } as any,
      ['Id'] as any
    );
  });
});

test('base.currency: Convert validates enum modes', async () => {
  const from = await Currency.Create(
    {
      Name: uid('From Currency'),
      Code: 'fmc',
      DecimalDigits: 2,
      Rounding: '0.01',
      IsActive: true,
    } as any,
    ['Id'] as any
  );

  const to = await Currency.Create(
    {
      Name: uid('To Currency'),
      Code: 'tmc',
      DecimalDigits: 2,
      Rounding: '0.01',
      IsActive: true,
    } as any,
    ['Id'] as any
  );

  const company = await Company.Create(
    {
      Name: uid('Convert Company'),
      Code: uid('CONVCO').slice(-8).toUpperCase(),
      Timezone: 'UTC',
      CurrencyId: (from as any).Id,
      IsActive: true,
    } as any,
    ['Id'] as any
  );

  await expectBaseInvalidArgument(async () => {
    await Currency.Convert({
      CompanyId: String((company as any).Id),
      Date: '2026-02-25',
      Amount: '1',
      FromCurrencyId: String((from as any).Id),
      ToCurrencyId: String((to as any).Id),
      RatePolicy: { Mode: 'bad_mode' as any },
    } as any);
  });

  await expectBaseInvalidArgument(async () => {
    await Currency.Convert({
      CompanyId: String((company as any).Id),
      Date: '2026-02-25',
      Amount: '1',
      FromCurrencyId: String((from as any).Id),
      ToCurrencyId: String((to as any).Id),
      Rounding: { Mode: 'bad_mode' as any },
    } as any);
  });

  await expectBaseInvalidArgument(async () => {
    await Currency.Convert({
      CompanyId: String((company as any).Id),
      Date: '2026-02-25',
      Amount: 1,
      FromCurrencyId: String((from as any).Id),
      ToCurrencyId: String((from as any).Id),
    } as any);
  });
});

test('base.currency: Convert falls back to global ExchangeRate by default and prefers company-specific', async () => {
  const companyCurrency = await Currency.Create(
    {
      Name: uid('CompanyCurrency'),
      Code: 'ccc',
      DecimalDigits: 2,
      Rounding: '0.01',
      IsActive: true,
    } as any,
    ['Id'] as any
  );

  const fromCurrency = await Currency.Create(
    {
      Name: uid('FromCurrency'),
      Code: 'ffc',
      DecimalDigits: 2,
      Rounding: '0.01',
      IsActive: true,
    } as any,
    ['Id'] as any
  );

  const company = await Company.Create(
    {
      Name: uid('Fx Company'),
      Code: uid('FXCO').slice(-8).toUpperCase(),
      Timezone: 'UTC',
      CurrencyId: (companyCurrency as any).Id,
      IsActive: true,
    } as any,
    ['Id'] as any
  );

  const companyId = String((company as any).Id);
  const date = '2026-02-01';

  await withModelContext(
    {
      activeCompanyId: companyId,
      enabledCompanyIds: [companyId],
    } as any,
    async () => {
      // Global fallback rate: 1 * From = 2 * Company
      await ExchangeRate.Create(
        {
          CompanyId: null,
          CurrencyId: (fromCurrency as any).Id,
          Date: date,
          Rate: '2',
        } as any,
        ['Id'] as any
      );

      const r1 = await Currency.Convert({
        CompanyId: companyId,
        Date: date,
        Amount: '10',
        FromCurrencyId: String((fromCurrency as any).Id),
        ToCurrencyId: String((companyCurrency as any).Id),
      } as any);
      expect(String((r1 as any).Amount)).toBe('20');
      expect(Array.isArray((r1 as any).Warnings)).toBe(true);
      expect(((r1 as any).Warnings || []).includes('rate.global.fallback')).toBe(true);

      // Company-specific rate should override global: 1 * From = 3 * Company
      await ExchangeRate.Create(
        {
          CompanyId: companyId,
          CurrencyId: (fromCurrency as any).Id,
          Date: date,
          Rate: '3',
        } as any,
        ['Id'] as any
      );

      const r2 = await Currency.Convert({
        CompanyId: companyId,
        Date: date,
        Amount: '10',
        FromCurrencyId: String((fromCurrency as any).Id),
        ToCurrencyId: String((companyCurrency as any).Id),
      } as any);
      expect(String((r2 as any).Amount)).toBe('30');
      expect(((r2 as any).Warnings || []).includes('rate.global.fallback')).toBe(false);
    },
    { merge: false }
  );
});

test('base.currency: Convert returns NotFound when CompanyId does not exist', async () => {
  const from = await Currency.Create(
    {
      Name: uid('From Currency NF'),
      Code: 'nfa',
      DecimalDigits: 2,
      Rounding: '0.01',
      IsActive: true,
    } as any,
    ['Id'] as any
  );

  const to = await Currency.Create(
    {
      Name: uid('To Currency NF'),
      Code: 'nfb',
      DecimalDigits: 2,
      Rounding: '0.01',
      IsActive: true,
    } as any,
    ['Id'] as any
  );

  await expectBaseNotFound(async () => {
    await Currency.Convert({
      CompanyId: 'NO_SUCH_COMPANY',
      Date: '2026-02-25',
      Amount: '1',
      FromCurrencyId: String((from as any).Id),
      ToCurrencyId: String((to as any).Id),
    } as any);
  });
});

test('base.currency: Name translate metadata is enabled', () => {
  const field = MetadataStorage.instance.getModelMetadata(Currency).fields.get('Name');
  expect(field?.translate).toBe(true);
  expect(field?.type).toBe('varchar');
  expect(field?.column?.size).toBeUndefined();
  expect(field?.column?.index).toBe('trigram');
  expect(field?.storageHints?.size).toBe(100);
});

test('base.currency: Name bilingual write/read unwraps by lang', async () => {
  const code = currencyCode3();
  const enName = uid('CurrencyEn');
  const zhName = uid('CurrencyZh');

  const created = await Currency.Create(
    {
      Name: { en_US: enName, zh_CN: zhName } as any,
      Code: code,
      DecimalDigits: 2,
      Rounding: '0.01',
      IsActive: true,
    } as any,
    ['Id', 'Name', 'Code'] as any
  );
  expect(String((created as any).Name)).toBe(enName);

  const id = String((created as any).Id);
  const zhBrowse = await withContext({ lang: 'zh_CN' }, () => Currency.Browse(id, ['Id', 'Name'] as any));
  expect(String((zhBrowse as any).Name)).toBe(zhName);

  const hit = await withContext({ lang: 'zh_CN' }, () =>
    Currency.Search(['Name', 'ilike', zhName] as any, { fields: ['Id', 'Code'], limit: 5 } as any)
  );
  expect(hit?.some((r: any) => String(r.Code) === String((created as any).Code))).toBe(true);
});

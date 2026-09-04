// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ChoysumError } from '@/core/service/error';
import { withContext as withModelContext } from '@/core/service/api/context';
import { resolveValidationSummary } from '@/core/service/api/validation';

import Company from '@/base/service/models/company';
import Currency from '@/base/service/models/currency';
import ExchangeRate from '@/base/service/models/exchange_rate';

import { companyCode8, currencyCode3, uid } from './_helpers';

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

test('base.exchange_rate: validates DateString and positive Rate', async () => {
  const companyCurrency = await Currency.Create(
    {
      Name: uid('MainCurrency'),
      Code: currencyCode3(),
      DecimalDigits: 2,
      Rounding: '0.01',
      IsActive: true,
    } as any,
    ['Id'] as any
  );

  const rateCurrency = await Currency.Create(
    {
      Name: uid('RateCurrency'),
      Code: currencyCode3(),
      DecimalDigits: 2,
      Rounding: '0.01',
      IsActive: true,
    } as any,
    ['Id'] as any
  );

  const company = await Company.Create(
    {
      Name: uid('RateCompany'),
      Code: companyCode8(),
      Timezone: 'UTC',
      CurrencyId: (companyCurrency as any).Id,
      IsActive: true,
    } as any,
    ['Id'] as any
  );

  await withModelContext(
    {
      activeCompanyId: String((company as any).Id),
      enabledCompanyIds: [String((company as any).Id)],
    } as any,
    async () => {
      await expectRepoValidationFailed('create', async () => {
        await ExchangeRate.Create(
          {
            CompanyId: (company as any).Id,
            CurrencyId: (rateCurrency as any).Id,
            Date: '2026-01-01T00:00:00Z',
            Rate: '1.23',
          } as any,
          ['Id'] as any
        );
      });

      await expectRepoValidationFailed('create', async () => {
        await ExchangeRate.Create(
          {
            CompanyId: (company as any).Id,
            CurrencyId: (rateCurrency as any).Id,
            Date: '2026-01-01',
            Rate: '0',
          } as any,
          ['Id'] as any
        );
      });

      let missingCurrency: unknown;
      try {
        await (ExchangeRate as any).ensureUniqueTuple(
          {
            CompanyId: (company as any).Id,
            CompanyScopeKey: String((company as any).Id),
            CurrencyId: '',
            Date: '2026-01-02',
          },
          undefined
        );
      } catch (err) {
        missingCurrency = err;
      }
      expect(missingCurrency instanceof ChoysumError).toBe(true);
      expect(String((missingCurrency as ChoysumError).message || '')).toMatch(/CurrencyId/);

      let missingCurrencyNull: unknown;
      try {
        await (ExchangeRate as any).ensureUniqueTuple(
          {
            CompanyId: (company as any).Id,
            CompanyScopeKey: String((company as any).Id),
            CurrencyId: null,
            Date: '2026-01-02',
          },
          undefined
        );
      } catch (err) {
        missingCurrencyNull = err;
      }
      expect(missingCurrencyNull instanceof ChoysumError).toBe(true);

      let missingCurrencyNumber: unknown;
      try {
        await (ExchangeRate as any).ensureUniqueTuple(
          {
            CompanyId: (company as any).Id,
            CompanyScopeKey: String((company as any).Id),
            CurrencyId: 0,
            Date: '2026-01-02',
          },
          undefined
        );
      } catch (err) {
        missingCurrencyNumber = err;
      }
      expect(missingCurrencyNumber instanceof ChoysumError).toBe(true);

      let missingCurrencyObj: unknown;
      try {
        await (ExchangeRate as any).ensureUniqueTuple(
          {
            CompanyId: (company as any).Id,
            CompanyScopeKey: String((company as any).Id),
            CurrencyId: { Id: '' },
            Date: '2026-01-02',
          },
          undefined
        );
      } catch (err) {
        missingCurrencyObj = err;
      }
      expect(missingCurrencyObj instanceof ChoysumError).toBe(true);

      let missingCurrencyObjNullId: unknown;
      try {
        await (ExchangeRate as any).ensureUniqueTuple(
          {
            CompanyId: (company as any).Id,
            CompanyScopeKey: String((company as any).Id),
            CurrencyId: { Id: null },
            Date: '2026-01-02',
          },
          undefined
        );
      } catch (err) {
        missingCurrencyObjNullId = err;
      }
      expect(missingCurrencyObjNullId instanceof ChoysumError).toBe(true);

      await (ExchangeRate as any).ensureUniqueTuple(
        {
          CompanyId: (company as any).Id,
          CompanyScopeKey: String((company as any).Id),
          CurrencyId: (rateCurrency as any).Id,
          Date: '2026-01-10',
        },
        undefined
      );

      await (ExchangeRate as any).ensureUniqueTuple(
        {
          CompanyId: (company as any).Id,
          CompanyScopeKey: String((company as any).Id),
          CurrencyId: { Id: (rateCurrency as any).Id },
          Date: '2026-01-11',
        },
        undefined
      );

      await (ExchangeRate as any).ensureUniqueTuple(
        {
          CompanyId: (company as any).Id,
          CompanyScopeKey: String((company as any).Id),
          CurrencyId: { id: (rateCurrency as any).Id },
          Date: '2026-01-12',
        },
        undefined
      );

      const created = await ExchangeRate.Create(
        {
          CompanyId: (company as any).Id,
          CurrencyId: (rateCurrency as any).Id,
          Date: '2026-01-02',
          Rate: '7.1234',
        } as any,
        ['Id'] as any
      );
      expect(Boolean((created as any).Id)).toBe(true);

      await expectRepoValidationFailed('create', async () => {
        await ExchangeRate.Create(
          {
            CompanyId: (company as any).Id,
            CurrencyId: (rateCurrency as any).Id,
            Date: '2026-01-03',
            Rate: { $bigdecimal: '0' },
          } as any,
          ['Id'] as any
        );
      });

      const createdEnvelope = await ExchangeRate.Create(
        {
          CompanyId: (company as any).Id,
          CurrencyId: (rateCurrency as any).Id,
          Date: '2026-01-04',
          Rate: { $bigdecimal: '8.5' },
        } as any,
        ['Id'] as any
      );
      expect(Boolean((createdEnvelope as any).Id)).toBe(true);
    },
    { merge: false }
  );
});

test('base.exchange_rate: Date defaults to businessToday(companyTz) independent of user tz', async () => {
  const companyCurrency = await Currency.Create(
    {
      Name: uid('BizDayCurrency'),
      Code: currencyCode3(),
      DecimalDigits: 2,
      Rounding: '0.01',
      IsActive: true,
    } as any,
    ['Id'] as any
  );

  const rateCurrency = await Currency.Create(
    {
      Name: uid('BizDayRateCurrency'),
      Code: currencyCode3(),
      DecimalDigits: 2,
      Rounding: '0.01',
      IsActive: true,
    } as any,
    ['Id'] as any
  );

  const company = await Company.Create(
    {
      Name: uid('BizDayCompany'),
      Code: companyCode8(),
      Timezone: 'Asia/Shanghai',
      CurrencyId: (companyCurrency as any).Id,
      IsActive: true,
    } as any,
    ['Id'] as any
  );

  // 2024-07-01 16:30Z is still 2024-07-01 in New York, but 2024-07-02 00:30 in Shanghai.
  const fixedNow = new Date('2024-07-01T16:30:00.000Z');
  const { businessToday } = await import('@/core/service/utils/datetime');

  await withModelContext(
    {
      activeCompanyId: String((company as any).Id),
      enabledCompanyIds: [String((company as any).Id)],
      companyTz: 'Asia/Shanghai',
      tz: 'America/New_York',
    } as any,
    async () => {
      expect(businessToday(undefined, fixedNow)).toBe('2024-07-02');
      expect(businessToday('America/New_York', fixedNow)).toBe('2024-07-01');

      const created = await ExchangeRate.Create(
        {
          CompanyId: (company as any).Id,
          CurrencyId: (rateCurrency as any).Id,
          Date: businessToday(undefined, fixedNow),
          Rate: '1.01',
        } as any,
        ['Id', 'Date'] as any
      );
      expect(String((created as any).Date).slice(0, 10)).toBe('2024-07-02');
    },
    { merge: false }
  );
});

test('base.exchange_rate: Date objects are rejected (no UTC calendar-day coercion)', async () => {
  const companyCurrency = await Currency.Create(
    {
      Name: uid('DateObjCurrency'),
      Code: currencyCode3(),
      DecimalDigits: 2,
      Rounding: '0.01',
      IsActive: true,
    } as any,
    ['Id'] as any
  );

  const rateCurrency = await Currency.Create(
    {
      Name: uid('DateObjRateCurrency'),
      Code: currencyCode3(),
      DecimalDigits: 2,
      Rounding: '0.01',
      IsActive: true,
    } as any,
    ['Id'] as any
  );

  const company = await Company.Create(
    {
      Name: uid('DateObjCompany'),
      Code: companyCode8(),
      Timezone: 'Asia/Shanghai',
      CurrencyId: (companyCurrency as any).Id,
      IsActive: true,
    } as any,
    ['Id'] as any
  );

  // Shanghai local midnight 2024-07-01 is still 2024-06-30 in UTC; Date→toISOString would shift the day.
  const shanghaiMidnightAsUtcDate = new Date('2024-06-30T16:00:00.000Z');

  await withModelContext(
    {
      activeCompanyId: String((company as any).Id),
      enabledCompanyIds: [String((company as any).Id)],
      companyTz: 'Asia/Shanghai',
    } as any,
    async () => {
      await expectRepoValidationFailed('create', async () => {
        await ExchangeRate.Create(
          {
            CompanyId: (company as any).Id,
            CurrencyId: (rateCurrency as any).Id,
            Date: shanghaiMidnightAsUtcDate,
            Rate: '1.23',
          } as any,
          ['Id'] as any
        );
      });

      const created = await ExchangeRate.Create(
        {
          CompanyId: (company as any).Id,
          CurrencyId: (rateCurrency as any).Id,
          Date: '2024-07-01',
          Rate: '1.23',
        } as any,
        ['Id', 'Date'] as any
      );
      expect(String((created as any).Date).slice(0, 10)).toBe('2024-07-01');
    },
    { merge: false }
  );
});

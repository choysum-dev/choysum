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

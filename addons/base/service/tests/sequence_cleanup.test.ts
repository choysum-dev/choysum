// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ChoysumError } from '@/core/service/error';
import { withContext as withModelContext } from '@/core/service/api/context';

import Company from '@/base/service/models/company';
import Currency from '@/base/service/models/currency';
import Sequence from '@/base/service/models/sequence';
import SequenceIdempotency from '@/base/service/models/sequence_idempotency';

import { companyCode8, currencyCode3, uid } from './_helpers';

async function createCompanyForSequence(): Promise<string> {
  const currency = await Currency.Create(
    {
      Name: uid('SeqCleanupCurrency'),
      Code: currencyCode3(),
      DecimalDigits: 2,
      Rounding: '0.01',
      IsActive: true,
    } as any,
    ['Id'] as any
  );

  const company = await Company.Create(
    {
      Name: uid('SeqCleanupCompany'),
      Code: companyCode8(),
      Timezone: 'UTC',
      CurrencyId: (currency as any).Id,
      IsActive: true,
    } as any,
    ['Id'] as any
  );

  return String((company as any).Id);
}

async function expectBaseInvalidArgument(fn: () => Promise<any>): Promise<void> {
  try {
    await fn();
    throw new Error('expected InvalidArgument error');
  } catch (err) {
    expect(err instanceof ChoysumError).toBe(true);
    const oe = err as ChoysumError;
    expect(oe.domain).toBe('base');
    expect(oe.code).toBe('InvalidArgument');
  }
}

test('base.sequence: CleanupIdempotency validates OlderThan and deletes expired records', async () => {
  const companyId = await createCompanyForSequence();

  await withModelContext(
    {
      activeCompanyId: companyId,
      enabledCompanyIds: [companyId],
    } as any,
    async () => {
      const seqCode = `cleanup.seq.${uid('code')}`;
      const sequence = await Sequence.Create(
        {
          Name: uid('CleanupSequence'),
          Code: seqCode,
          CompanyId: companyId,
          Prefix: 'CL/',
          Padding: 4,
          NextNumber: '1',
          IsActive: true,
        } as any,
        ['Id'] as any
      );

      await SequenceIdempotency.Create(
        {
          CompanyId: companyId,
          SequenceId: (sequence as any).Id,
          CodeSnapshot: seqCode,
          FormatSnapshot: { Prefix: 'CL/', Suffix: '', Padding: 4 },
          IdempotencyKey: `cleanup.idem.${uid('k')}`,
          Count: 1,
          DryRun: false,
          RangeStart: '1',
          RangeEnd: '1',
          ExpiresAt: new Date(Date.now() - 60 * 1000),
        } as any,
        ['Id'] as any
      );

      await expectBaseInvalidArgument(async () => {
        await Sequence.CleanupIdempotency({ OlderThan: '2026/01/01' });
      });

      const cleaned = await Sequence.CleanupIdempotency();
      expect(Number(cleaned.Deleted) >= 1).toBe(true);
    },
    { merge: false }
  );
});

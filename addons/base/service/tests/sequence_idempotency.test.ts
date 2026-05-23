// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ChoysumError } from '@/core/service/error';
import { withContext as withModelContext } from '@/core/service/api/context';
import { resolveValidationSummary } from '@/core/service/api/validation';

import Company from '@/base/service/models/company';
import Currency from '@/base/service/models/currency';
import Sequence from '@/base/service/models/sequence';
import SequenceIdempotency from '@/base/service/models/sequence_idempotency';

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

async function createCompanyForSequence(): Promise<string> {
  const currency = await Currency.Create(
    {
      Name: uid('SeqIdemCurrency'),
      Code: currencyCode3(),
      DecimalDigits: 2,
      Rounding: '0.01',
      IsActive: true,
    } as any,
    ['Id'] as any
  );

  const company = await Company.Create(
    {
      Name: uid('SeqIdemCompany'),
      Code: companyCode8(),
      Timezone: 'UTC',
      CurrencyId: (currency as any).Id,
      IsActive: true,
    } as any,
    ['Id'] as any
  );

  return String((company as any).Id);
}

test('base.sequence_idempotency: Create enforces count and range invariants', async () => {
  const companyId = await createCompanyForSequence();

  await withModelContext(
    {
      activeCompanyId: companyId,
      enabledCompanyIds: [companyId],
    } as any,
    async () => {
      const sequence = await Sequence.Create(
        {
          Name: uid('SeqIdemInvariant'),
          Code: `seq.idem.inv.${uid('code')}`,
          CompanyId: companyId,
          Prefix: 'SI/',
          Padding: 4,
          NextNumber: '1',
          IsActive: true,
        } as any,
        ['Id'] as any
      );

      await expectRepoValidationFailed('create', async () => {
        await SequenceIdempotency.Create(
          {
            CompanyId: companyId,
            SequenceId: (sequence as any).Id,
            CodeSnapshot: 'S1',
            FormatSnapshot: { Prefix: 'SI/', Suffix: '', Padding: 4 },
            IdempotencyKey: `idem.${uid('count0')}`,
            Count: 0,
            DryRun: false,
            RangeStart: '10',
            RangeEnd: '9',
            ExpiresAt: new Date(Date.now() + 60 * 1000),
          } as any,
          ['Id'] as any
        );
      });

      await expectRepoValidationFailed('create', async () => {
        await SequenceIdempotency.Create(
          {
            CompanyId: companyId,
            SequenceId: (sequence as any).Id,
            CodeSnapshot: 'S1',
            FormatSnapshot: { Prefix: 'SI/', Suffix: '', Padding: 4 },
            IdempotencyKey: `idem.${uid('range')}`,
            Count: 2,
            DryRun: false,
            RangeStart: '10',
            RangeEnd: '12',
            ExpiresAt: new Date(Date.now() + 60 * 1000),
          } as any,
          ['Id'] as any
        );
      });
    },
    { merge: false }
  );
});

test('base.sequence_idempotency: enforces CompanyId and Sequence.CompanyId consistency on create and update', async () => {
  const companyA = await createCompanyForSequence();
  const companyB = await createCompanyForSequence();

  let sequenceA: any;
  let sequenceB: any;

  await withModelContext(
    {
      activeCompanyId: companyA,
      enabledCompanyIds: [companyA, companyB],
    } as any,
    async () => {
      sequenceA = await Sequence.Create(
        {
          Name: uid('SeqIdemA'),
          Code: `seq.idem.a.${uid('code')}`,
          CompanyId: companyA,
          Prefix: 'A/',
          Padding: 4,
          NextNumber: '1',
          IsActive: true,
        } as any,
        ['Id'] as any
      );

      sequenceB = await Sequence.Create(
        {
          Name: uid('SeqIdemB'),
          Code: `seq.idem.b.${uid('code')}`,
          CompanyId: companyB,
          Prefix: 'B/',
          Padding: 4,
          NextNumber: '1',
          IsActive: true,
        } as any,
        ['Id'] as any
      );

      await expectRepoValidationFailed('create', async () => {
        await SequenceIdempotency.Create(
          {
            CompanyId: companyA,
            SequenceId: (sequenceB as any).Id,
            CodeSnapshot: 'S2',
            FormatSnapshot: { Prefix: 'B/', Suffix: '', Padding: 4 },
            IdempotencyKey: `idem.${uid('mismatch.create')}`,
            Count: 1,
            DryRun: false,
            RangeStart: '1',
            RangeEnd: '1',
            ExpiresAt: new Date(Date.now() + 60 * 1000),
          } as any,
          ['Id'] as any
        );
      });

      const valid = await SequenceIdempotency.Create(
        {
          CompanyId: companyA,
          SequenceId: (sequenceA as any).Id,
          CodeSnapshot: 'S2',
          FormatSnapshot: { Prefix: 'A/', Suffix: '', Padding: 4 },
          IdempotencyKey: `idem.${uid('valid')}`,
          Count: 1,
          DryRun: false,
          RangeStart: '20',
          RangeEnd: '20',
          ExpiresAt: new Date(Date.now() + 60 * 1000),
        } as any,
        ['Id'] as any
      );

      await expectRepoValidationFailed('update', async () => {
        await SequenceIdempotency.UpdateById(
          String((valid as any).Id),
          {
            Count: 2,
          } as any,
          ['Id'] as any
        );
      });

      await expectRepoValidationFailed('update', async () => {
        await SequenceIdempotency.UpdateById(
          String((valid as any).Id),
          {
            SequenceId: (sequenceB as any).Id,
          } as any,
          ['Id'] as any
        );
      });
    },
    { merge: false }
  );
});

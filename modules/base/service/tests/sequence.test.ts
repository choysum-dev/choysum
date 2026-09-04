// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ChoysumError } from '@/core/service/error';
import { withContext as withModelContext } from '@/core/service/api/context';

import Company from '@/base/service/models/company';
import Currency from '@/base/service/models/currency';
import Sequence from '@/base/service/models/sequence';
import SequenceIdempotency from '@/base/service/models/sequence_idempotency';

import { companyCode8, currencyCode3, expectBaseInvalidArgument, uid } from './_helpers';

const TTL_ENV_KEY = 'CHOYSUM_BASE_SEQUENCE_IDEMPOTENCY_TTL_DAYS';

async function withBackendEnv<T>(key: string, value: any, run: () => Promise<T>): Promise<T> {
  const meta = import.meta as any;
  if (!meta.env) meta.env = {};
  const prev = meta.env[key];
  meta.env[key] = value;

  const globalAny = globalThis as any;
  const envKey = '__choysumBackendEnv';
  if (!globalAny[envKey]) globalAny[envKey] = {};
  const prevGlobal = globalAny[envKey][key];
  globalAny[envKey][key] = value;

  try {
    return await run();
  } finally {
    if (prev === undefined) {
      delete meta.env[key];
    } else {
      meta.env[key] = prev;
    }

    if (prevGlobal === undefined) {
      delete globalAny[envKey][key];
    } else {
      globalAny[envKey][key] = prevGlobal;
    }
  }
}

async function createCompanyForSequence(): Promise<string> {
  const currency = await Currency.Create(
    {
      Name: uid('SeqCurrency'),
      Code: currencyCode3(),
      DecimalDigits: 2,
      Rounding: '0.01',
      IsActive: true,
    } as any,
    ['Id'] as any
  );

  const company = await Company.Create(
    {
      Name: uid('SeqCompany'),
      Code: companyCode8(),
      Timezone: 'UTC',
      CurrencyId: (currency as any).Id,
      IsActive: true,
    } as any,
    ['Id'] as any
  );

  return String((company as any).Id);
}

async function expectBaseCode(fn: () => Promise<any>, code: string): Promise<void> {
  try {
    await fn();
    throw new Error(`expected base.${code}`);
  } catch (err) {
    expect(err instanceof ChoysumError).toBe(true);
    const oe = err as ChoysumError;
    expect(oe.domain).toBe('base');
    expect(oe.code).toBe(code);
  }
}

test('base.sequence: Next returns monotonic formatted values', async () => {
  const companyId = await createCompanyForSequence();

  await withModelContext(
    {
      activeCompanyId: companyId,
      enabledCompanyIds: [companyId],
    } as any,
    async () => {
      const seqCode = `sale.order.${uid('seq')}`;
      await Sequence.Create(
        {
          Name: uid('Sequence'),
          Code: seqCode,
          CompanyId: companyId,
          Prefix: 'SO/',
          Suffix: '',
          Padding: 4,
          NextNumber: '1',
          IsActive: true,
        } as any,
        ['Id'] as any
      );

      const first = await Sequence.Next({ CompanyId: companyId, Code: seqCode, Count: 2 });
      expect(first.Items.length).toBe(2);
      expect(first.Items[0].Number).toBe(1);
      expect(first.Items[0].Value).toBe('SO/0001');
      expect(first.Items[1].Number).toBe(2);
      expect(first.Items[1].Value).toBe('SO/0002');

      const second = await Sequence.Next({ CompanyId: companyId, Code: seqCode, Count: 1 });
      expect(second.Items.length).toBe(1);
      expect(second.Items[0].Number).toBe(3);
      expect(second.Items[0].Value).toBe('SO/0003');

      const defaultCount = await Sequence.Next({ CompanyId: companyId, Code: seqCode });
      expect(defaultCount.Items.length).toBe(1);
      expect(defaultCount.Items[0].Number).toBe(4);

      const nullCount = await Sequence.Next({ CompanyId: companyId, Code: seqCode, Count: null as any });
      expect(nullCount.Items.length).toBe(1);
      expect(nullCount.Items[0].Number).toBe(5);
    },
    { merge: false }
  );
});

test('base.sequence: idempotency returns same result and rejects mismatched request', async () => {
  const companyId = await createCompanyForSequence();

  await withModelContext(
    {
      activeCompanyId: companyId,
      enabledCompanyIds: [companyId],
    } as any,
    async () => {
      const seqCode = `account.move.${uid('seq')}`;
      await Sequence.Create(
        {
          Name: uid('Sequence'),
          Code: seqCode,
          CompanyId: companyId,
          Prefix: 'MOVE/',
          Suffix: '',
          Padding: 5,
          NextNumber: '1',
          IsActive: true,
        } as any,
        ['Id'] as any
      );

      const idemKey = `idem.${uid('k')}`;
      const first = await Sequence.Next({ CompanyId: companyId, Code: seqCode, Count: 2, IdempotencyKey: idemKey });
      const hit = await Sequence.Next({ CompanyId: companyId, Code: seqCode, Count: 2, IdempotencyKey: idemKey });

      expect(hit.Items.length).toBe(2);
      expect(hit.Items[0].Number).toBe(first.Items[0].Number);
      expect(hit.Items[0].Value).toBe(first.Items[0].Value);
      expect(hit.Items[1].Number).toBe(first.Items[1].Number);
      expect(hit.Items[1].Value).toBe(first.Items[1].Value);

      await expectBaseCode(async () => {
        await Sequence.Next({ CompanyId: companyId, Code: seqCode, Count: 1, IdempotencyKey: idemKey });
      }, 'Conflict');
    },
    { merge: false }
  );
});

test('base.sequence: validates count bounds and inactive sequence', async () => {
  await expectBaseInvalidArgument(async () => {
    await Sequence.Next({ Code: 'any', Count: 0 });
  });

  await expectBaseInvalidArgument(async () => {
    await Sequence.Next({ Code: 'any', Count: 1001 });
  });

  await expectBaseInvalidArgument(async () => {
    await Sequence.Next({ Code: 'any', Count: 1, IdempotencyKey: '   ' });
  });

  const companyId = await createCompanyForSequence();
  await withModelContext(
    {
      activeCompanyId: companyId,
      enabledCompanyIds: [companyId],
    } as any,
    async () => {
      const seqCode = `inactive.${uid('seq')}`;
      await Sequence.Create(
        {
          Name: uid('InactiveSeq'),
          Code: seqCode,
          CompanyId: companyId,
          Prefix: 'IN/',
          Padding: 3,
          NextNumber: '1',
          IsActive: false,
        } as any,
        ['Id'] as any
      );

      await expectBaseCode(async () => {
        await Sequence.Next({ CompanyId: companyId, Code: seqCode, Count: 1 });
      }, 'FailedPrecondition');
    },
    { merge: false }
  );
});

test('base.sequence: idempotency TTL supports env override and invalid fallback', async () => {
  const companyId = await createCompanyForSequence();

  await withModelContext(
    {
      activeCompanyId: companyId,
      enabledCompanyIds: [companyId],
    } as any,
    async () => {
      await withBackendEnv(TTL_ENV_KEY, 2, async () => {
        const seqCode = `ttl.${uid('seq2d')}`;
        await Sequence.Create(
          {
            Name: uid('TtlSeq2d'),
            Code: seqCode,
            CompanyId: companyId,
            Prefix: 'TTL/',
            Padding: 4,
            NextNumber: '1',
            IsActive: true,
          } as any,
          ['Id'] as any
        );

        const now = Date.now();
        const idemKey = `idem.${uid('ttl2')}`;
        await Sequence.Next({ CompanyId: companyId, Code: seqCode, Count: 1, IdempotencyKey: idemKey });
        const hits = await SequenceIdempotency.Search(
          {
            And: [['IdempotencyKey', '=', idemKey]],
          } as any,
          { limit: 1, fields: ['Id', 'ExpiresAt'] as any } as any
        );
        expect(hits.length).toBe(1);
        const expiresAt = new Date((hits[0] as any).ExpiresAt).getTime();
        const deltaHours = (expiresAt - now) / (60 * 60 * 1000);
        expect(deltaHours >= 47 && deltaHours <= 49).toBe(true);
      });

      await withBackendEnv(TTL_ENV_KEY, 'bad', async () => {
        const seqCode = `ttl.${uid('seqfb')}`;
        await Sequence.Create(
          {
            Name: uid('TtlSeqFb'),
            Code: seqCode,
            CompanyId: companyId,
            Prefix: 'TTL/',
            Padding: 4,
            NextNumber: '1',
            IsActive: true,
          } as any,
          ['Id'] as any
        );

        const now = Date.now();
        const idemKey = `idem.${uid('ttlfb')}`;
        await Sequence.Next({ CompanyId: companyId, Code: seqCode, Count: 1, IdempotencyKey: idemKey });
        const hits = await SequenceIdempotency.Search(
          {
            And: [['IdempotencyKey', '=', idemKey]],
          } as any,
          { limit: 1, fields: ['Id', 'ExpiresAt'] as any } as any
        );
        expect(hits.length).toBe(1);
        const expiresAt = new Date((hits[0] as any).ExpiresAt).getTime();
        const deltaHours = (expiresAt - now) / (60 * 60 * 1000);
        expect(deltaHours >= 167 && deltaHours <= 169).toBe(true);
      });
    },
    { merge: false }
  );
});

test('base.sequence: expired idempotency record is treated as miss', async () => {
  const companyId = await createCompanyForSequence();

  await withModelContext(
    {
      activeCompanyId: companyId,
      enabledCompanyIds: [companyId],
    } as any,
    async () => {
      const seqCode = `ttl.expired.${uid('seq')}`;
      await Sequence.Create(
        {
          Name: uid('ExpiredIdemSeq'),
          Code: seqCode,
          CompanyId: companyId,
          Prefix: 'EX/',
          Padding: 4,
          NextNumber: '1',
          IsActive: true,
        } as any,
        ['Id'] as any
      );

      const idemKey = `idem.${uid('expired')}`;
      const first = await Sequence.Next({ CompanyId: companyId, Code: seqCode, Count: 1, IdempotencyKey: idemKey });

      const hits = await SequenceIdempotency.Search(
        {
          And: [['IdempotencyKey', '=', idemKey]],
        } as any,
        { limit: 1, fields: ['Id'] as any } as any
      );
      expect(hits.length).toBe(1);

      await SequenceIdempotency.UpdateById(
        String((hits[0] as any).Id),
        {
          ExpiresAt: new Date(Date.now() - 60 * 1000),
        } as any,
        ['Id'] as any
      );

      const second = await Sequence.Next({ CompanyId: companyId, Code: seqCode, Count: 1, IdempotencyKey: idemKey });
      expect(second.Items.length).toBe(1);
      expect(second.Items[0].Number).toBe(first.Items[0].Number + 1);
    },
    { merge: false }
  );
});

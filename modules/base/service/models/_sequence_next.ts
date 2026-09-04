// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ChoysumError, GrpcCode, raiseDomainError } from '@/core/service/error';
import { createTranslate } from '@/core/service/i18n';
import { asBigInt, isExpiredAt, normalizeOptionalNonEmptyString, parsePositiveInt } from '@/core/service/utils/normalization';
import { buildPaddedNumberItems, resolvePaddedNumberFormat } from '@/core/service/utils/format';
import { getBackendEnvPositiveInt } from '@/core/service/runtime/env/backend_env';
import { mapNormalizationToBase, assertCodeRequired } from './_normalizers';
import { buildSequenceIdempotencyPayload, buildSequenceNextResult } from './_sequence_next_payload';
import type Sequence from './sequence';
import type { SequenceNextItem, SequenceNextParams, SequenceNextResult } from './sequence';

const { _t } = createTranslate('base');

const IDEMPOTENCY_TTL_ENV_KEY = 'CHOYSUM_BASE_SEQUENCE_IDEMPOTENCY_TTL_DAYS';
const DEFAULT_IDEMPOTENCY_TTL_DAYS = 7;
const IDEMPOTENCY_KEY_MAX_LENGTH = 200;

function assertCount(count: unknown): number {
  if (count == null) return 1;
  const n = mapNormalizationToBase(
    () => parsePositiveInt(count),
    () => _t('Count must be an integer >= 1', { scope: 'service/models/_sequence_next' })
  );
  if (n > 1000) {
    raiseDomainError('base', 'InvalidArgument', _t('Count must be within 1..1000', { scope: 'service/models/_sequence_next' }));
  }
  return n;
}

function assertIdempotencyKey(key: unknown): string | undefined {
  return mapNormalizationToBase(
    () => normalizeOptionalNonEmptyString(key, { maxLength: IDEMPOTENCY_KEY_MAX_LENGTH }),
    err => {
      if (err.code === 'required') return _t('IdempotencyKey must be non-empty', { scope: 'service/models/_sequence_next' });
      if (err.code === 'string_too_long') return _t('IdempotencyKey is too long', { scope: 'service/models/_sequence_next' });
      return _t('IdempotencyKey is invalid', { scope: 'service/models/_sequence_next' });
    }
  );
}

async function findIdempotencyHit(sequenceId: string, idempotencyKey: string): Promise<any | undefined> {
  const { default: SequenceIdempotency } = await import('./sequence_idempotency');
  const existing = await SequenceIdempotency.Search(
    {
      And: [
        ['SequenceId', '=', sequenceId],
        ['IdempotencyKey', '=', idempotencyKey],
      ],
    } as any,
    {
      limit: 1,
      fields: ['Id', 'SequenceId', 'CodeSnapshot', 'FormatSnapshot', 'IdempotencyKey', 'Count', 'DryRun', 'RangeStart', 'RangeEnd', 'ExpiresAt'] as any,
    } as any
  );
  return existing?.[0] as any;
}

function buildItemsFromIdempotencyHit(seq: Sequence, hit: any, count: number): SequenceNextItem[] {
  const { prefix, suffix, padding } = resolvePaddedNumberFormat(hit?.FormatSnapshot, {
    prefix: seq.Prefix,
    suffix: seq.Suffix,
    padding: seq.Padding,
  });
  const start = asBigInt(hit?.RangeStart);
  return buildPaddedNumberItems(start, count, prefix, suffix, padding);
}

function assertIdempotencyRequestMatch(hit: any, count: number, dryRun: boolean): void {
  if (Number(hit?.Count) !== count || Boolean(hit?.DryRun) !== dryRun) {
    throw new ChoysumError({
      domain: 'base',
      code: 'Conflict',
      message: _t('IdempotencyKey conflict with different request', { scope: 'service/models/_sequence_next' }),
    }).withGrpcCode(GrpcCode.Aborted);
  }
}

async function resolveSequence(
  model: { Search: (condition: any, options: any) => Promise<any[]> },
  companyId: string | undefined,
  code: string
): Promise<Sequence> {
  const company = String(companyId ?? '').trim();
  if (company) {
    const list = await model.Search(
      {
        And: [
          ['CompanyScopeKey', '=', company],
          ['Code', '=', code],
        ],
      } as any,
      {
        limit: 1,
        fields: ['Id', 'CompanyId', 'CompanyScopeKey', 'Code', 'Prefix', 'Suffix', 'Padding', 'NextNumber', 'IsActive', 'UpdatedAt'] as any,
      } as any
    );
    if (list?.[0]) return list[0] as any;
  }

  const global = await model.Search(
    {
      And: [
        ['CompanyScopeKey', '=', '__GLOBAL__'],
        ['Code', '=', code],
      ],
    } as any,
    {
      limit: 1,
      fields: ['Id', 'CompanyId', 'CompanyScopeKey', 'Code', 'Prefix', 'Suffix', 'Padding', 'NextNumber', 'IsActive', 'UpdatedAt'] as any,
    } as any
  );
  if (global?.[0]) return global[0] as any;

  throw new ChoysumError({
    domain: 'base',
    code: 'NotFound',
    message: _t('Sequence not found for Code=%s', { scope: 'service/models/_sequence_next' }, code),
  }).withGrpcCode(GrpcCode.NotFound);
}

async function allocateRangeAtomic(
  model: { Browse: (id: string, fields: any) => Promise<any>; Update: (condition: any, values: any, fields: any) => Promise<any> },
  seq: Sequence,
  count: number
): Promise<{ rangeStart: bigint; rangeEnd: bigint }> {
  for (let attempt = 0; attempt < 20; attempt++) {
    const current = attempt === 0 ? seq : ((await model.Browse(seq.Id, ['Id', 'NextNumber'] as any)) as any);
    if (!current) {
      throw new ChoysumError({ domain: 'base', code: 'NotFound', message: _t('Sequence not found', { scope: 'service/models/_sequence_next' }) }).withGrpcCode(GrpcCode.NotFound);
    }
    const currentNext = asBigInt((current as any).NextNumber);
    const rangeStart = currentNext;
    const rangeEnd = currentNext + BigInt(count) - 1n;
    const nextNumber = rangeEnd + 1n;
    const cond = {
      And: [
        ['Id', '=', seq.Id],
        ['NextNumber', '=', currentNext.toString()],
      ],
    } as any;

    const res = await model.Update(cond, { NextNumber: nextNumber.toString() } as any, ['Id', 'UpdatedAt', 'NextNumber'] as any);
    if (Array.isArray(res) && res.length > 0) {
      return { rangeStart, rangeEnd };
    }
  }

  throw new ChoysumError({
    domain: 'base',
    code: 'Conflict',
    message: _t('Sequence allocation conflicted; please retry', { scope: 'service/models/_sequence_next' }),
  }).withGrpcCode(GrpcCode.Aborted);
}

export async function nextSequence(
  model: {
    Search: (condition: any, options: any) => Promise<any[]>;
    Browse: (id: string, fields: any) => Promise<any>;
    Update: (condition: any, values: any, fields: any) => Promise<any>;
  },
  params: SequenceNextParams
): Promise<SequenceNextResult> {
  const code = assertCodeRequired(params?.Code, { uppercase: false });
  const count = assertCount(params?.Count);
  const idemKey = assertIdempotencyKey(params?.IdempotencyKey);
  const dryRun = params?.DryRun === true;

  const seq = await resolveSequence(model, params?.CompanyId, code);
  if (seq.IsActive !== true) {
    throw new ChoysumError({ domain: 'base', code: 'FailedPrecondition', message: _t('Sequence is inactive', { scope: 'service/models/_sequence_next' }) }).withGrpcCode(
      GrpcCode.FailedPrecondition
    );
  }

  const generatedAt = new Date().toISOString();
  let expiredIdempotencyHitId: string | undefined;

  // Idempotency hit path
  if (idemKey) {
    const hit = await findIdempotencyHit(seq.Id, idemKey);
    if (hit?.Id) {
      if (!isExpiredAt((hit as any)?.ExpiresAt)) {
        assertIdempotencyRequestMatch(hit, count, dryRun);
        const items = buildItemsFromIdempotencyHit(seq, hit, count);
        return buildSequenceNextResult(seq, items, generatedAt);
      }
      expiredIdempotencyHitId = String((hit as any).Id || '');
    }
  }

  // Dry-run path: preview only
  if (dryRun) {
    const cur = (await model.Browse(seq.Id, ['Id', 'NextNumber'] as any)) as any;
    if (!cur) {
      throw new ChoysumError({ domain: 'base', code: 'NotFound', message: _t('Sequence not found', { scope: 'service/models/_sequence_next' }) }).withGrpcCode(GrpcCode.NotFound);
    }
    const start = asBigInt(cur.NextNumber);
    const items = buildPaddedNumberItems(start, count, seq.Prefix, seq.Suffix, seq.Padding);
    return buildSequenceNextResult(seq, items, generatedAt);
  }

  // Allocate range
  const { rangeStart, rangeEnd } = await allocateRangeAtomic(model, seq, count);
  const items = buildPaddedNumberItems(rangeStart, count, seq.Prefix, seq.Suffix, seq.Padding);

  // Persist idempotency record (strong consistency)
  if (idemKey) {
    const { default: SequenceIdempotency } = await import('./sequence_idempotency');
    const ttlDays = getBackendEnvPositiveInt(IDEMPOTENCY_TTL_ENV_KEY, DEFAULT_IDEMPOTENCY_TTL_DAYS);
    const expiresAt = new Date(Date.now() + ttlDays * 24 * 60 * 60 * 1000);
    const payload = buildSequenceIdempotencyPayload(seq, {
      idempotencyKey: idemKey,
      count,
      dryRun,
      rangeStart,
      rangeEnd,
      expiresAt,
    });

    if (expiredIdempotencyHitId) {
      try {
        await SequenceIdempotency.UpdateById(expiredIdempotencyHitId, payload as any, ['Id'] as any);
        return buildSequenceNextResult(seq, items, generatedAt);
      } catch {
        expiredIdempotencyHitId = undefined;
      }
    }

    try {
      await SequenceIdempotency.Create(payload as any);
    } catch (err) {
      const hit = await findIdempotencyHit(seq.Id, idemKey);
      if (hit?.Id) {
        if (!isExpiredAt((hit as any)?.ExpiresAt)) {
          assertIdempotencyRequestMatch(hit, count, dryRun);
          const replayItems = buildItemsFromIdempotencyHit(seq, hit, count);
          return buildSequenceNextResult(seq, replayItems, generatedAt);
        }

        try {
          await SequenceIdempotency.UpdateById(String((hit as any).Id || ''), payload as any, ['Id'] as any);
        } catch (replaceErr) {
          const replayHit = await findIdempotencyHit(seq.Id, idemKey);
          if (replayHit?.Id && !isExpiredAt((replayHit as any)?.ExpiresAt)) {
            assertIdempotencyRequestMatch(replayHit, count, dryRun);
            const replayItems = buildItemsFromIdempotencyHit(seq, replayHit, count);
            return buildSequenceNextResult(seq, replayItems, generatedAt);
          }
          throw replaceErr;
        }
        return buildSequenceNextResult(seq, items, generatedAt);
      }
      throw err;
    }
  }

  return buildSequenceNextResult(seq, items, generatedAt);
}

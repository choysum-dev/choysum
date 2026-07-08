// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ChoysumError, GrpcCode } from '@/core/service/error';
import { normalizeCodeRequired } from './_normalizers';
import type Sequence from './sequence';
import type { SequenceNextItem, SequenceNextParams, SequenceNextResult } from './sequence';

const IDEMPOTENCY_TTL_ENV_KEY = 'CHOYSUM_BASE_SEQUENCE_IDEMPOTENCY_TTL_DAYS';
const DEFAULT_IDEMPOTENCY_TTL_DAYS = 7;

function normalizeCount(count: unknown): number {
  if (count == null) return 1;
  const n = Number(count);
  if (!Number.isFinite(n) || n <= 0 || Math.floor(n) !== n) {
    throw new ChoysumError({ domain: 'base', code: 'InvalidArgument', message: 'Count must be a positive integer' }).withGrpcCode(GrpcCode.InvalidArgument);
  }
  if (n < 1 || n > 1000) {
    throw new ChoysumError({ domain: 'base', code: 'InvalidArgument', message: 'Count must be within 1..1000' }).withGrpcCode(GrpcCode.InvalidArgument);
  }
  return n;
}

function normalizeIdempotencyKey(key: unknown): string | undefined {
  if (key === undefined || key === null) return undefined;
  const v = String(key).trim();
  if (!v) {
    throw new ChoysumError({ domain: 'base', code: 'InvalidArgument', message: 'IdempotencyKey must be non-empty' }).withGrpcCode(GrpcCode.InvalidArgument);
  }
  if (v.length > 200) {
    throw new ChoysumError({ domain: 'base', code: 'InvalidArgument', message: 'IdempotencyKey is too long' }).withGrpcCode(GrpcCode.InvalidArgument);
  }
  return v;
}

function resolveIdempotencyTtlDays(): number {
  const globalEnv = (globalThis as any)?.__choysumBackendEnv as Record<string, any> | undefined;
  const raw = globalEnv?.[IDEMPOTENCY_TTL_ENV_KEY] ?? (import.meta as any)?.env?.[IDEMPOTENCY_TTL_ENV_KEY];
  const parsed = typeof raw === 'number' ? raw : typeof raw === 'string' ? Number(raw) : NaN;
  if (!Number.isFinite(parsed) || parsed <= 0) return DEFAULT_IDEMPOTENCY_TTL_DAYS;
  return Math.floor(parsed);
}

function asBigInt(v: any): bigint {
  if (typeof v === 'bigint') return v;
  if (v && typeof v === 'object' && typeof v.$bigint === 'string') return BigInt(v.$bigint);
  if (typeof v === 'number' && Number.isFinite(v)) return BigInt(Math.trunc(v));
  const s = String(v ?? '').trim();
  if (!s) return 0n;
  return BigInt(s);
}

function formatValue(prefix: string | undefined, suffix: string | undefined, padding: number, number: bigint): string {
  const p = prefix ?? '';
  const s = suffix ?? '';
  const pad = Number.isFinite(padding) && padding > 0 ? Math.floor(padding) : 0;
  const core = number.toString();
  const padded = pad > 0 ? core.padStart(pad, '0') : core;
  return `${p}${padded}${s}`;
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
  const snap = (hit?.FormatSnapshot || {}) as any;
  const prefix = typeof snap.Prefix === 'string' ? snap.Prefix : seq.Prefix;
  const suffix = typeof snap.Suffix === 'string' ? snap.Suffix : seq.Suffix;
  const padding = Number.isFinite(Number(snap.Padding)) ? Number(snap.Padding) : seq.Padding;
  const start = asBigInt(hit?.RangeStart);
  const items: SequenceNextItem[] = [];
  for (let i = 0; i < count; i++) {
    const n = start + BigInt(i);
    items.push({ Value: formatValue(prefix, suffix, padding, n), Number: Number(n) });
  }
  return items;
}

function assertIdempotencyRequestMatch(hit: any, count: number, dryRun: boolean): void {
  if (Number(hit?.Count) !== count || Boolean(hit?.DryRun) !== dryRun) {
    throw new ChoysumError({ domain: 'base', code: 'Conflict', message: 'IdempotencyKey conflict with different request' }).withGrpcCode(GrpcCode.Aborted);
  }
}

function isIdempotencyExpired(hit: any, nowMs: number = Date.now()): boolean {
  const raw = hit?.ExpiresAt;
  if (!raw) return true;
  const t = new Date(raw).getTime();
  if (!Number.isFinite(t)) return true;
  return t <= nowMs;
}

function buildNextResult(seq: Sequence, items: SequenceNextItem[], generatedAt: string): SequenceNextResult {
  return {
    Items: items,
    Sequence: {
      Id: seq.Id,
      CompanyId: (seq as any).CompanyId?.Id ?? (seq as any).CompanyId,
      Code: seq.Code,
      Prefix: seq.Prefix,
      Suffix: seq.Suffix,
      Padding: seq.Padding,
    },
    GeneratedAt: generatedAt,
  };
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

  throw new ChoysumError({ domain: 'base', code: 'NotFound', message: `Sequence not found for Code=${code}` }).withGrpcCode(GrpcCode.NotFound);
}

async function allocateRangeAtomic(
  model: { Browse: (id: string, fields: any) => Promise<any>; Update: (condition: any, values: any, fields: any) => Promise<any> },
  seq: Sequence,
  count: number
): Promise<{ rangeStart: bigint; rangeEnd: bigint; updated: Sequence }> {
  for (let attempt = 0; attempt < 20; attempt++) {
    const current = attempt === 0 ? seq : ((await model.Browse(seq.Id, ['Id', 'NextNumber'] as any)) as any);
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
      const updated = (await model.Browse(seq.Id, ['Id', 'CompanyId', 'Code', 'Prefix', 'Suffix', 'Padding', 'IsActive'] as any)) as any;
      return { rangeStart, rangeEnd, updated };
    }
  }

  throw new ChoysumError({ domain: 'base', code: 'Conflict', message: 'Sequence allocation conflicted; please retry' }).withGrpcCode(GrpcCode.Aborted);
}

export async function nextSequence(
  model: {
    Search: (condition: any, options: any) => Promise<any[]>;
    Browse: (id: string, fields: any) => Promise<any>;
    Update: (condition: any, values: any, fields: any) => Promise<any>;
  },
  params: SequenceNextParams
): Promise<SequenceNextResult> {
  const code = normalizeCodeRequired(params?.Code, { uppercase: false });
  const count = normalizeCount(params?.Count);
  const idemKey = normalizeIdempotencyKey(params?.IdempotencyKey);
  const dryRun = params?.DryRun === true;

  const seq = await resolveSequence(model, params?.CompanyId, code);
  if (seq.IsActive !== true) {
    throw new ChoysumError({ domain: 'base', code: 'FailedPrecondition', message: 'Sequence is inactive' }).withGrpcCode(GrpcCode.FailedPrecondition);
  }

  const generatedAt = new Date().toISOString();
  let expiredIdempotencyHitId: string | undefined;

  // Idempotency hit path
  if (idemKey) {
    const hit = await findIdempotencyHit(seq.Id, idemKey);
    if (hit?.Id) {
      if (!isIdempotencyExpired(hit)) {
        assertIdempotencyRequestMatch(hit, count, dryRun);
        const items = buildItemsFromIdempotencyHit(seq, hit, count);
        return buildNextResult(seq, items, generatedAt);
      }
      expiredIdempotencyHitId = String((hit as any).Id || '');
    }
  }

  // Dry-run path: preview only
  if (dryRun) {
    const cur = (await model.Browse(seq.Id, ['Id', 'NextNumber'] as any)) as any;
    const start = asBigInt(cur.NextNumber);
    const items: SequenceNextItem[] = [];
    for (let i = 0; i < count; i++) {
      const n = start + BigInt(i);
      items.push({ Value: formatValue(seq.Prefix, seq.Suffix, seq.Padding, n), Number: Number(n) });
    }
    return buildNextResult(seq, items, generatedAt);
  }

  // Allocate range
  const { rangeStart, rangeEnd } = await allocateRangeAtomic(model, seq, count);
  const items: SequenceNextItem[] = [];
  for (let i = 0; i < count; i++) {
    const n = rangeStart + BigInt(i);
    items.push({ Value: formatValue(seq.Prefix, seq.Suffix, seq.Padding, n), Number: Number(n) });
  }

  // Persist idempotency record (strong consistency)
  if (idemKey) {
    const { default: SequenceIdempotency } = await import('./sequence_idempotency');
    const ttlDays = resolveIdempotencyTtlDays();
    const expiresAt = new Date(Date.now() + ttlDays * 24 * 60 * 60 * 1000);
    const formatSnapshot = { Prefix: seq.Prefix ?? '', Suffix: seq.Suffix ?? '', Padding: seq.Padding };
    const payload = {
      CompanyId: (seq as any).CompanyId?.Id ?? (seq as any).CompanyId ?? null,
      SequenceId: seq.Id,
      CodeSnapshot: seq.Code,
      FormatSnapshot: formatSnapshot,
      IdempotencyKey: idemKey,
      Count: count,
      DryRun: dryRun,
      RangeStart: rangeStart.toString(),
      RangeEnd: rangeEnd.toString(),
      ExpiresAt: expiresAt,
    };

    if (expiredIdempotencyHitId) {
      try {
        await SequenceIdempotency.UpdateById(expiredIdempotencyHitId, payload as any, ['Id'] as any);
        return buildNextResult(seq, items, generatedAt);
      } catch {
        expiredIdempotencyHitId = undefined;
      }
    }

    try {
      await SequenceIdempotency.Create(payload as any);
    } catch (err) {
      const hit = await findIdempotencyHit(seq.Id, idemKey);
      if (hit?.Id) {
        if (!isIdempotencyExpired(hit)) {
          assertIdempotencyRequestMatch(hit, count, dryRun);
          const replayItems = buildItemsFromIdempotencyHit(seq, hit, count);
          return buildNextResult(seq, replayItems, generatedAt);
        }

        try {
          await SequenceIdempotency.UpdateById(String((hit as any).Id || ''), payload as any, ['Id'] as any);
        } catch (replaceErr) {
          const replayHit = await findIdempotencyHit(seq.Id, idemKey);
          if (replayHit?.Id && !isIdempotencyExpired(replayHit)) {
            assertIdempotencyRequestMatch(replayHit, count, dryRun);
            const replayItems = buildItemsFromIdempotencyHit(seq, replayHit, count);
            return buildNextResult(seq, replayItems, generatedAt);
          }
          throw replaceErr;
        }
        return buildNextResult(seq, items, generatedAt);
      }
      throw err;
    }
  }

  return buildNextResult(seq, items, generatedAt);
}

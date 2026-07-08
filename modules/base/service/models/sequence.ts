// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { Constraint } from '@/core/service/api/constraint';
import { GrpcCode, ChoysumError } from '@/core/service/error';
import Company from './company';
import { asRefId, normalizeCompanyScopeKey } from './_refs';
import { normalizeCodeRequired } from './_normalizers';

export type SequenceNextParams = {
  CompanyId?: string;
  Code: string;
  Count?: number;
  IdempotencyKey?: string;
  DryRun?: boolean;
};

export type SequenceNextItem = { Value: string; Number: number };

export type SequenceNextResult = {
  Items: SequenceNextItem[];
  Sequence: { Id: string; CompanyId?: string; Code: string; Prefix?: string; Suffix?: string; Padding: number };
  GeneratedAt: string;
};

export type SequenceCleanupIdempotencyParams = { OlderThan?: string };

export type SequenceCleanupIdempotencyResult = { Deleted: number };

const IDEMPOTENCY_TTL_ENV_KEY = 'CHOYSUM_BASE_SEQUENCE_IDEMPOTENCY_TTL_DAYS';
const DEFAULT_IDEMPOTENCY_TTL_DAYS = 7;

@Model('Sequence', { companyScoped: true })
export default class Sequence extends BaseModel {
  @Field({ type: 'varchar', column: { size: 100, notNull: true, index: true } })
  Name: string;

  @Field({ type: 'varchar', column: { size: 100, notNull: true, index: true, uniqueIndex: 'uidx_base_sequence_scope_code' } })
  Code: string;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => Company }, column: { index: true } })
  CompanyId?: Company;

  @Field({ type: 'varchar', column: { size: 20, notNull: true, default: () => '__GLOBAL__', index: true, uniqueIndex: 'uidx_base_sequence_scope_code' } })
  CompanyScopeKey: string;

  @Field({ type: 'varchar', column: { size: 64 } })
  Prefix?: string;

  @Field({ type: 'varchar', column: { size: 64 } })
  Suffix?: string;

  @Field({ type: 'int', column: { notNull: true, default: () => 5 } })
  Padding: number;

  @Field({ type: 'bigint', column: { notNull: true, default: () => 1 } })
  NextNumber: bigint;

  @Field({ type: 'boolean', column: { notNull: true, default: () => true, index: true } })
  IsActive: boolean;

  private static validateWriteEntity(values: Record<string, any>): void {
    values.Code = normalizeCodeRequired(values.Code, { uppercase: false });
    values.CompanyScopeKey = normalizeCompanyScopeKey(values.CompanyId);
  }

  @Constraint<Sequence>(['Code', 'CompanyId'])
  static validateSequenceConstraint(self: Sequence, ctx: any): void {
    const mode = String(ctx?.mode || '');
    const values = (ctx?.values || {}) as Record<string, any>;
    Sequence.validateWriteEntity(self as any);

    if (mode === 'create' || Object.prototype.hasOwnProperty.call(values, 'Code')) {
      values.Code = self.Code;
    }
    if (mode === 'create' || Object.prototype.hasOwnProperty.call(values, 'CompanyId') || Object.prototype.hasOwnProperty.call(values, 'CompanyScopeKey')) {
      values.CompanyScopeKey = (self as any).CompanyScopeKey;
    }
  }

  private static normalizeCount(count: unknown): number {
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

  private static normalizeIdempotencyKey(key: unknown): string | undefined {
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

  private static resolveIdempotencyTtlDays(): number {
    const globalEnv = (globalThis as any)?.__choysumBackendEnv as Record<string, any> | undefined;
    const raw = globalEnv?.[IDEMPOTENCY_TTL_ENV_KEY] ?? (import.meta as any)?.env?.[IDEMPOTENCY_TTL_ENV_KEY];
    const parsed = typeof raw === 'number' ? raw : typeof raw === 'string' ? Number(raw) : NaN;
    if (!Number.isFinite(parsed) || parsed <= 0) return DEFAULT_IDEMPOTENCY_TTL_DAYS;
    return Math.floor(parsed);
  }

  private static asBigInt(v: any): bigint {
    if (typeof v === 'bigint') return v;
    if (v && typeof v === 'object' && typeof v.$bigint === 'string') return BigInt(v.$bigint);
    if (typeof v === 'number' && Number.isFinite(v)) return BigInt(Math.trunc(v));
    const s = String(v ?? '').trim();
    if (!s) return 0n;
    return BigInt(s);
  }

  private static formatValue(prefix: string | undefined, suffix: string | undefined, padding: number, number: bigint): string {
    const p = prefix ?? '';
    const s = suffix ?? '';
    const pad = Number.isFinite(padding) && padding > 0 ? Math.floor(padding) : 0;
    const core = number.toString();
    const padded = pad > 0 ? core.padStart(pad, '0') : core;
    return `${p}${padded}${s}`;
  }

  private static async findIdempotencyHit(sequenceId: string, idempotencyKey: string): Promise<any | undefined> {
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

  private static buildItemsFromIdempotencyHit(seq: Sequence, hit: any, count: number): SequenceNextItem[] {
    const snap = (hit?.FormatSnapshot || {}) as any;
    const prefix = typeof snap.Prefix === 'string' ? snap.Prefix : seq.Prefix;
    const suffix = typeof snap.Suffix === 'string' ? snap.Suffix : seq.Suffix;
    const padding = Number.isFinite(Number(snap.Padding)) ? Number(snap.Padding) : seq.Padding;
    const start = this.asBigInt(hit?.RangeStart);
    const items: SequenceNextItem[] = [];
    for (let i = 0; i < count; i++) {
      const n = start + BigInt(i);
      items.push({ Value: this.formatValue(prefix, suffix, padding, n), Number: Number(n) });
    }
    return items;
  }

  private static assertIdempotencyRequestMatch(hit: any, count: number, dryRun: boolean): void {
    if (Number(hit?.Count) !== count || Boolean(hit?.DryRun) !== dryRun) {
      throw new ChoysumError({ domain: 'base', code: 'Conflict', message: 'IdempotencyKey conflict with different request' }).withGrpcCode(GrpcCode.Aborted);
    }
  }

  private static isIdempotencyExpired(hit: any, nowMs: number = Date.now()): boolean {
    const raw = hit?.ExpiresAt;
    if (!raw) return true;
    const t = new Date(raw).getTime();
    if (!Number.isFinite(t)) return true;
    return t <= nowMs;
  }

  private static buildNextResult(seq: Sequence, items: SequenceNextItem[], generatedAt: string): SequenceNextResult {
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

  private static async resolveSequence(companyId: string | undefined, code: string): Promise<Sequence> {
    const company = String(companyId ?? '').trim();
    if (company) {
      const list = await this.Search(
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

    const global = await this.Search(
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

  private static async allocateRangeAtomic(seq: Sequence, count: number): Promise<{ rangeStart: bigint; rangeEnd: bigint; updated: Sequence }> {
    // Best-effort optimistic lock loop.
    for (let attempt = 0; attempt < 20; attempt++) {
      const current = attempt === 0 ? seq : ((await this.Browse(seq.Id, ['Id', 'NextNumber'] as any)) as any);
      const currentNext = this.asBigInt((current as any).NextNumber);
      const rangeStart = currentNext;
      const rangeEnd = currentNext + BigInt(count) - 1n;
      const nextNumber = rangeEnd + 1n;
      const cond = {
        And: [
          ['Id', '=', seq.Id],
          ['NextNumber', '=', currentNext.toString()],
        ],
      } as any;

      const res = await this.Update(cond, { NextNumber: nextNumber.toString() } as any, ['Id', 'UpdatedAt', 'NextNumber'] as any);
      if (Array.isArray(res) && res.length > 0) {
        const updated = (await this.Browse(seq.Id, ['Id', 'CompanyId', 'Code', 'Prefix', 'Suffix', 'Padding', 'IsActive'] as any)) as any;
        return { rangeStart, rangeEnd, updated };
      }
    }

    throw new ChoysumError({ domain: 'base', code: 'Conflict', message: 'Sequence allocation conflicted; please retry' }).withGrpcCode(GrpcCode.Aborted);
  }

  static async Next(params: SequenceNextParams): Promise<SequenceNextResult> {
    const code = normalizeCodeRequired(params?.Code, { uppercase: false });
    const count = this.normalizeCount(params?.Count);
    const idemKey = this.normalizeIdempotencyKey(params?.IdempotencyKey);
    const dryRun = params?.DryRun === true;

    const seq = await this.resolveSequence(params?.CompanyId, code);
    if (seq.IsActive !== true) {
      throw new ChoysumError({ domain: 'base', code: 'FailedPrecondition', message: 'Sequence is inactive' }).withGrpcCode(GrpcCode.FailedPrecondition);
    }

    const generatedAt = new Date().toISOString();
    let expiredIdempotencyHitId: string | undefined;

    // Idempotency hit path
    if (idemKey) {
      const hit = await this.findIdempotencyHit(seq.Id, idemKey);
      if (hit?.Id) {
        if (!this.isIdempotencyExpired(hit)) {
          this.assertIdempotencyRequestMatch(hit, count, dryRun);
          const items = this.buildItemsFromIdempotencyHit(seq, hit, count);
          return this.buildNextResult(seq, items, generatedAt);
        }
        expiredIdempotencyHitId = String((hit as any).Id || '');
      }
    }

    // Dry-run path: preview only
    if (dryRun) {
      const cur = (await this.Browse(seq.Id, ['Id', 'NextNumber'] as any)) as any;
      const start = this.asBigInt(cur.NextNumber);
      const items: SequenceNextItem[] = [];
      for (let i = 0; i < count; i++) {
        const n = start + BigInt(i);
        items.push({ Value: this.formatValue(seq.Prefix, seq.Suffix, seq.Padding, n), Number: Number(n) });
      }
      return this.buildNextResult(seq, items, generatedAt);
    }

    // Allocate range
    const { rangeStart, rangeEnd } = await this.allocateRangeAtomic(seq, count);
    const items: SequenceNextItem[] = [];
    for (let i = 0; i < count; i++) {
      const n = rangeStart + BigInt(i);
      items.push({ Value: this.formatValue(seq.Prefix, seq.Suffix, seq.Padding, n), Number: Number(n) });
    }

    // Persist idempotency record (strong consistency)
    if (idemKey) {
      const { default: SequenceIdempotency } = await import('./sequence_idempotency');
      const ttlDays = this.resolveIdempotencyTtlDays();
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
          return this.buildNextResult(seq, items, generatedAt);
        } catch {
          expiredIdempotencyHitId = undefined;
        }
      }

      try {
        await SequenceIdempotency.Create(payload as any);
      } catch (err) {
        const hit = await this.findIdempotencyHit(seq.Id, idemKey);
        if (hit?.Id) {
          if (!this.isIdempotencyExpired(hit)) {
            this.assertIdempotencyRequestMatch(hit, count, dryRun);
            const replayItems = this.buildItemsFromIdempotencyHit(seq, hit, count);
            return this.buildNextResult(seq, replayItems, generatedAt);
          }

          try {
            await SequenceIdempotency.UpdateById(String((hit as any).Id || ''), payload as any, ['Id'] as any);
          } catch (replaceErr) {
            const replayHit = await this.findIdempotencyHit(seq.Id, idemKey);
            if (replayHit?.Id && !this.isIdempotencyExpired(replayHit)) {
              this.assertIdempotencyRequestMatch(replayHit, count, dryRun);
              const replayItems = this.buildItemsFromIdempotencyHit(seq, replayHit, count);
              return this.buildNextResult(seq, replayItems, generatedAt);
            }
            throw replaceErr;
          }
          return this.buildNextResult(seq, items, generatedAt);
        }
        throw err;
      }
    }

    return this.buildNextResult(seq, items, generatedAt);
  }

  static async CleanupIdempotency(params?: SequenceCleanupIdempotencyParams): Promise<SequenceCleanupIdempotencyResult> {
    const { default: SequenceIdempotency } = await import('./sequence_idempotency');
    const olderThan = String(params?.OlderThan ?? '').trim();
    let cutoff: Date;
    if (!olderThan) {
      cutoff = new Date();
    } else {
      if (!/^\d{4}-\d{2}-\d{2}$/.test(olderThan)) {
        throw new ChoysumError({ domain: 'base', code: 'InvalidArgument', message: 'OlderThan must be YYYY-MM-DD' }).withGrpcCode(GrpcCode.InvalidArgument);
      }
      cutoff = new Date(`${olderThan}T00:00:00.000Z`);
    }
    const deleted = await SequenceIdempotency.Delete(['ExpiresAt', '<', cutoff] as any);
    return { Deleted: Number(deleted) || 0 };
  }
}

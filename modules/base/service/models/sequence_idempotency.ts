// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { Constraint } from '@/core/service/api/constraint';
import { GrpcCode, ChoysumError } from '@/core/service/error';
import Company from './company';
import Sequence from './sequence';
import { asRefId } from './_refs';
import { parsePositiveInt, parseBigInt } from './_normalizers';

@Model('SequenceIdempotency', { companyScoped: true })
export default class SequenceIdempotency extends BaseModel {
  @Field({ type: 'ManyToOne', relation: { targetModel: () => Company }, column: { index: true } })
  CompanyId?: Company;

  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => Sequence },
    column: { notNull: true, uniqueIndex: 'uidx_base_sequence_idem_seq_key', index: true },
  })
  SequenceId: Sequence;

  @Field({ type: 'varchar', column: { size: 100, notNull: true } })
  CodeSnapshot: string;

  @Field({ type: 'jsonobject', column: { notNull: true } })
  FormatSnapshot: Record<string, any>;

  @Field({ type: 'varchar', column: { size: 200, notNull: true, uniqueIndex: 'uidx_base_sequence_idem_seq_key', index: true } })
  IdempotencyKey: string;

  @Field({ type: 'int', column: { notNull: true } })
  Count: number;

  @Field({ type: 'boolean', column: { notNull: true, default: () => false } })
  DryRun: boolean;

  @Field({ type: 'bigint', column: { notNull: true } })
  RangeStart: bigint;

  @Field({ type: 'bigint', column: { notNull: true } })
  RangeEnd: bigint;

  @Field({ type: 'varchar', column: { size: 128 } })
  RequestHash?: string;

  @Field({ type: 'datetime', column: { notNull: true, index: true } })
  ExpiresAt: Date;

  private static async validateWriteEntity(values: Record<string, any>, existing?: any): Promise<void> {
    const candidate = {
      SequenceId: Object.prototype.hasOwnProperty.call(values, 'SequenceId') ? values.SequenceId : existing?.SequenceId,
      CompanyId: Object.prototype.hasOwnProperty.call(values, 'CompanyId') ? values.CompanyId : existing?.CompanyId,
      Count: Object.prototype.hasOwnProperty.call(values, 'Count') ? values.Count : existing?.Count,
      RangeStart: Object.prototype.hasOwnProperty.call(values, 'RangeStart') ? values.RangeStart : existing?.RangeStart,
      RangeEnd: Object.prototype.hasOwnProperty.call(values, 'RangeEnd') ? values.RangeEnd : existing?.RangeEnd,
    };

    const sequenceId = asRefId(candidate.SequenceId);
    if (!sequenceId) {
      throw new ChoysumError({ domain: 'base', code: 'InvalidArgument', message: 'SequenceId is required' }).withGrpcCode(GrpcCode.InvalidArgument);
    }

    const count = parsePositiveInt(candidate.Count, 'Count');
    const rangeStart = parseBigInt(candidate.RangeStart, 'RangeStart');
    const rangeEnd = parseBigInt(candidate.RangeEnd, 'RangeEnd');
    const expectedRangeEnd = rangeStart + BigInt(count) - 1n;
    if (rangeEnd !== expectedRangeEnd) {
      throw new ChoysumError({
        domain: 'base',
        code: 'InvalidArgument',
        message: 'RangeEnd must equal RangeStart + Count - 1',
      }).withGrpcCode(GrpcCode.InvalidArgument);
    }

    const sequence = await Sequence.Browse(sequenceId, ['Id', 'CompanyId'] as any);
    const sequenceCompanyId = asRefId((sequence as any)?.CompanyId);
    const companyId = asRefId(candidate.CompanyId) ?? null;
    if (companyId !== sequenceCompanyId) {
      throw new ChoysumError({
        domain: 'base',
        code: 'InvalidArgument',
        message: 'CompanyId must match Sequence.CompanyId',
      }).withGrpcCode(GrpcCode.InvalidArgument);
    }
  }

  @Constraint<SequenceIdempotency>(['SequenceId', 'CompanyId', 'Count', 'RangeStart', 'RangeEnd'])
  static async validateSequenceIdempotencyConstraint(self: SequenceIdempotency, ctx: any): Promise<void> {
    const current = (ctx?.current || {}) as Record<string, any>;
    await SequenceIdempotency.validateWriteEntity(self as any, current);
  }
}

// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { Constraint } from '@/core/service/api/constraint';
import { raiseDomainError } from '@/core/service/error';
import { normalizeRefId, parseBigInt, parsePositiveInt } from '@/core/service/utils/normalization';
import Company from './company';
import Sequence from './sequence';
import { mapNormalizationToBase } from './_normalizers';

@Model('SequenceIdempotency', { companyScoped: true })
export default class SequenceIdempotency extends BaseModel {
  @Field({ type: 'ManyToOne', relation: { targetModel: () => Company }, index: true})
  CompanyId?: Company;

  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => Sequence },
    notNull: true, uniqueIndex: 'uidx_base_sequence_idem_seq_key', index: true,
  })
  SequenceId: Sequence;

  @Field({ type: 'varchar', size: 100, notNull: true})
  CodeSnapshot: string;

  @Field({ type: 'jsonobject', notNull: true})
  FormatSnapshot: Record<string, any>;

  @Field({ type: 'varchar', size: 200, notNull: true, uniqueIndex: 'uidx_base_sequence_idem_seq_key', index: true})
  IdempotencyKey: string;

  @Field({ type: 'int', notNull: true})
  Count: number;

  @Field({ type: 'boolean', notNull: true, default: () => false})
  DryRun: boolean;

  @Field({ type: 'bigint', notNull: true})
  RangeStart: bigint;

  @Field({ type: 'bigint', notNull: true})
  RangeEnd: bigint;

  @Field({ type: 'varchar', size: 128})
  RequestHash?: string;

  @Field({ type: 'datetime', notNull: true, index: true})
  ExpiresAt: Date;

  private static async validateWriteEntity(values: Record<string, any>): Promise<void> {
    const sequenceId = normalizeRefId(values.SequenceId);
    if (!sequenceId) {
      raiseDomainError('base', 'InvalidArgument', 'SequenceId is required');
    }

    const count = mapNormalizationToBase(
      () => parsePositiveInt(values.Count),
      () => 'Count must be an integer >= 1'
    );
    const rangeStart = mapNormalizationToBase(
      () => parseBigInt(values.RangeStart),
      () => 'RangeStart must be a valid integer'
    );
    const rangeEnd = mapNormalizationToBase(
      () => parseBigInt(values.RangeEnd),
      () => 'RangeEnd must be a valid integer'
    );
    const expectedRangeEnd = rangeStart + BigInt(count) - 1n;
    if (rangeEnd !== expectedRangeEnd) {
      raiseDomainError('base', 'InvalidArgument', 'RangeEnd must equal RangeStart + Count - 1');
    }

    const sequence = await Sequence.Browse(sequenceId, ['Id', 'CompanyId'] as any);
    const sequenceCompanyId = normalizeRefId((sequence as any)?.CompanyId);
    const companyId = normalizeRefId(values.CompanyId) ?? null;
    if (companyId !== sequenceCompanyId) {
      raiseDomainError('base', 'InvalidArgument', 'CompanyId must match Sequence.CompanyId');
    }
  }

  @Constraint<SequenceIdempotency>(['SequenceId', 'CompanyId', 'Count', 'RangeStart', 'RangeEnd'])
  async validateSequenceIdempotencyConstraint(): Promise<void> {
    await SequenceIdempotency.validateWriteEntity(this as any);
  }
}

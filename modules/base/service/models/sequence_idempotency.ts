// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { createTranslate } from '@/core/service/i18n';
import { Constraint } from '@/core/service/api/constraint';
import { raiseDomainError } from '@/core/service/error';
import { normalizeRefId, parseBigInt, parsePositiveInt } from '@/core/service/utils/normalization';
import Company from './company';
import Sequence from './sequence';
import { mapNormalizationToBase } from './_normalizers';

const { _t } = createTranslate('base');

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
      raiseDomainError('base', 'InvalidArgument', _t('SequenceId is required', { scope: 'service/models/sequence_idempotency' }));
    }

    const count = mapNormalizationToBase(
      () => parsePositiveInt(values.Count),
      () => _t('Count must be an integer >= 1', { scope: 'service/models/sequence_idempotency' })
    );
    const rangeStart = mapNormalizationToBase(
      () => parseBigInt(values.RangeStart),
      () => _t('RangeStart must be a valid integer', { scope: 'service/models/sequence_idempotency' })
    );
    const rangeEnd = mapNormalizationToBase(
      () => parseBigInt(values.RangeEnd),
      () => _t('RangeEnd must be a valid integer', { scope: 'service/models/sequence_idempotency' })
    );
    const expectedRangeEnd = rangeStart + BigInt(count) - 1n;
    if (rangeEnd !== expectedRangeEnd) {
      raiseDomainError('base', 'InvalidArgument', _t('RangeEnd must equal RangeStart + Count - 1', { scope: 'service/models/sequence_idempotency' }));
    }

    const sequence = await Sequence.Browse(sequenceId, ['Id', 'CompanyId'] as any);
    const sequenceCompanyId = normalizeRefId((sequence as any)?.CompanyId);
    const companyId = normalizeRefId(values.CompanyId) ?? null;
    if (companyId !== sequenceCompanyId) {
      raiseDomainError('base', 'InvalidArgument', _t('CompanyId must match Sequence.CompanyId', { scope: 'service/models/sequence_idempotency' }));
    }
  }

  @Constraint<SequenceIdempotency>(['SequenceId', 'CompanyId', 'Count', 'RangeStart', 'RangeEnd'])
  async validateSequenceIdempotencyConstraint(): Promise<void> {
    await SequenceIdempotency.validateWriteEntity(this as any);
  }
}

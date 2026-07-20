// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { Constraint } from '@/core/service/api/constraint';
import { raiseDomainError } from '@/core/service/error';
import { normalizeRefId, parseBigInt, parsePositiveInt } from '@/core/service/utils/normalization';
import { _t, _lt } from '../i18n';
import Company from './company';
import Sequence from './sequence';
import { mapNormalizationToBase } from './_normalizers';

@Model('SequenceIdempotency', { companyScoped: true })
export default class SequenceIdempotency extends BaseModel {
  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => Company },
    index: true,
    string: _lt('Company', { scope: 'base.model.SequenceIdempotency.fields' }),
  })
  CompanyId?: Company;

  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => Sequence },
    notNull: true,
    uniqueIndex: 'uidx_base_sequence_idem_seq_key',
    index: true,
    string: _lt('Sequence', { scope: 'base.model.SequenceIdempotency.fields' }),
  })
  SequenceId: Sequence;

  @Field({
    type: 'varchar',
    size: 100,
    notNull: true,
    string: _lt('Code Snapshot', { scope: 'base.model.SequenceIdempotency.fields' }),
  })
  CodeSnapshot: string;

  @Field({
    type: 'jsonobject',
    notNull: true,
    string: _lt('Format Snapshot', { scope: 'base.model.SequenceIdempotency.fields' }),
  })
  FormatSnapshot: Record<string, any>;

  @Field({
    type: 'varchar',
    size: 200,
    notNull: true,
    uniqueIndex: 'uidx_base_sequence_idem_seq_key',
    index: true,
    string: _lt('Idempotency Key', { scope: 'base.model.SequenceIdempotency.fields' }),
  })
  IdempotencyKey: string;

  @Field({
    type: 'int',
    notNull: true,
    string: _lt('Count', { scope: 'base.model.SequenceIdempotency.fields' }),
  })
  Count: number;

  @Field({
    type: 'boolean',
    notNull: true,
    default: () => false,
    string: _lt('Dry Run', { scope: 'base.model.SequenceIdempotency.fields' }),
  })
  DryRun: boolean;

  @Field({
    type: 'bigint',
    notNull: true,
    string: _lt('Start', { scope: 'base.model.SequenceIdempotency.fields' }),
  })
  RangeStart: bigint;

  @Field({
    type: 'bigint',
    notNull: true,
    string: _lt('End', { scope: 'base.model.SequenceIdempotency.fields' }),
  })
  RangeEnd: bigint;

  @Field({
    type: 'varchar',
    size: 128,
    string: _lt('Request Hash', { scope: 'base.model.SequenceIdempotency.fields' }),
  })
  RequestHash?: string;

  @Field({
    type: 'datetime',
    notNull: true,
    index: true,
    string: _lt('Expires At', { scope: 'base.model.SequenceIdempotency.fields' }),
  })
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

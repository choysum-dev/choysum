// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { Constraint } from '@/core/service/api/constraint';
import Company from './company';
import { normalizeRefId } from '@/core/service/utils/normalization';
import { normalizeCodeRequired } from './_normalizers';
import { writeConstraintFields } from './_constraint_helpers';
import { nextSequence } from './_sequence_next';
import { cleanupSequenceIdempotency } from './_sequence_cleanup';

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
    values.CompanyScopeKey = normalizeRefId(values.CompanyId) || '__GLOBAL__';
  }

  @Constraint<Sequence>(['Code', 'CompanyId'])
  static validateSequenceConstraint(self: Sequence, ctx: any): void {
    Sequence.validateWriteEntity(self as any);
    writeConstraintFields(self as any, ctx, ['Code'], { forceOnCreate: true });
    writeConstraintFields(self as any, ctx, [], {
      forceOnCreate: true,
      triggerFields: ['CompanyId', 'CompanyScopeKey'],
      targetField: 'CompanyScopeKey',
    });
  }

  static async Next(params: SequenceNextParams): Promise<SequenceNextResult> {
    return nextSequence(this, params);
  }

  static async CleanupIdempotency(params?: SequenceCleanupIdempotencyParams): Promise<SequenceCleanupIdempotencyResult> {
    return cleanupSequenceIdempotency(params);
  }
}

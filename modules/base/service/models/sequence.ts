// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { Constraint } from '@/core/service/api/constraint';
import Company from './company';
import { normalizeRefId } from '@/core/service/utils/normalization';
import { normalizeCodeRequired } from './_normalizers';
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
  @Field({ type: 'varchar', size: 100, notNull: true, index: true})
  Name: string;

  @Field({ type: 'varchar', size: 100, notNull: true, index: true, uniqueIndex: 'uidx_base_sequence_scope_code'})
  Code: string;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => Company }, index: true})
  CompanyId?: Company;

  @Field({ type: 'varchar', size: 20, notNull: true, default: () => '__GLOBAL__', index: true, uniqueIndex: 'uidx_base_sequence_scope_code'})
  CompanyScopeKey: string;

  @Field({ type: 'varchar', size: 64})
  Prefix?: string;

  @Field({ type: 'varchar', size: 64})
  Suffix?: string;

  @Field({ type: 'int', notNull: true, default: () => 5})
  Padding: number;

  @Field({ type: 'bigint', notNull: true, default: () => 1})
  NextNumber: bigint;

  @Field({ type: 'boolean', notNull: true, default: () => true, index: true})
  IsActive: boolean;

  @Constraint<Sequence>(['Code', 'CompanyId'])
  validateSequenceConstraint(): void {
    (this as any).Code = normalizeCodeRequired(this.Code, { uppercase: false });
    // CompanyScopeKey is always derived from CompanyId.
    (this as any).CompanyScopeKey = normalizeRefId(this.CompanyId) || '__GLOBAL__';
  }

  static async Next(params: SequenceNextParams): Promise<SequenceNextResult> {
    return nextSequence(this, params);
  }

  static async CleanupIdempotency(params?: SequenceCleanupIdempotencyParams): Promise<SequenceCleanupIdempotencyResult> {
    return cleanupSequenceIdempotency(params);
  }
}

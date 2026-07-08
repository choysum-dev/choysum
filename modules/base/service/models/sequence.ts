// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { Constraint } from '@/core/service/api/constraint';
import Company from './company';
import { normalizeCompanyScopeKey } from './_refs';
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

  static async Next(params: SequenceNextParams): Promise<SequenceNextResult> {
    return nextSequence(this, params);
  }

  static async CleanupIdempotency(params?: SequenceCleanupIdempotencyParams): Promise<SequenceCleanupIdempotencyResult> {
    return cleanupSequenceIdempotency(params);
  }
}

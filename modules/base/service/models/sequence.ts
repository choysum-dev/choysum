// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { Constraint } from '@/core/service/api/constraint';
import { _lt } from '../i18n';
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

@Model('Sequence', { companyField: 'CompanyId' })
export default class Sequence extends BaseModel {
  @Field({
    type: 'varchar',
    size: 100,
    notNull: true,
    translate: true,
    index: 'trigram',
    string: _lt('Name', { scope: 'base.model.Sequence.fields' }),
  })
  Name: string;

  @Field({
    type: 'varchar',
    size: 100,
    notNull: true,
    index: true,
    uniqueIndex: 'uidx_base_sequence_scope_code',
    string: _lt('Code', { scope: 'base.model.Sequence.fields' }),
    help: _lt('Stable id passed to Sequence.Next; unique per company or globally.', {
      scope: 'base.model.Sequence.fields',
    }),
  })
  Code: string;

  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => Company },
    index: true,
    string: _lt('Company', { scope: 'base.model.Sequence.fields' }),
    help: _lt('Leave empty for a global sequence shared by all companies.', {
      scope: 'base.model.Sequence.fields',
    }),
  })
  CompanyId?: Company;

  @Field({
    type: 'varchar',
    size: 20,
    notNull: true,
    default: () => '__GLOBAL__',
    index: true,
    uniqueIndex: 'uidx_base_sequence_scope_code',
  })
  CompanyScopeKey: string;

  @Field({
    type: 'varchar',
    size: 64,
    string: _lt('Prefix', { scope: 'base.model.Sequence.fields' }),
    help: _lt('Literal text prepended to the generated number.', {
      scope: 'base.model.Sequence.fields',
    }),
  })
  Prefix?: string;

  @Field({
    type: 'varchar',
    size: 64,
    string: _lt('Suffix', { scope: 'base.model.Sequence.fields' }),
    help: _lt('Literal text appended to the generated number.', {
      scope: 'base.model.Sequence.fields',
    }),
  })
  Suffix?: string;

  @Field({
    type: 'int',
    notNull: true,
    default: () => 5,
    string: _lt('Padding Length', { scope: 'base.model.Sequence.fields' }),
    help: _lt('Zero-padded numeric width between prefix and suffix.', {
      scope: 'base.model.Sequence.fields',
    }),
  })
  Padding: number;

  @Field({
    type: 'bigint',
    notNull: true,
    default: () => 1,
    copy: false,
    string: _lt('Next Number', { scope: 'base.model.Sequence.fields' }),
    help: _lt('Next issued number; lowering it may reuse already-assigned values.', {
      scope: 'base.model.Sequence.fields',
    }),
  })
  NextNumber: bigint;

  @Field({
    type: 'boolean',
    notNull: true,
    default: () => true,
    index: true,
    string: _lt('Active', { scope: 'base.model.Sequence.fields' }),
  })
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

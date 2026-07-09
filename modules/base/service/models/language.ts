// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { Constraint } from '@/core/service/api/constraint';
import Locale from './locale';
import { normalizeDirection } from './_normalizers';
import { writeConstraintFields } from '@/core/service/utils/constraint_writeback';

@Model('Language')
export default class Language extends BaseModel {
  @Field({ type: 'varchar', column: { size: 100, notNull: true, index: true } })
  Name: string;

  @Field({ type: 'varchar', column: { size: 16, notNull: true, unique: true, index: true } })
  Code: string;

  @Field({ type: 'boolean', column: { notNull: true, default: () => true, index: true } })
  IsActive: boolean;

  @Field({
    type: 'selection',
    selection: [
      { value: 'ltr', label: 'ltr' },
      { value: 'rtl', label: 'rtl' },
    ],
    column: { size: 8 },
  })
  Direction?: 'ltr' | 'rtl';

  @Field({ type: 'ManyToOne', relation: { targetModel: () => Locale }, column: { index: true } })
  DefaultLocaleId?: Locale;

  private static validateEntity(values: Record<string, any>): void {
    if (Object.prototype.hasOwnProperty.call(values, 'Direction')) {
      values.Direction = normalizeDirection(values.Direction);
    }
  }

  @Constraint<Language>(['Direction'])
  static validateLanguageConstraint(self: Language, ctx: any): void {
    Language.validateEntity(self as any);
    writeConstraintFields(self as any, ctx, ['Direction']);
  }
}

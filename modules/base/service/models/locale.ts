// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { Constraint } from '@/core/service/api/constraint';
import { fail, normalizeCurrencySymbolPosition, normalizeCurrencySymbolSpacing } from './_normalizers';
import { writeConstraintFields } from './_constraint_helpers';

@Model('Locale')
export default class Locale extends BaseModel {
  @Field({ type: 'varchar', column: { size: 100, notNull: true, index: true } })
  Name: string;

  @Field({ type: 'varchar', column: { size: 16, notNull: true, unique: true, index: true } })
  Code: string;

  @Field({ type: 'boolean', column: { notNull: true, default: () => true, index: true } })
  IsActive: boolean;

  @Field({ type: 'varchar', column: { size: 8 } })
  DecimalSeparator?: string;

  @Field({ type: 'varchar', column: { size: 8 } })
  ThousandSeparator?: string;

  @Field({ type: 'varchar', column: { size: 32 } })
  DateFormat?: string;

  @Field({ type: 'varchar', column: { size: 32 } })
  TimeFormat?: string;

  @Field({ type: 'int' })
  FirstDayOfWeek?: number;

  @Field({
    type: 'selection',
    selection: [
      { value: 'before', label: 'before' },
      { value: 'after', label: 'after' },
    ],
    column: { size: 16, default: () => 'before' },
  })
  CurrencySymbolPosition?: 'before' | 'after';

  @Field({ type: 'boolean', column: { default: () => false } })
  CurrencySymbolSpacing?: boolean;

  private static validateEntity(values: Record<string, any>): void {
    if (Object.prototype.hasOwnProperty.call(values, 'CurrencySymbolPosition')) {
      values.CurrencySymbolPosition = normalizeCurrencySymbolPosition(values.CurrencySymbolPosition);
    }
    if (Object.prototype.hasOwnProperty.call(values, 'CurrencySymbolSpacing')) {
      values.CurrencySymbolSpacing = normalizeCurrencySymbolSpacing(values.CurrencySymbolSpacing);
    }
  }

  @Constraint<Locale>(['CurrencySymbolPosition', 'CurrencySymbolSpacing'])
  static validateLocaleConstraint(self: Locale, ctx: any): void {
    Locale.validateEntity(self as any);
    writeConstraintFields(self as any, ctx, ['CurrencySymbolPosition', 'CurrencySymbolSpacing']);
  }
}

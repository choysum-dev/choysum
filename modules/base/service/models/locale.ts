// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { Constraint } from '@/core/service/api/constraint';
import { fail } from './_normalizers';

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

  private static normalizeCurrencySymbolPosition(value: unknown): 'before' | 'after' {
    if (value === undefined || value === null || value === '') return 'before';
    if (value === 'before' || value === 'after') return value;
    fail('CurrencySymbolPosition must be before or after');
  }

  private static normalizeCurrencySymbolSpacing(value: unknown): boolean {
    if (value === undefined || value === null || value === '') return false;
    return Boolean(value);
  }

  private static validateEntity(values: Record<string, any>): void {
    if (Object.prototype.hasOwnProperty.call(values, 'CurrencySymbolPosition')) {
      values.CurrencySymbolPosition = this.normalizeCurrencySymbolPosition(values.CurrencySymbolPosition);
    }
    if (Object.prototype.hasOwnProperty.call(values, 'CurrencySymbolSpacing')) {
      values.CurrencySymbolSpacing = this.normalizeCurrencySymbolSpacing(values.CurrencySymbolSpacing);
    }
  }

  @Constraint<Locale>(['CurrencySymbolPosition', 'CurrencySymbolSpacing'])
  static validateLocaleConstraint(self: Locale, ctx: any): void {
    const values = (ctx?.values || {}) as Record<string, any>;
    Locale.validateEntity(self as any);

    if (Object.prototype.hasOwnProperty.call(values, 'CurrencySymbolPosition')) {
      values.CurrencySymbolPosition = self.CurrencySymbolPosition;
    }
    if (Object.prototype.hasOwnProperty.call(values, 'CurrencySymbolSpacing')) {
      values.CurrencySymbolSpacing = self.CurrencySymbolSpacing;
    }
  }
}

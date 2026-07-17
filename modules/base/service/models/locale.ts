// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { Constraint } from '@/core/service/api/constraint';
import { createTranslate } from '@/core/service/i18n';
import { normalizeCurrencySymbolPosition, normalizeCurrencySymbolSpacing } from './_normalizers';

const { _td } = createTranslate('base');

@Model('Locale')
export default class Locale extends BaseModel {
  @Field({ type: 'varchar', size: 100, notNull: true, index: true})
  Name: string;

  @Field({ type: 'varchar', size: 16, notNull: true, unique: true, index: true})
  Code: string;

  @Field({ type: 'boolean', notNull: true, default: () => true, index: true})
  IsActive: boolean;

  @Field({ type: 'varchar', size: 8})
  DecimalSeparator?: string;

  @Field({ type: 'varchar', size: 8})
  ThousandSeparator?: string;

  @Field({ type: 'varchar', size: 32})
  DateFormat?: string;

  @Field({ type: 'varchar', size: 32})
  TimeFormat?: string;

  @Field({ type: 'int' })
  FirstDayOfWeek?: number;

  @Field({
    type: 'selection',
    selection: [
      { value: 'before', label: _td('Before amount', { scope: 'base.Locale.CurrencySymbolPosition.before' }) },
      { value: 'after', label: _td('After amount', { scope: 'base.Locale.CurrencySymbolPosition.after' }) },
    ],
    size: 16, default: () => 'before',
  })
  CurrencySymbolPosition?: 'before' | 'after';

  @Field({ type: 'boolean', default: () => false})
  CurrencySymbolSpacing?: boolean;

  @Constraint<Locale>(['CurrencySymbolPosition', 'CurrencySymbolSpacing'])
  validateLocaleConstraint(): void {
    if (this.CurrencySymbolPosition !== undefined) {
      (this as any).CurrencySymbolPosition = normalizeCurrencySymbolPosition(this.CurrencySymbolPosition);
    }
    if (this.CurrencySymbolSpacing !== undefined) {
      (this as any).CurrencySymbolSpacing = normalizeCurrencySymbolSpacing(this.CurrencySymbolSpacing);
    }
  }
}

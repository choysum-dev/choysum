// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { Constraint } from '@/core/service/api/constraint';
import { _lt } from '../i18n';
import { normalizeCurrencySymbolPosition, normalizeCurrencySymbolSpacing } from './_normalizers';

@Model('Locale')
export default class Locale extends BaseModel {
  @Field({
    type: 'varchar',
    size: 100,
    notNull: true,
    index: true,
    string: _lt('Name', { scope: 'base.model.Locale.fields' }),
  })
  Name: string;

  @Field({
    type: 'varchar',
    size: 16,
    notNull: true,
    unique: true,
    index: true,
    string: _lt('Code', { scope: 'base.model.Locale.fields' }),
  })
  Code: string;

  @Field({
    type: 'boolean',
    notNull: true,
    default: () => true,
    index: true,
    string: _lt('Active', { scope: 'base.model.Locale.fields' }),
  })
  IsActive: boolean;

  @Field({
    type: 'varchar',
    size: 8,
    string: _lt('Decimal Separator', { scope: 'base.model.Locale.fields' }),
  })
  DecimalSeparator?: string;

  @Field({
    type: 'varchar',
    size: 8,
    string: _lt('Thousands Separator', { scope: 'base.model.Locale.fields' }),
  })
  ThousandSeparator?: string;

  @Field({
    type: 'varchar',
    size: 32,
    string: _lt('Date Format', { scope: 'base.model.Locale.fields' }),
  })
  DateFormat?: string;

  @Field({
    type: 'varchar',
    size: 32,
    string: _lt('Time Format', { scope: 'base.model.Locale.fields' }),
  })
  TimeFormat?: string;

  @Field({
    type: 'int',
    string: _lt('First Day of Week', { scope: 'base.model.Locale.fields' }),
  })
  FirstDayOfWeek?: number;

  @Field({
    type: 'selection',
    selection: [
      { value: 'before', label: _lt('Before amount', { scope: 'base.model.Locale.fields' }) },
      { value: 'after', label: _lt('After amount', { scope: 'base.model.Locale.fields' }) },
    ],
    size: 16,
    default: () => 'before',
    string: _lt('Currency Symbol Position', { scope: 'base.model.Locale.fields' }),
  })
  CurrencySymbolPosition?: 'before' | 'after';

  @Field({
    type: 'boolean',
    default: () => false,
    string: _lt('Symbol Spacing', { scope: 'base.model.Locale.fields' }),
  })
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

// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { Constraint } from '@/core/service/api/constraint';
import { _lt } from '../i18n';
import { normalizeCurrencySymbolPosition, normalizeCurrencySymbolSpacing, normalizeDirection } from './_normalizers';

@Model('Language')
export default class Language extends BaseModel {
  @Field({
    type: 'varchar',
    size: 100,
    notNull: true,
    index: true,
    string: _lt('Name', { scope: 'base.model.Language.fields' }),
  })
  Name: string;

  @Field({
    type: 'varchar',
    size: 16,
    notNull: true,
    unique: true,
    index: true,
    string: _lt('Code', { scope: 'base.model.Language.fields' }),
  })
  Code: string;

  @Field({
    type: 'boolean',
    notNull: true,
    default: () => true,
    index: true,
    string: _lt('Active', { scope: 'base.model.Language.fields' }),
  })
  IsActive: boolean;

  @Field({
    type: 'selection',
    selection: [
      { value: 'ltr', label: _lt('Left to right', { scope: 'base.model.Language.fields' }) },
      { value: 'rtl', label: _lt('Right to left', { scope: 'base.model.Language.fields' }) },
    ],
    size: 8,
    string: _lt('Direction', { scope: 'base.model.Language.fields' }),
  })
  Direction?: 'ltr' | 'rtl';

  @Field({
    type: 'varchar',
    size: 8,
    string: _lt('Decimal Separator', { scope: 'base.model.Language.fields' }),
  })
  DecimalSeparator?: string;

  @Field({
    type: 'varchar',
    size: 8,
    string: _lt('Thousands Separator', { scope: 'base.model.Language.fields' }),
  })
  ThousandSeparator?: string;

  @Field({
    type: 'varchar',
    size: 32,
    default: () => '[3,0]',
    string: _lt('Grouping', { scope: 'base.model.Language.fields' }),
  })
  Grouping?: string;

  @Field({
    type: 'varchar',
    size: 32,
    string: _lt('Date Format', { scope: 'base.model.Language.fields' }),
  })
  DateFormat?: string;

  @Field({
    type: 'varchar',
    size: 32,
    string: _lt('Time Format', { scope: 'base.model.Language.fields' }),
  })
  TimeFormat?: string;

  @Field({
    type: 'int',
    string: _lt('First Day of Week', { scope: 'base.model.Language.fields' }),
  })
  FirstDayOfWeek?: number;

  @Field({
    type: 'selection',
    selection: [
      { value: 'before', label: _lt('Before amount', { scope: 'base.model.Language.fields' }) },
      { value: 'after', label: _lt('After amount', { scope: 'base.model.Language.fields' }) },
    ],
    size: 16,
    default: () => 'before',
    string: _lt('Currency Symbol Position', { scope: 'base.model.Language.fields' }),
  })
  CurrencySymbolPosition?: 'before' | 'after';

  @Field({
    type: 'boolean',
    default: () => false,
    string: _lt('Symbol Spacing', { scope: 'base.model.Language.fields' }),
  })
  CurrencySymbolSpacing?: boolean;

  @Constraint<Language>(['Direction', 'CurrencySymbolPosition', 'CurrencySymbolSpacing'])
  validateLanguageConstraint(): void {
    if (this.Direction != null) {
      this.Direction = normalizeDirection(this.Direction) as any;
    }
    if (this.CurrencySymbolPosition !== undefined) {
      (this as any).CurrencySymbolPosition = normalizeCurrencySymbolPosition(this.CurrencySymbolPosition);
    }
    if (this.CurrencySymbolSpacing !== undefined) {
      (this as any).CurrencySymbolSpacing = normalizeCurrencySymbolSpacing(this.CurrencySymbolSpacing);
    }
  }
}

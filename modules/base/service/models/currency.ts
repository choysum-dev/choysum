// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { Constraint } from '@/core/service/api/constraint';
import { normalizeDecimalDigits, normalizePositiveDecimalString } from '@/core/service/utils/normalization';
import { _t, _lt } from '../i18n';
import { mapNormalizationToBase, assertCodeRequired } from './_normalizers';
import { convertCurrency } from './_currency_convert';

export type CurrencyConvertRatePolicy = {
  Mode?: 'exact' | 'latest_before';
  AllowFallbackToGlobal?: boolean;
};

export type CurrencyConvertRounding = {
  Mode?: 'currency' | 'none';
  ToDecimalDigitsOverride?: number;
};

export type CurrencyConvertParams = {
  CompanyId: string;
  Date: string;
  Amount: any;
  FromCurrencyId: string;
  ToCurrencyId: string;
  RatePolicy?: CurrencyConvertRatePolicy;
  Rounding?: CurrencyConvertRounding;
};

export type CurrencyConvertResult = { Amount: any; RateUsed?: any; Warnings?: string[] };

@Model('Currency')
export default class Currency extends BaseModel {
  @Field({
    type: 'varchar',
    size: 100,
    notNull: true,
    translate: true,
    index: 'trigram',
    string: _lt('Name', { scope: 'base.model.Currency.fields' }),
  })
  Name: string;

  @Field({
    type: 'varchar',
    size: 8,
    notNull: true,
    unique: true,
    index: true,
    string: _lt('Code', { scope: 'base.model.Currency.fields' }),
    help: _lt('ISO 4217 code (e.g. USD, EUR).', { scope: 'base.model.Currency.fields' }),
  })
  Code: string;

  @Field({
    type: 'varchar',
    size: 16,
    string: _lt('Symbol', { scope: 'base.model.Currency.fields' }),
  })
  Symbol?: string;

  @Field({
    type: 'int',
    notNull: true,
    default: () => 2,
    string: _lt('Decimal Digits', { scope: 'base.model.Currency.fields' }),
    help: _lt('Decimal places used when rounding amounts in this currency.', {
      scope: 'base.model.Currency.fields',
    }),
  })
  DecimalDigits: number;

  @Field({
    type: 'decimal',
    notNull: true,
    string: _lt('Rounding Precision', { scope: 'base.model.Currency.fields' }),
    help: _lt('Smallest rounding increment; must be greater than zero.', {
      scope: 'base.model.Currency.fields',
    }),
  })
  Rounding: any;

  @Field({
    type: 'boolean',
    notNull: true,
    default: () => true,
    index: true,
    string: _lt('Active', { scope: 'base.model.Currency.fields' }),
  })
  IsActive: boolean;

  @Constraint<Currency>(['Code', 'DecimalDigits', 'Rounding'])
  validateCurrencyConstraint(): void {
    this.Code = assertCodeRequired(this.Code as string);
    (this as any).DecimalDigits = mapNormalizationToBase(
      () => normalizeDecimalDigits(this.DecimalDigits),
      err =>
        err.code === 'required'
          ? _t('DecimalDigits is required', { scope: 'service/models/currency' })
          : _t('DecimalDigits must be a non-negative integer', { scope: 'service/models/currency' })
    );
    (this as any).Rounding = mapNormalizationToBase(
      () => normalizePositiveDecimalString(this.Rounding),
      err =>
        err.code === 'non_positive_decimal'
          ? _t('Rounding must be greater than 0', { scope: 'service/models/currency' })
          : _t('Rounding must be a valid decimal', { scope: 'service/models/currency' })
    );
  }

  static async Convert(params: CurrencyConvertParams): Promise<CurrencyConvertResult> {
    return convertCurrency(this, params);
  }
}

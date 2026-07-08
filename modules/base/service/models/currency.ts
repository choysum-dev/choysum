// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { Constraint } from '@/core/service/api/constraint';
import { normalizeCodeRequired, normalizeDecimalDigits, normalizePositiveDecimalString } from './_normalizers';
import { writeConstraintFields } from './_constraint_helpers';
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
  @Field({ type: 'varchar', column: { size: 100, notNull: true, index: true } })
  Name: string;

  @Field({ type: 'varchar', column: { size: 8, notNull: true, unique: true, index: true } })
  Code: string;

  @Field({ type: 'varchar', column: { size: 16 } })
  Symbol?: string;

  @Field({ type: 'int', column: { notNull: true, default: () => 2 } })
  DecimalDigits: number;

  @Field({ type: 'decimal', column: { notNull: true, precision: 38, scale: 18 } })
  Rounding: any;

  @Field({ type: 'boolean', column: { notNull: true, default: () => true, index: true } })
  IsActive: boolean;

  private static validateEntity(values: Record<string, any>): void {
    values.Code = normalizeCodeRequired(values.Code);
    values.DecimalDigits = normalizeDecimalDigits(values.DecimalDigits);
    values.Rounding = normalizePositiveDecimalString(values.Rounding, 'Rounding');
  }

  @Constraint<Currency>(['Code', 'DecimalDigits', 'Rounding'])
  static validateCurrencyConstraint(self: Currency, ctx: any): void {
    Currency.validateEntity(self as any);
    writeConstraintFields(self as any, ctx, ['Code', 'DecimalDigits', 'Rounding'], { forceOnCreate: true });
  }

  static async Convert(params: CurrencyConvertParams): Promise<CurrencyConvertResult> {
    return convertCurrency(this, params);
  }
}

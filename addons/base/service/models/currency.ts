// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Decimal, Field, Model } from '@/core/service';
import { Constraint } from '@/core/service/api/constraint';
import { GrpcCode, ChoysumError } from '@/core/service/error';

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

  private static normalizeDecimalDigits(value: any): number {
    if (value === undefined || value === null || value === '') {
      throw new ChoysumError({ domain: 'base', code: 'InvalidArgument', message: 'DecimalDigits is required' }).withGrpcCode(GrpcCode.InvalidArgument);
    }
    const n = Number(value);
    if (!Number.isFinite(n) || Math.floor(n) !== n || n < 0) {
      throw new ChoysumError({ domain: 'base', code: 'InvalidArgument', message: 'DecimalDigits must be a non-negative integer' }).withGrpcCode(
        GrpcCode.InvalidArgument
      );
    }
    return n;
  }

  private static normalizeCode(value: any): string {
    const code = String(value ?? '')
      .trim()
      .toUpperCase();
    if (!code) {
      throw new ChoysumError({ domain: 'base', code: 'InvalidArgument', message: 'Code is required' }).withGrpcCode(GrpcCode.InvalidArgument);
    }
    return code;
  }

  private static normalizePositiveDecimal(value: any, fieldName: string): string {
    try {
      if (value == null || value === '') throw new Error('required');
      const decimal = value instanceof Decimal ? value : new Decimal((value as any)?.$bigdecimal ?? value);
      if (!decimal.gt(0)) {
        throw new ChoysumError({ domain: 'base', code: 'InvalidArgument', message: `${fieldName} must be greater than 0` }).withGrpcCode(
          GrpcCode.InvalidArgument
        );
      }
      return decimal.toString();
    } catch (err: any) {
      if (err instanceof ChoysumError) throw err;
      throw new ChoysumError({ domain: 'base', code: 'InvalidArgument', message: `${fieldName} must be a valid decimal` }).withGrpcCode(GrpcCode.InvalidArgument);
    }
  }

  private static validateEntity(values: Record<string, any>): void {
    values.Code = this.normalizeCode(values.Code);
    values.DecimalDigits = this.normalizeDecimalDigits(values.DecimalDigits);
    values.Rounding = this.normalizePositiveDecimal(values.Rounding, 'Rounding');
  }

  @Constraint<Currency>(['Code', 'DecimalDigits', 'Rounding'])
  static validateCurrencyConstraint(self: Currency, ctx: any): void {
    const values = (ctx?.values || {}) as Record<string, any>;
    Currency.validateEntity(self as any);

    if (Object.prototype.hasOwnProperty.call(values, 'Code') || String(ctx?.mode || '') === 'create') {
      values.Code = self.Code;
    }
    if (Object.prototype.hasOwnProperty.call(values, 'DecimalDigits') || String(ctx?.mode || '') === 'create') {
      values.DecimalDigits = self.DecimalDigits;
    }
    if (Object.prototype.hasOwnProperty.call(values, 'Rounding') || String(ctx?.mode || '') === 'create') {
      values.Rounding = self.Rounding;
    }
  }

  private static parseDateString(value: unknown): string {
    const v = String(value ?? '').trim();
    if (!/^\d{4}-\d{2}-\d{2}$/.test(v)) {
      throw new ChoysumError({ domain: 'base', code: 'InvalidArgument', message: 'Date must be YYYY-MM-DD' }).withGrpcCode(GrpcCode.InvalidArgument);
    }
    return v;
  }

  private static normalizeRatePolicyMode(value: unknown): 'exact' | 'latest_before' {
    if (value === undefined || value === null || value === '') return 'latest_before';
    if (value === 'exact' || value === 'latest_before') return value;
    throw new ChoysumError({ domain: 'base', code: 'InvalidArgument', message: 'RatePolicy.Mode must be exact or latest_before' }).withGrpcCode(
      GrpcCode.InvalidArgument
    );
  }

  private static normalizeRoundingMode(value: unknown): 'currency' | 'none' {
    if (value === undefined || value === null || value === '') return 'currency';
    if (value === 'currency' || value === 'none') return value;
    throw new ChoysumError({ domain: 'base', code: 'InvalidArgument', message: 'Rounding.Mode must be currency or none' }).withGrpcCode(GrpcCode.InvalidArgument);
  }

  private static parseDecimalInput(value: any): Decimal {
    try {
      if (value == null) throw new Error('amount is required');

      if (value instanceof Decimal) return value;

      if (typeof value === 'number') {
        throw new Error('Amount must not be a number');
      }

      if (typeof value === 'object' && value && typeof value.$bigdecimal === 'string') {
        return new Decimal(String(value.$bigdecimal));
      }

      if (typeof value === 'string') {
        return new Decimal(value);
      }

      throw new Error('Invalid Amount');
    } catch {
      throw new ChoysumError({ domain: 'base', code: 'InvalidArgument', message: 'Invalid Amount' }).withGrpcCode(GrpcCode.InvalidArgument);
    }
  }

  private static toDateOnlyString(input: any): string {
    if (input instanceof Date) return input.toISOString().slice(0, 10);
    const s = String(input ?? '').trim();
    if (!s) return '';
    return s.length >= 10 ? s.slice(0, 10) : s;
  }

  private static async getRateRecord(opts: {
    companyId: string;
    currencyId: string;
    date: string;
    mode: 'exact' | 'latest_before';
    allowFallbackToGlobal: boolean;
  }): Promise<{ rec?: any; usedGlobal: boolean; usedFallbackDate: boolean }> {
    const { default: ExchangeRate } = await import('./exchange_rate');
    const { companyId, currencyId, date, mode } = opts;
    const searchOne = async (companyCond: any): Promise<any | undefined> => {
      if (mode === 'exact') {
        const rows = await ExchangeRate.Search(
          { And: [companyCond, ['CurrencyId', '=', currencyId], ['Date', '=', date]] } as any,
          {
            limit: 1,
            fields: ['Id', 'CurrencyId', 'CompanyId', 'Date', 'Rate'] as any,
          } as any
        );
        return rows?.[0] as any;
      }
      const rows = await ExchangeRate.Search(
        { And: [companyCond, ['CurrencyId', '=', currencyId], ['Date', '<=', date]] } as any,
        {
          limit: 1,
          orderBy: { field: 'Date', order: 'desc' } as any,
          fields: ['Id', 'CurrencyId', 'CompanyId', 'Date', 'Rate'] as any,
        } as any
      );
      return rows?.[0] as any;
    };

    // company scoped
    const recCompany = await searchOne(['CompanyId', '=', companyId]);
    if (recCompany) {
      const usedFallbackDate = mode === 'latest_before' && this.toDateOnlyString((recCompany as any).Date) !== date;
      return { rec: recCompany, usedGlobal: false, usedFallbackDate };
    }

    if (opts.allowFallbackToGlobal) {
      const recGlobal = await searchOne(['CompanyId', 'is', null]);
      if (recGlobal) {
        const usedFallbackDate = mode === 'latest_before' && this.toDateOnlyString((recGlobal as any).Date) !== date;
        return { rec: recGlobal, usedGlobal: true, usedFallbackDate };
      }
    }

    return { rec: undefined, usedGlobal: false, usedFallbackDate: false };
  }

  private static roundToCurrency(amount: Decimal, toCurrency: Currency, overrideDigits?: number): Decimal {
    const digits = Number.isFinite(overrideDigits as any)
      ? Math.max(0, Math.floor(overrideDigits as any))
      : Math.max(0, Math.floor(Number(toCurrency.DecimalDigits) || 0));
    const step = (toCurrency as any).Rounding;
    try {
      if (step != null) {
        const dStep = step instanceof Decimal ? step : new Decimal((step as any).$bigdecimal ?? step);
        if (dStep.gt(0)) {
          // Round to nearest multiple of step.
          const q = amount.div(dStep);
          const qRounded = q.toDecimalPlaces(0, Decimal.ROUND_HALF_UP);
          const stepped = qRounded.times(dStep);
          return stepped.toDecimalPlaces(digits, Decimal.ROUND_HALF_UP);
        }
      }
    } catch {
      // fall back to digits rounding
    }
    return amount.toDecimalPlaces(digits, Decimal.ROUND_HALF_UP);
  }

  static async Convert(params: CurrencyConvertParams): Promise<CurrencyConvertResult> {
    const companyId = String(params?.CompanyId ?? '').trim();
    const fromCurrencyId = String(params?.FromCurrencyId ?? '').trim();
    const toCurrencyId = String(params?.ToCurrencyId ?? '').trim();
    if (!companyId || !fromCurrencyId || !toCurrencyId) {
      throw new ChoysumError({ domain: 'base', code: 'InvalidArgument', message: 'CompanyId/FromCurrencyId/ToCurrencyId are required' }).withGrpcCode(
        GrpcCode.InvalidArgument
      );
    }

    const date = this.parseDateString(params?.Date);
    const amount = this.parseDecimalInput(params?.Amount);

    if (fromCurrencyId === toCurrencyId) {
      return { Amount: amount };
    }

    const { default: Company } = await import('./company');
    let company: any;
    try {
      company = (await Company.Browse(companyId, ['Id', 'CurrencyId'] as any)) as any;
    } catch {
      throw new ChoysumError({ domain: 'base', code: 'NotFound', message: 'Company not found' }).withGrpcCode(GrpcCode.NotFound);
    }
    const companyCurrencyId = String(company?.CurrencyId?.Id ?? company?.CurrencyId ?? '').trim();
    if (!companyCurrencyId) {
      throw new ChoysumError({ domain: 'base', code: 'FailedPrecondition', message: 'Company.CurrencyId is required' }).withGrpcCode(GrpcCode.FailedPrecondition);
    }

    const mode = this.normalizeRatePolicyMode(params?.RatePolicy?.Mode);
    const allowFallbackToGlobal = Boolean(params?.RatePolicy?.AllowFallbackToGlobal ?? true);
    const roundingMode = this.normalizeRoundingMode(params?.Rounding?.Mode);
    const overrideDigits = params?.Rounding?.ToDecimalDigitsOverride;

    const warnings: string[] = [];
    const rateUsed: any = {};

    const needFromRate = fromCurrencyId !== companyCurrencyId;
    const needToRate = toCurrencyId !== companyCurrencyId;

    let rateFrom: Decimal | undefined;
    let rateTo: Decimal | undefined;

    if (needFromRate) {
      const { rec, usedGlobal, usedFallbackDate } = await this.getRateRecord({
        companyId,
        currencyId: fromCurrencyId,
        date,
        mode,
        allowFallbackToGlobal,
      });
      if (!rec) {
        throw new ChoysumError({ domain: 'base', code: 'NotFound', message: `ExchangeRate not found for currency ${fromCurrencyId}` }).withGrpcCode(
          GrpcCode.NotFound
        );
      }
      rateFrom = (rec as any).Rate as Decimal;
      rateUsed.From = { CurrencyId: fromCurrencyId, Date: this.toDateOnlyString((rec as any).Date), Rate: rateFrom };
      if (usedFallbackDate) warnings.push('rate.latest_before.fallback');
      if (usedGlobal) warnings.push('rate.global.fallback');
    }

    if (needToRate) {
      const { rec, usedGlobal, usedFallbackDate } = await this.getRateRecord({
        companyId,
        currencyId: toCurrencyId,
        date,
        mode,
        allowFallbackToGlobal,
      });
      if (!rec) {
        throw new ChoysumError({ domain: 'base', code: 'NotFound', message: `ExchangeRate not found for currency ${toCurrencyId}` }).withGrpcCode(
          GrpcCode.NotFound
        );
      }
      rateTo = (rec as any).Rate as Decimal;
      rateUsed.To = { CurrencyId: toCurrencyId, Date: this.toDateOnlyString((rec as any).Date), Rate: rateTo };
      if (usedFallbackDate) warnings.push('rate.latest_before.fallback');
      if (usedGlobal) warnings.push('rate.global.fallback');
    }

    // compute via pivot (company currency)
    let amountCompany: Decimal;
    if (!needFromRate) amountCompany = amount;
    else amountCompany = amount.times(rateFrom!);

    let out: Decimal;
    if (!needToRate) out = amountCompany;
    else out = amountCompany.div(rateTo!);

    if (roundingMode === 'currency') {
      const toCurrency = (await this.Browse(toCurrencyId, ['Id', 'DecimalDigits', 'Rounding'] as any)) as any;
      out = this.roundToCurrency(out, toCurrency as any, overrideDigits);
    }

    const result: any = { Amount: out };
    if (Object.keys(rateUsed).length > 0) result.RateUsed = rateUsed;
    if (warnings.length > 0) result.Warnings = Array.from(new Set(warnings));
    return result;
  }
}

// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { Decimal } from '@/core/service';
import { ChoysumError, GrpcCode } from '@/core/service/error';
import { normalizeDateString, normalizeRatePolicyMode, normalizeRoundingMode } from './_normalizers';
import type Currency from './currency';
import type { CurrencyConvertParams, CurrencyConvertResult } from './currency';

function parseDecimalInput(value: any): Decimal {
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

function toDateOnlyString(input: any): string {
  if (input instanceof Date) return input.toISOString().slice(0, 10);
  const s = String(input ?? '').trim();
  if (!s) return '';
  return s.length >= 10 ? s.slice(0, 10) : s;
}

async function getRateRecord(opts: {
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
    const usedFallbackDate = mode === 'latest_before' && toDateOnlyString((recCompany as any).Date) !== date;
    return { rec: recCompany, usedGlobal: false, usedFallbackDate };
  }

  if (opts.allowFallbackToGlobal) {
    const recGlobal = await searchOne(['CompanyId', 'is', null]);
    if (recGlobal) {
      const usedFallbackDate = mode === 'latest_before' && toDateOnlyString((recGlobal as any).Date) !== date;
      return { rec: recGlobal, usedGlobal: true, usedFallbackDate };
    }
  }

  return { rec: undefined, usedGlobal: false, usedFallbackDate: false };
}

function roundToCurrency(amount: Decimal, toCurrency: Currency, overrideDigits?: number): Decimal {
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

export async function convertCurrency(
  model: { Browse: (id: string, fields: any) => Promise<any> },
  params: CurrencyConvertParams
): Promise<CurrencyConvertResult> {
  const companyId = String(params?.CompanyId ?? '').trim();
  const fromCurrencyId = String(params?.FromCurrencyId ?? '').trim();
  const toCurrencyId = String(params?.ToCurrencyId ?? '').trim();
  if (!companyId || !fromCurrencyId || !toCurrencyId) {
    throw new ChoysumError({ domain: 'base', code: 'InvalidArgument', message: 'CompanyId/FromCurrencyId/ToCurrencyId are required' }).withGrpcCode(
      GrpcCode.InvalidArgument
    );
  }

  const date = normalizeDateString(params?.Date, 'Date');
  const amount = parseDecimalInput(params?.Amount);

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

  const mode = normalizeRatePolicyMode(params?.RatePolicy?.Mode);
  const allowFallbackToGlobal = Boolean(params?.RatePolicy?.AllowFallbackToGlobal ?? true);
  const roundingMode = normalizeRoundingMode(params?.Rounding?.Mode);
  const overrideDigits = params?.Rounding?.ToDecimalDigitsOverride;

  const warnings: string[] = [];
  const rateUsed: any = {};

  const needFromRate = fromCurrencyId !== companyCurrencyId;
  const needToRate = toCurrencyId !== companyCurrencyId;

  let rateFrom: Decimal | undefined;
  let rateTo: Decimal | undefined;

  if (needFromRate) {
    const { rec, usedGlobal, usedFallbackDate } = await getRateRecord({
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
    rateUsed.From = { CurrencyId: fromCurrencyId, Date: toDateOnlyString((rec as any).Date), Rate: rateFrom };
    if (usedFallbackDate) warnings.push('rate.latest_before.fallback');
    if (usedGlobal) warnings.push('rate.global.fallback');
  }

  if (needToRate) {
    const { rec, usedGlobal, usedFallbackDate } = await getRateRecord({
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
    rateUsed.To = { CurrencyId: toCurrencyId, Date: toDateOnlyString((rec as any).Date), Rate: rateTo };
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
    const toCurrency = (await model.Browse(toCurrencyId, ['Id', 'DecimalDigits', 'Rounding'] as any)) as any;
    out = roundToCurrency(out, toCurrency as any, overrideDigits);
  }

  const result: any = { Amount: out };
  if (Object.keys(rateUsed).length > 0) result.RateUsed = rateUsed;
  if (warnings.length > 0) result.Warnings = Array.from(new Set(warnings));
  return result;
}

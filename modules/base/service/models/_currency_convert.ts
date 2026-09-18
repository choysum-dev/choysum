// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { Decimal } from '@/core/service';
import { ChoysumError, GrpcCode } from '@/core/service/error';
import { createTranslate } from '@/core/service/i18n';
import { condition, type BaseQueryCondition, type OrderBy } from '@/core/service/api/query';
import type { FieldSelection } from '@/core/service/api/selection';
import {
  assertDateString,
  parseDecimalInput,
  resolveModelRefId,
  roundToCurrencyAmount,
  toDateOnlyString,
  type CurrencyRoundingSpec,
} from '@/core/service/utils/normalization';
import { mapNormalizationToBase, assertRatePolicyMode, assertRoundingMode } from './_normalizers';
import type Company from './company';
import type Currency from './currency';
import type ExchangeRate from './exchange_rate';
import type { CurrencyConvertParams, CurrencyConvertResult } from './currency';

const { _t } = createTranslate('base');

const RATE_FIELDS = ['Id', 'CurrencyId', 'CompanyId', 'Date', 'Rate'] as const;
/** Pick (not Projected): ExchangeRate.Date/Rate are typed `any` and filtered out of Selectable. */
type RateRecord = Pick<ExchangeRate, (typeof RATE_FIELDS)[number]>;

async function getRateRecord(opts: {
  companyId: string;
  currencyId: string;
  date: string;
  mode: 'exact' | 'latest_before';
  allowFallbackToGlobal: boolean;
}): Promise<{ rec?: RateRecord; usedGlobal: boolean; usedFallbackDate: boolean }> {
  const { default: ExchangeRateModel } = await import('./exchange_rate');
  const { companyId, currencyId, date, mode } = opts;
  const rateFields = RATE_FIELDS as unknown as FieldSelection<ExchangeRate>;
  const searchOne = async (companyCond: BaseQueryCondition): Promise<RateRecord | undefined> => {
    if (mode === 'exact') {
      const rows = await ExchangeRateModel.Search(
        condition<ExchangeRate>({ And: [companyCond, ['CurrencyId', '=', currencyId], ['Date', '=', date]] }),
        {
          limit: 1,
          fields: rateFields,
        }
      );
      return rows?.[0] as RateRecord | undefined;
    }
    const rows = await ExchangeRateModel.Search(
      condition<ExchangeRate>({ And: [companyCond, ['CurrencyId', '=', currencyId], ['Date', '<=', date]] }),
      {
        limit: 1,
        orderBy: { field: 'Date', order: 'desc' } as unknown as OrderBy<ExchangeRate>,
        fields: rateFields,
      }
    );
    return rows?.[0] as RateRecord | undefined;
  };

  // company scoped
  const recCompany = await searchOne(['CompanyId', '=', companyId]);
  if (recCompany) {
    const usedFallbackDate = mode === 'latest_before' && toDateOnlyString(recCompany.Date) !== date;
    return { rec: recCompany, usedGlobal: false, usedFallbackDate };
  }

  if (opts.allowFallbackToGlobal) {
    const recGlobal = await searchOne(['CompanyId', 'is', null]);
    if (recGlobal) {
      const usedFallbackDate = mode === 'latest_before' && toDateOnlyString(recGlobal.Date) !== date;
      return { rec: recGlobal, usedGlobal: true, usedFallbackDate };
    }
  }

  return { rec: undefined, usedGlobal: false, usedFallbackDate: false };
}

type CurrencyOps = {
  Browse: (id: string, fields?: FieldSelection<Currency>) => Promise<Currency | null | undefined>;
};

export async function convertCurrency(model: CurrencyOps, params: CurrencyConvertParams): Promise<CurrencyConvertResult> {
  const companyId = String(params?.CompanyId ?? '').trim();
  const fromCurrencyId = String(params?.FromCurrencyId ?? '').trim();
  const toCurrencyId = String(params?.ToCurrencyId ?? '').trim();
  if (!companyId || !fromCurrencyId || !toCurrencyId) {
    throw new ChoysumError({
      domain: 'base',
      code: 'InvalidArgument',
      message: _t('CompanyId/FromCurrencyId/ToCurrencyId are required', { scope: 'service/models/_currency_convert' }),
    }).withGrpcCode(GrpcCode.InvalidArgument);
  }

  const date = mapNormalizationToBase(
    () => assertDateString(params?.Date),
    err => {
      if (err.code === 'required') return _t('Date is required', { scope: 'service/models/_currency_convert' });
      if (err.code === 'invalid_date_value') return _t('Date is invalid', { scope: 'service/models/_currency_convert' });
      return _t('Date must be YYYY-MM-DD', { scope: 'service/models/_currency_convert' });
    }
  );
  const amount = mapNormalizationToBase(
    () => parseDecimalInput(params?.Amount, { allowNumber: false }),
    () => _t('Invalid Amount', { scope: 'service/models/_currency_convert' })
  );

  const mode = assertRatePolicyMode(params?.RatePolicy?.Mode) ?? 'latest_before';
  const allowFallbackToGlobal = Boolean(params?.RatePolicy?.AllowFallbackToGlobal ?? true);
  const roundingMode = assertRoundingMode(params?.Rounding?.Mode) ?? 'currency';
  const overrideDigits = params?.Rounding?.ToDecimalDigitsOverride;

  if (fromCurrencyId === toCurrencyId) {
    return { Amount: amount };
  }

  const { default: Company } = await import('./company');
  let company: Pick<Company, 'Id' | 'CurrencyId'> | null | undefined;
  try {
    company = await Company.Browse(companyId, ['Id', 'CurrencyId']);
  } catch {
    throw new ChoysumError({ domain: 'base', code: 'NotFound', message: _t('Company not found', { scope: 'service/models/_currency_convert' }) }).withGrpcCode(
      GrpcCode.NotFound
    );
  }
  if (!company) {
    throw new ChoysumError({ domain: 'base', code: 'NotFound', message: _t('Company not found', { scope: 'service/models/_currency_convert' }) }).withGrpcCode(
      GrpcCode.NotFound
    );
  }
  const companyCurrencyId = String(resolveModelRefId(company, 'CurrencyId') ?? '').trim();
  if (!companyCurrencyId) {
    throw new ChoysumError({
      domain: 'base',
      code: 'FailedPrecondition',
      message: _t('Company.CurrencyId is required', { scope: 'service/models/_currency_convert' }),
    }).withGrpcCode(GrpcCode.FailedPrecondition);
  }

  const warnings: string[] = [];
  const rateUsed: NonNullable<CurrencyConvertResult['RateUsed']> = {};

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
      throw new ChoysumError({
        domain: 'base',
        code: 'NotFound',
        message: _t('ExchangeRate not found for currency %s', { scope: 'service/models/_currency_convert' }, fromCurrencyId),
      }).withGrpcCode(GrpcCode.NotFound);
    }
    rateFrom = rec.Rate as Decimal;
    rateUsed.From = { CurrencyId: fromCurrencyId, Date: toDateOnlyString(rec.Date), Rate: rateFrom };
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
      throw new ChoysumError({
        domain: 'base',
        code: 'NotFound',
        message: _t('ExchangeRate not found for currency %s', { scope: 'service/models/_currency_convert' }, toCurrencyId),
      }).withGrpcCode(GrpcCode.NotFound);
    }
    rateTo = rec.Rate as Decimal;
    rateUsed.To = { CurrencyId: toCurrencyId, Date: toDateOnlyString(rec.Date), Rate: rateTo };
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
    const toCurrency = await model.Browse(toCurrencyId, ['Id', 'DecimalDigits', 'Rounding']);
    if (!toCurrency) {
      throw new ChoysumError({
        domain: 'base',
        code: 'NotFound',
        message: _t('Target currency %s not found', { scope: 'service/models/_currency_convert' }, toCurrencyId),
      }).withGrpcCode(GrpcCode.NotFound);
    }
    const roundingSpec: CurrencyRoundingSpec = {
      DecimalDigits: (toCurrency as { DecimalDigits?: unknown }).DecimalDigits,
      Rounding: (toCurrency as { Rounding?: unknown }).Rounding,
    };
    out = roundToCurrencyAmount(out, roundingSpec, overrideDigits);
  }

  const result: CurrencyConvertResult = { Amount: out };
  if (Object.keys(rateUsed).length > 0) result.RateUsed = rateUsed;
  if (warnings.length > 0) result.Warnings = Array.from(new Set(warnings));
  return result;
}

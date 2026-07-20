// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Decimal, Field, Model } from '@/core/service';
import { Constraint } from '@/core/service/api/constraint';
import { normalizeRefId, normalizeDateString, toPositiveDecimal } from '@/core/service/utils/normalization';
import { _t, _lt } from '../i18n';
import Company from './company';
import Currency from './currency';
import { fail, mapNormalizationToBase, requireRefId } from './_normalizers';

@Model('ExchangeRate', { companyScoped: true })
export default class ExchangeRate extends BaseModel {
  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => Currency },
    notNull: true,
    index: true,
    uniqueIndex: 'uidx_base_exchange_rate_scope_currency_date',
    string: _lt('Currency', { scope: 'base.model.ExchangeRate.fields' }),
  })
  CurrencyId: Currency;

  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => Company },
    index: true,
    string: _lt('Company', { scope: 'base.model.ExchangeRate.fields' }),
  })
  CompanyId?: Company;

  @Field({
    type: 'varchar',
    size: 20,
    notNull: true,
    default: () => '__GLOBAL__',
    index: true,
    uniqueIndex: 'uidx_base_exchange_rate_scope_currency_date',
  })
  CompanyScopeKey: string;

  @Field({
    type: 'date',
    notNull: true,
    index: true,
    uniqueIndex: 'uidx_base_exchange_rate_scope_currency_date',
    string: _lt('Date', { scope: 'base.model.ExchangeRate.fields' }),
  })
  Date: any;

  @Field({
    type: 'decimal',
    notNull: true,
    precision: 38,
    scale: 18,
    string: _lt('Exchange Rate', { scope: 'base.model.ExchangeRate.fields' }),
  })
  Rate: any;

  private static coerceDateKey(value: any): string {
    if (value instanceof Date) {
      if (Number.isNaN(value.getTime())) {
        fail(_t('Date is invalid', { scope: 'service/models/exchange_rate' }));
      }
      return value.toISOString().slice(0, 10);
    }
    return mapNormalizationToBase(
      () => normalizeDateString(value),
      err => {
        if (err.code === 'required') return _t('Date is required', { scope: 'service/models/exchange_rate' });
        if (err.code === 'invalid_date_value') return _t('Date is invalid', { scope: 'service/models/exchange_rate' });
        return _t('Date must be YYYY-MM-DD', { scope: 'service/models/exchange_rate' });
      }
    );
  }

  private static dateKey(value: any): string {
    return this.coerceDateKey(value);
  }

  private static async ensureUniqueTuple(values: Record<string, any>, currentId?: string): Promise<void> {
    const scopeKey = String(values.CompanyScopeKey ?? (normalizeRefId(values.CompanyId) || '__GLOBAL__'));
    const currencyId = requireRefId(values.CurrencyId, 'CurrencyId');
    const dateKey = this.dateKey(values.Date);

    const conflicts = await this.Search(
      {
        And: [
          ['CompanyScopeKey', '=', scopeKey],
          ['CurrencyId', '=', currencyId],
          ['Date', '=', dateKey],
        ],
      } as any,
      { fields: ['Id'] as any, limit: 2 } as any
    );

    const hasConflict = (conflicts || []).some((item: any) => String(item?.Id || '') !== String(currentId || ''));
    if (hasConflict) {
      fail(_t('ExchangeRate must be unique for CompanyId + CurrencyId + Date', { scope: 'service/models/exchange_rate' }));
    }
  }

  private static async validateEntity(values: Record<string, any>, currentId?: string): Promise<void> {
    values.Rate = mapNormalizationToBase(
      () => toPositiveDecimal(values.Rate).toString(),
      err =>
        err.code === 'non_positive_decimal'
          ? _t('Rate must be greater than 0', { scope: 'service/models/exchange_rate' })
          : _t('Rate must be a valid decimal', { scope: 'service/models/exchange_rate' })
    );
    values.CompanyScopeKey = normalizeRefId(values.CompanyId) || '__GLOBAL__';
    values.Date = this.dateKey(values.Date);
    await this.ensureUniqueTuple(values, currentId);
  }

  @Constraint<ExchangeRate>(['CompanyId', 'CurrencyId', 'Date', 'Rate'])
  async validateExchangeRateConstraint(): Promise<void> {
    const currentId = String((this as any).Id || '').trim() || undefined;
    await ExchangeRate.validateEntity(this as any, currentId);
  }
}

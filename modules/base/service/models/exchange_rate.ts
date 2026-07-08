// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Decimal, Field, Model } from '@/core/service';
import { Constraint } from '@/core/service/api/constraint';
import Company from './company';
import Currency from './currency';
import { asRefId, normalizeCompanyScopeKey } from './_refs';
import { fail, toPositiveDecimal } from './_normalizers';

@Model('ExchangeRate', { companyScoped: true })
export default class ExchangeRate extends BaseModel {
  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => Currency },
    column: { notNull: true, index: true, uniqueIndex: 'uidx_base_exchange_rate_scope_currency_date' },
  })
  CurrencyId: Currency;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => Company }, column: { index: true } })
  CompanyId?: Company;

  @Field({
    type: 'varchar',
    column: { size: 20, notNull: true, default: () => '__GLOBAL__', index: true, uniqueIndex: 'uidx_base_exchange_rate_scope_currency_date' },
  })
  CompanyScopeKey: string;

  @Field({ type: 'date', column: { notNull: true, index: true, uniqueIndex: 'uidx_base_exchange_rate_scope_currency_date' } })
  Date: any;

  @Field({ type: 'decimal', column: { notNull: true, precision: 38, scale: 18 } })
  Rate: any;

  private static normalizeDateStringInput(value: any): string {
    if (value === undefined || value === null || value === '') fail('Date is required');
    if (value instanceof Date) fail('Date must be YYYY-MM-DD');
    const raw = String(value).trim();
    if (!/^\d{4}-\d{2}-\d{2}$/.test(raw)) {
      fail('Date must be YYYY-MM-DD');
    }
    const date = new Date(`${raw}T00:00:00.000Z`);
    if (Number.isNaN(date.getTime()) || date.toISOString().slice(0, 10) !== raw) {
      fail('Date is invalid');
    }
    return raw;
  }

  private static coerceDateKey(value: any): string {
    if (value === undefined || value === null || value === '') fail('Date is required');
    if (value instanceof Date) return value.toISOString().slice(0, 10);
    return this.normalizeDateStringInput(value);
  }

  private static dateKey(value: any): string {
    return this.coerceDateKey(value);
  }

  private static async ensureUniqueTuple(values: Record<string, any>, currentId?: string): Promise<void> {
    const scopeKey = String(values.CompanyScopeKey ?? normalizeCompanyScopeKey(values.CompanyId));
    const currencyId = asRefId(values.CurrencyId);
    const dateKey = this.dateKey(values.Date);

    if (!currencyId) fail('CurrencyId is required');

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
      fail('ExchangeRate must be unique for CompanyId + CurrencyId + Date');
    }
  }

  private static async validateEntity(values: Record<string, any>, currentId?: string): Promise<void> {
    toPositiveDecimal(values.Rate, 'Rate');
    values.CompanyScopeKey = normalizeCompanyScopeKey(values.CompanyId);
    values.Date = this.dateKey(values.Date);
    await this.ensureUniqueTuple(values, currentId);
  }

  @Constraint<ExchangeRate>(['CompanyId', 'CurrencyId', 'Date', 'Rate'])
  static async validateExchangeRateConstraint(self: ExchangeRate, ctx: any): Promise<void> {
    const current = (ctx?.current || {}) as Record<string, any>;
    const values = (ctx?.values || {}) as Record<string, any>;
    const currentId = String(current?.Id || '').trim() || undefined;

    await ExchangeRate.validateEntity(self as any, currentId);

    if (Object.prototype.hasOwnProperty.call(values, 'Date')) {
      values.Date = self.Date;
    }
    if (Object.prototype.hasOwnProperty.call(values, 'CompanyId') || Object.prototype.hasOwnProperty.call(values, 'CompanyScopeKey')) {
      values.CompanyScopeKey = (self as any).CompanyScopeKey;
    }
  }
}

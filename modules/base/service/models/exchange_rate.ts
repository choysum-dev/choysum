// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Decimal, Field, Model } from '@/core/service';
import { Constraint } from '@/core/service/api/constraint';
import { normalizeRefId, normalizeDateString, toPositiveDecimal } from '@/core/service/utils/normalization';
import { businessToday } from '@/core/service/utils/datetime';
import { _t, _lt } from '../i18n';
import Company from './company';
import Currency from './currency';
import { fail, mapNormalizationToBase } from './_normalizers';

@Model('ExchangeRate', { companyField: 'CompanyId' })
export default class ExchangeRate extends BaseModel {
  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => Currency },
    condition: ['IsActive', '=', true],
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
    help: _lt('Leave empty to use a global rate when no company rate exists.', {
      scope: 'base.model.ExchangeRate.fields',
    }),
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

  /**
   * Business calendar day for this rate (company timezone via businessToday; see multi-timezone-design §4.2 / §8).
   * Not a UTC instant — UI must not rezone this `date` field.
   */
  @Field({
    type: 'date',
    notNull: true,
    index: true,
    uniqueIndex: 'uidx_base_exchange_rate_scope_currency_date',
    default: () => businessToday(),
    string: _lt('Date', { scope: 'base.model.ExchangeRate.fields' }),
    help: _lt('Business calendar date in the company timezone, not a UTC instant.', {
      scope: 'base.model.ExchangeRate.fields',
    }),
  })
  Date: any;

  @Field({
    type: 'decimal',
    notNull: true,
    string: _lt('Exchange Rate', { scope: 'base.model.ExchangeRate.fields' }),
    help: _lt('Units of this currency per one unit of the company base currency.', {
      scope: 'base.model.ExchangeRate.fields',
    }),
  })
  Rate: any;

  private static coerceDateKey(value: any): string {
    // Date-only business keys must be YYYY-MM-DD strings. Reject Date objects:
    // toISOString().slice(0, 10) reinterprets local midnights as UTC calendar days.
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
    let currencyId = '';
    const currencyRaw = values.CurrencyId;
    if (typeof currencyRaw === 'string') {
      currencyId = currencyRaw.trim();
    } else if (currencyRaw != null && typeof currencyRaw === 'object') {
      let rawId = (currencyRaw as any).Id;
      if (rawId == null) {
        rawId = (currencyRaw as any).id;
      }
      if (rawId != null) {
        currencyId = String(rawId).trim();
      }
    }
    if (!currencyId) {
      fail(_t('%s is required', { scope: 'service/models/exchange_rate' }, 'CurrencyId'));
    }
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
    if (values.Date == null || values.Date === '') {
      // Default posting/business day uses companyTz, independent of user display tz.
      values.Date = businessToday();
    }
    values.Date = this.dateKey(values.Date);
    await this.ensureUniqueTuple(values, currentId);
  }

  @Constraint<ExchangeRate>(['CompanyId', 'CurrencyId', 'Date', 'Rate'])
  async validateExchangeRateConstraint(): Promise<void> {
    const currentId = String((this as any).Id || '').trim() || undefined;
    await ExchangeRate.validateEntity(this as any, currentId);
  }
}

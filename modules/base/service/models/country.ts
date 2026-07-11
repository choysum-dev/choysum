// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { Constraint } from '@/core/service/api/constraint';
import Currency from './currency';
import { fail, normalizeCodeRequired } from './_normalizers';

@Model('Country')
export default class Country extends BaseModel {
  @Field({ type: 'varchar', column: { size: 100, notNull: true, index: true } })
  Name: string;

  @Field({ type: 'varchar', column: { size: 8, notNull: true, unique: true, index: true } })
  Code: string;

  @Field({ type: 'varchar', column: { size: 16 } })
  PhonePrefix?: string;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => Currency }, column: { index: true } })
  DefaultCurrencyId?: Currency;

  @Field({ type: 'text' })
  AddressFormat?: string;

  @Field({ type: 'boolean', column: { notNull: true, default: () => true } })
  ZipRequired: boolean;

  @Field({ type: 'boolean', column: { notNull: true, default: () => false } })
  StateRequired: boolean;

  @Field({ type: 'boolean', column: { notNull: true, default: () => true, index: true } })
  IsActive: boolean;

  private validateAddressFormat(value: any): string | null {
    if (value === undefined || value === null) return null;
    const format = String(value).trim();
    if (!format) return null;

    const allowed = new Set(['street1', 'street2', 'zip', 'cityname', 'statename', 'statecode', 'countryname', 'countrycode']);
    const tokenPattern = /%\(([A-Za-z][A-Za-z0-9_]*)\)s|\{([A-Za-z][A-Za-z0-9_]*)\}/g;
    let m: RegExpExecArray | null;
    while ((m = tokenPattern.exec(format)) !== null) {
      const token = (m[1] || m[2] || '').toLowerCase();
      if (!allowed.has(token)) {
        fail(`AddressFormat contains unsupported token: ${m[1] || m[2]}`);
      }
    }
    return format;
  }

  @Constraint<Country>(['Code', 'AddressFormat'])
  validateCountryConstraint(): void {
    this.Code = normalizeCodeRequired(this.Code as string);
    if (this.AddressFormat != null) {
      this.AddressFormat = this.validateAddressFormat(this.AddressFormat) as any;
    }
  }
}

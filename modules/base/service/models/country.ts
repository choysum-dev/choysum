// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { Constraint } from '@/core/service/api/constraint';
import { _t, _lt } from '../i18n';
import Currency from './currency';
import { fail, normalizeCodeRequired } from './_normalizers';

@Model('Country')
export default class Country extends BaseModel {
  @Field({
    type: 'varchar',
    size: 100,
    notNull: true,
    translate: true,
    index: 'trigram',
    string: _lt('Name', { scope: 'base.model.Country.fields' }),
  })
  Name: string;

  @Field({
    type: 'varchar',
    size: 8,
    notNull: true,
    unique: true,
    index: true,
    string: _lt('Code', { scope: 'base.model.Country.fields' }),
  })
  Code: string;

  @Field({
    type: 'varchar',
    size: 16,
    string: _lt('Dialing Code', { scope: 'base.model.Country.fields' }),
  })
  PhonePrefix?: string;

  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => Currency },
    condition: ['IsActive', '=', true],
    index: true,
    string: _lt('Default Currency', { scope: 'base.model.Country.fields' }),
  })
  DefaultCurrencyId?: Currency;

  @Field({
    type: 'text',
    string: _lt('Address Format', { scope: 'base.model.Country.fields' }),
  })
  AddressFormat?: string;

  @Field({
    type: 'boolean',
    notNull: true,
    default: () => true,
    string: _lt('ZIP Required', { scope: 'base.model.Country.fields' }),
  })
  ZipRequired: boolean;

  @Field({
    type: 'boolean',
    notNull: true,
    default: () => false,
    string: _lt('State/Province Required', { scope: 'base.model.Country.fields' }),
  })
  StateRequired: boolean;

  @Field({
    type: 'boolean',
    notNull: true,
    default: () => true,
    index: true,
    string: _lt('Active', { scope: 'base.model.Country.fields' }),
  })
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
        fail(_t('AddressFormat contains unsupported token: %s', { scope: 'service/models/country' }, m[1] || m[2]));
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

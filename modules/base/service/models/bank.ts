// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { Constraint } from '@/core/service/api/constraint';
import { _lt } from '../i18n';
import Address from './address';
import Country from './country';
import { normalizeCodeOptional } from './_normalizers';

@Model('Bank')
export default class Bank extends BaseModel {
  @Field({
    type: 'varchar',
    size: 120,
    notNull: true,
    translate: true,
    index: 'trigram',
    string: _lt('Name', { scope: 'base.model.Bank.fields' }),
  })
  Name: string;

  @Field({
    type: 'varchar',
    size: 40,
    index: true,
    string: _lt('Code', { scope: 'base.model.Bank.fields' }),
  })
  Code?: string;

  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => Country },
    index: true,
    string: _lt('Country', { scope: 'base.model.Bank.fields' }),
  })
  CountryId?: Country;

  @Field({
    type: 'varchar',
    size: 20,
    index: true,
    string: _lt('BIC', { scope: 'base.model.Bank.fields' }),
  })
  BIC?: string;

  @Field({
    type: 'boolean',
    notNull: true,
    default: () => true,
    index: true,
    string: _lt('Active', { scope: 'base.model.Bank.fields' }),
  })
  IsActive: boolean;

  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => Address },
    index: true,
    string: _lt('Address', { scope: 'base.model.Bank.fields' }),
  })
  AddressId?: Address;

  @Constraint<Bank>(['Code'])
  validateBankConstraint(): void {
    if (this.Code != null) {
      (this as any).Code = normalizeCodeOptional(this.Code as string);
    }
  }
}

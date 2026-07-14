// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { Constraint } from '@/core/service/api/constraint';
import Address from './address';
import Country from './country';
import { normalizeCodeOptional } from './_normalizers';

@Model('Bank')
export default class Bank extends BaseModel {
  @Field({ type: 'varchar', size: 120, notNull: true, index: true})
  Name: string;

  @Field({ type: 'varchar', size: 40, index: true})
  Code?: string;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => Country }, index: true})
  CountryId?: Country;

  @Field({ type: 'varchar', size: 20, index: true})
  BIC?: string;

  @Field({ type: 'boolean', notNull: true, default: () => true, index: true})
  IsActive: boolean;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => Address }, index: true})
  AddressId?: Address;

  @Constraint<Bank>(['Code'])
  validateBankConstraint(): void {
    if (this.Code != null) {
      (this as any).Code = normalizeCodeOptional(this.Code as string);
    }
  }
}

// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { Constraint } from '@/core/service/api/constraint';
import { normalizeRefId, normalizeRequiredText as normalizeRequiredTextCore } from '@/core/service/utils/normalization';
import { isIanaTimezone, listIanaTimezoneSelection } from '@/core/service/utils/datetime';
import { raiseDomainError } from '@/core/service/error';
import { _t, _lt } from '../i18n';
import Address from './address';
import Country from './country';
import Currency from './currency';
import Language from './language';
import { fail, mapNormalizationToBase } from './_normalizers';

@Model('Company', { parentField: 'ParentId' })
export default class Company extends BaseModel {
  @Field({
    type: 'varchar',
    size: 100,
    unique: true,
    notNull: true,
    string: _lt('Name', { scope: 'base.model.Company.fields' }),
  })
  Name: string;

  @Field({
    type: 'varchar',
    size: 40,
    unique: true,
    notNull: true,
    index: true,
    string: _lt('Code', { scope: 'base.model.Company.fields' }),
  })
  Code: string;

  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => Company },
    index: true,
    string: _lt('Parent Company', { scope: 'base.model.Company.fields' }),
  })
  ParentId?: Company;

  @Field({
    type: 'selection',
    selection: () => listIanaTimezoneSelection(),
    size: 64,
    notNull: true,
    string: _lt('Time Zone', { scope: 'base.model.Company.fields' }),
  })
  Timezone: string;

  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => Currency },
    notNull: true,
    index: true,
    string: _lt('Base Currency', { scope: 'base.model.Company.fields' }),
  })
  CurrencyId: Currency;

  @Field({
    type: 'boolean',
    notNull: true,
    default: () => true,
    index: true,
    string: _lt('Active', { scope: 'base.model.Company.fields' }),
  })
  IsActive: boolean;

  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => Language },
    index: true,
    string: _lt('Language', { scope: 'base.model.Company.fields' }),
  })
  LanguageId?: Language;

  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => Country },
    index: true,
    string: _lt('Country', { scope: 'base.model.Company.fields' }),
  })
  CountryId?: Country;

  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => Address },
    index: true,
    string: _lt('Address', { scope: 'base.model.Company.fields' }),
  })
  AddressId?: Address;

  private static normalizeRequiredText(value: unknown, fieldName: string): string {
    return mapNormalizationToBase(
      () => normalizeRequiredTextCore(value),
      () => _t('%s is required', { scope: 'service/models/company' }, fieldName)
    );
  }

  private static async ensureUnique(values: Record<string, any>, currentId?: string): Promise<void> {
    const name = this.normalizeRequiredText(values.Name, 'Name');
    const code = this.normalizeRequiredText(values.Code, 'Code');

    const byName = await this.Search(
      {
        And: [['Name', '=', name]],
      } as any,
      { fields: ['Id'] as any, limit: 2 } as any
    );
    const nameConflict = (byName || []).some((item: any) => String(item?.Id || '') !== String(currentId || ''));
    if (nameConflict) fail(_t('Company Name must be unique', { scope: 'service/models/company' }));

    const byCode = await this.Search(
      {
        And: [['Code', '=', code]],
      } as any,
      { fields: ['Id'] as any, limit: 2 } as any
    );
    const codeConflict = (byCode || []).some((item: any) => String(item?.Id || '') !== String(currentId || ''));
    if (codeConflict) fail(_t('Company Code must be unique', { scope: 'service/models/company' }));

    values.Name = name;
    values.Code = code;
  }

  private static normalizeCurrencyId(value: unknown): string {
    const id = normalizeRefId(value);
    if (!id) {
      raiseDomainError('base', 'InvalidArgument', _t('CurrencyId is required', { scope: 'service/models/company' }));
    }
    return id;
  }

  private static normalizeTimezone(value: unknown): string {
    const timezone = String(value ?? '').trim();
    if (!timezone) {
      raiseDomainError('base', 'InvalidArgument', _t('Timezone is required', { scope: 'service/models/company' }));
    }
    if (!isIanaTimezone(timezone)) {
      raiseDomainError('base', 'InvalidArgument', _t('Invalid IANA timezone: %s', { scope: 'service/models/company' }, timezone));
    }
    return timezone;
  }

  private static async validateParentUpdate(targetId: string, parentIdRaw: any): Promise<void> {
    const parentId = normalizeRefId(parentIdRaw);
    if (!parentId) return;

    if (parentId === targetId) {
      raiseDomainError('base', 'InvalidArgument', _t('ParentId cannot be self', { scope: 'service/models/company' }));
    }

    const found = await this.Search(
      {
        And: [
          ['Id', 'child_of', targetId],
          ['Id', '=', parentId],
        ],
      } as any,
      { limit: 1, fields: ['Id'] as any } as any
    );
    if (found?.[0]) {
      raiseDomainError('base', 'InvalidArgument', _t('ParentId cannot be a descendant of the company', { scope: 'service/models/company' }));
    }
  }

  @Constraint<Company>(['Name', 'Code', 'Timezone', 'CurrencyId', 'ParentId'])
  async validateCompanyConstraint(): Promise<void> {
    const currentId = String((this as any).Id || '').trim() || undefined;
    const isCreate = !currentId;

    // Timezone and CurrencyId are always required; normalize on `this`
    // so the draft proxy auto-collects the writeback.
    (this as any).Timezone = Company.normalizeTimezone(this.Timezone);
    (this as any).CurrencyId = Company.normalizeCurrencyId(this.CurrencyId);

    await Company.ensureUnique(this as any, currentId);

    if (!isCreate && currentId) {
      await Company.validateParentUpdate(currentId, (this as any).ParentId);
    }
  }
}

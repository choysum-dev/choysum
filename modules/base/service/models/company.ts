// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { Constraint } from '@/core/service/api/constraint';
import { normalizeRefId, normalizeRequiredText as normalizeRequiredTextCore } from '@/core/service/utils/normalization';
import { isIanaTimezone } from '@/core/service/utils/datetime';
import { raiseDomainError } from '@/core/service/error';
import Address from './address';
import Country from './country';
import Currency from './currency';
import Language from './language';
import Locale from './locale';
import { fail, mapNormalizationToBase } from './_normalizers';

@Model('Company', { parentField: 'ParentId' })
export default class Company extends BaseModel {
  @Field({ type: 'varchar', column: { size: 100, unique: true, notNull: true } })
  Name: string;

  @Field({ type: 'varchar', column: { size: 40, unique: true, notNull: true, index: true } })
  Code: string;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => Company }, column: { index: true } })
  ParentId?: Company;

  @Field({ type: 'varchar', column: { size: 64, notNull: true } })
  Timezone: string;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => Currency }, column: { notNull: true, index: true } })
  CurrencyId: Currency;

  @Field({ type: 'boolean', column: { notNull: true, default: () => true, index: true } })
  IsActive: boolean;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => Language }, column: { index: true } })
  LanguageId?: Language;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => Locale }, column: { index: true } })
  LocaleId?: Locale;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => Country }, column: { index: true } })
  CountryId?: Country;

  @Field({ type: 'ManyToOne', relation: { targetModel: () => Address }, column: { index: true } })
  AddressId?: Address;

  private static normalizeRequiredText(value: unknown, fieldName: string): string {
    return mapNormalizationToBase(
      () => normalizeRequiredTextCore(value),
      () => `${fieldName} is required`
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
    if (nameConflict) fail('Company Name must be unique');

    const byCode = await this.Search(
      {
        And: [['Code', '=', code]],
      } as any,
      { fields: ['Id'] as any, limit: 2 } as any
    );
    const codeConflict = (byCode || []).some((item: any) => String(item?.Id || '') !== String(currentId || ''));
    if (codeConflict) fail('Company Code must be unique');

    values.Name = name;
    values.Code = code;
  }

  private static normalizeCurrencyId(value: unknown): string {
    const id = normalizeRefId(value);
    if (!id) {
      raiseDomainError('base', 'InvalidArgument', 'CurrencyId is required');
    }
    return id;
  }

  private static normalizeTimezone(value: unknown): string {
    const timezone = String(value ?? '').trim();
    if (!timezone) {
      raiseDomainError('base', 'InvalidArgument', 'Timezone is required');
    }
    if (!isIanaTimezone(timezone)) {
      raiseDomainError('base', 'InvalidArgument', `Invalid IANA timezone: ${timezone}`);
    }
    return timezone;
  }

  private static async validateParentUpdate(targetId: string, parentIdRaw: any): Promise<void> {
    const parentId = normalizeRefId(parentIdRaw);
    if (!parentId) return;

    if (parentId === targetId) {
      raiseDomainError('base', 'InvalidArgument', 'ParentId cannot be self');
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
      raiseDomainError('base', 'InvalidArgument', 'ParentId cannot be a descendant of the company');
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

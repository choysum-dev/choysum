// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { Constraint } from '@/core/service/api/constraint';
import type { QueryCondition } from '@/core/service/api/query';
import { normalizeRefId, normalizeRequiredText as normalizeRequiredTextCore } from '@/core/service/utils/normalization';
import { isIanaTimezone } from '@/core/service/utils/timezone';
import { GrpcCode, ChoysumError } from '@/core/service/error';
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
      throw new ChoysumError({ domain: 'base', code: 'InvalidArgument', message: 'CurrencyId is required' }).withGrpcCode(GrpcCode.InvalidArgument);
    }
    return id;
  }

  private static normalizeTimezone(value: unknown): string {
    const timezone = String(value ?? '').trim();
    if (!timezone) {
      throw new ChoysumError({ domain: 'base', code: 'InvalidArgument', message: 'Timezone is required' }).withGrpcCode(GrpcCode.InvalidArgument);
    }
    if (!isIanaTimezone(timezone)) {
      throw new ChoysumError({ domain: 'base', code: 'InvalidArgument', message: `Invalid IANA timezone: ${timezone}` }).withGrpcCode(GrpcCode.InvalidArgument);
    }
    return timezone;
  }

  private static applyRequiredForCreate(values: Record<string, any>): void {
    values.Timezone = this.normalizeTimezone(values.Timezone);
    values.CurrencyId = this.normalizeCurrencyId(values.CurrencyId);
  }

  private static async applyRequiredForUpdate(values: Record<string, any>, conditionOrId: QueryCondition<Company> | string): Promise<void> {
    if (Object.prototype.hasOwnProperty.call(values, 'Timezone')) {
      values.Timezone = this.normalizeTimezone(values.Timezone);
    }

    if (Object.prototype.hasOwnProperty.call(values, 'CurrencyId')) {
      values.CurrencyId = this.normalizeCurrencyId(values.CurrencyId);
    }

    if (Object.prototype.hasOwnProperty.call(values, 'Timezone') || Object.prototype.hasOwnProperty.call(values, 'CurrencyId')) {
      return;
    }

    if (typeof conditionOrId === 'string') {
      const existing = (await this.Browse(conditionOrId, ['Id', 'Timezone'] as any)) as any;
      this.normalizeTimezone(existing?.Timezone);
      return;
    }

    const rows = await this.Search(conditionOrId as any, { limit: 1, fields: ['Id', 'Timezone'] as any } as any);
    if (rows?.[0]) {
      this.normalizeTimezone((rows[0] as any).Timezone);
    }
  }

  private static async validateParentUpdate(targetId: string, parentIdRaw: any): Promise<void> {
    const parentId = normalizeRefId(parentIdRaw);
    if (!parentId) return;

    if (parentId === targetId) {
      throw new ChoysumError({ domain: 'base', code: 'InvalidArgument', message: 'ParentId cannot be self' }).withGrpcCode(GrpcCode.InvalidArgument);
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
      throw new ChoysumError({ domain: 'base', code: 'InvalidArgument', message: 'ParentId cannot be a descendant of the company' }).withGrpcCode(
        GrpcCode.InvalidArgument
      );
    }
  }

  @Constraint<Company>(['Name', 'Code', 'Timezone', 'CurrencyId', 'ParentId'])
  static async validateCompanyConstraint(self: Company, ctx: any): Promise<void> {
    const mode = String(ctx?.mode || '');
    const current = (ctx?.current || {}) as Record<string, any>;
    const values = (ctx?.values || {}) as Record<string, any>;
    const currentId = String(current?.Id || '').trim() || undefined;

    if (mode === 'create') {
      Company.applyRequiredForCreate(self as any);
    } else if (mode === 'update') {
      if (Object.prototype.hasOwnProperty.call(values, 'Timezone')) {
        (self as any).Timezone = Company.normalizeTimezone((self as any).Timezone);
      } else {
        Company.normalizeTimezone(current?.Timezone);
      }

      if (Object.prototype.hasOwnProperty.call(values, 'CurrencyId')) {
        (self as any).CurrencyId = Company.normalizeCurrencyId((self as any).CurrencyId);
      }
    }

    await Company.ensureUnique(self as any, currentId);

    if (mode === 'update' && currentId && Object.prototype.hasOwnProperty.call(values, 'ParentId')) {
      await Company.validateParentUpdate(currentId, (self as any).ParentId);
    }
  }
}

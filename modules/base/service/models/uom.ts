// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Decimal, Field, Model } from '@/core/service';
import { Constraint } from '@/core/service/api/constraint';
import { GrpcCode, ChoysumError } from '@/core/service/error';
import UoMCategory from './uom_category';

@Model('UoM')
export default class UoM extends BaseModel {
  @Field({ type: 'varchar', column: { size: 100, notNull: true, index: true, uniqueIndex: 'uidx_base_uom_category_name' } })
  Name: string;

  @Field({ type: 'varchar', column: { size: 24 } })
  Symbol?: string;

  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => UoMCategory },
    column: { notNull: true, index: true, uniqueIndex: 'uidx_base_uom_category_name' },
  })
  CategoryId: UoMCategory;

  @Field({ type: 'boolean', column: { notNull: true, default: () => false } })
  IsReference: boolean;

  @Field({ type: 'decimal', column: { notNull: true, precision: 38, scale: 18 } })
  Factor: any;

  @Field({ type: 'decimal', column: { precision: 38, scale: 18 } })
  Rounding?: any;

  @Field({ type: 'boolean', column: { notNull: true, default: () => true, index: true } })
  IsActive: boolean;

  private static fail(message: string): never {
    throw new ChoysumError({ domain: 'base', code: 'InvalidArgument', message }).withGrpcCode(GrpcCode.InvalidArgument);
  }

  private static asRefId(value: any): string | null | undefined {
    if (value === undefined) return undefined;
    if (value === null) return null;
    const raw = typeof value === 'object' && value !== null ? (value.Id ?? value.id) : value;
    const id = String(raw ?? '').trim();
    return id ? id : null;
  }

  private static toPositiveDecimal(value: any, fieldName: string): Decimal {
    try {
      if (value === undefined || value === null || value === '') throw new Error('required');
      const decimal = value instanceof Decimal ? value : new Decimal((value as any)?.$bigdecimal ?? value);
      if (!decimal.gt(0)) this.fail(`${fieldName} must be greater than 0`);
      return decimal;
    } catch (err) {
      if (err instanceof ChoysumError) throw err;
      this.fail(`${fieldName} must be a valid decimal`);
    }
  }

  private static isReferenceValue(value: any): boolean {
    return value === true;
  }

  private static normalizeName(value: any): string {
    const name = String(value ?? '').trim();
    if (!name) this.fail('Name is required');
    return name;
  }

  private static async ensureNameUnique(categoryId: string, name: string, currentId?: string): Promise<void> {
    const hits = await this.Search(
      {
        And: [
          ['CategoryId', '=', categoryId],
          ['Name', '=', name],
        ],
      } as any,
      { fields: ['Id'] as any, limit: 2 } as any
    );
    const conflict = (hits || []).some((item: any) => String(item?.Id || '') !== String(currentId || ''));
    if (conflict) this.fail('UoM Name must be unique within Category');
  }

  private static async ensureCategoryReferenceInvariant(categoryId: string, isReference: boolean, currentId?: string): Promise<void> {
    const refs = await this.Search(
      {
        And: [
          ['CategoryId', '=', categoryId],
          ['IsReference', '=', true],
        ],
      } as any,
      { fields: ['Id'] as any, limit: 2 } as any
    );
    const refsExcludingCurrent = (refs || []).filter((item: any) => String(item?.Id || '') !== String(currentId || ''));

    if (isReference) {
      if (refsExcludingCurrent.length > 0) {
        this.fail('Each UoM category can only have one reference unit');
      }
      return;
    }

    if (refsExcludingCurrent.length === 0) {
      this.fail('Each UoM category must have one reference unit');
    }
  }

  private static async validateEntity(values: Record<string, any>, currentId?: string): Promise<void> {
    const name = this.normalizeName(values.Name);
    const categoryId = this.asRefId(values.CategoryId);
    if (!categoryId) this.fail('CategoryId is required');

    const isReference = this.isReferenceValue(values.IsReference);
    const factor = this.toPositiveDecimal(values.Factor, 'Factor');
    if (isReference && !factor.eq(1)) {
      this.fail('Reference UoM must have Factor = 1');
    }

    if (values.Rounding !== undefined && values.Rounding !== null && values.Rounding !== '') {
      const rounding = this.toPositiveDecimal(values.Rounding, 'Rounding');
      values.Rounding = rounding.toString();
    } else {
      values.Rounding = null;
    }

    values.CategoryId = categoryId;
    values.Name = name;
    values.IsReference = isReference;
    values.Factor = factor.toString();

    await this.ensureNameUnique(categoryId, name, currentId);
    await this.ensureCategoryReferenceInvariant(categoryId, isReference, currentId);
  }

  @Constraint<UoM>(['Name', 'CategoryId', 'IsReference', 'Factor', 'Rounding'])
  static async validateUoMConstraint(self: UoM, ctx: any): Promise<void> {
    const currentId = String((ctx?.current as any)?.Id || '').trim() || undefined;
    await UoM.validateEntity(self as any, currentId);
  }
}

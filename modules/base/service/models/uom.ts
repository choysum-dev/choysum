// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Decimal, Field, Model } from '@/core/service';
import { createTranslate } from '@/core/service/i18n';
import { Constraint } from '@/core/service/api/constraint';
import { toPositiveDecimal } from '@/core/service/utils/normalization';
import UoMCategory from './uom_category';
import { fail, mapNormalizationToBase, normalizeName, requireRefId } from './_normalizers';

const { _t } = createTranslate('base');

@Model('UoM')
export default class UoM extends BaseModel {
  @Field({ type: 'varchar', size: 100, notNull: true, index: true, uniqueIndex: 'uidx_base_uom_category_name'})
  Name: string;

  @Field({ type: 'varchar', size: 24})
  Symbol?: string;

  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => UoMCategory },
    notNull: true, index: true, uniqueIndex: 'uidx_base_uom_category_name',
  })
  CategoryId: UoMCategory;

  @Field({ type: 'boolean', notNull: true, default: () => false})
  IsReference: boolean;

  @Field({ type: 'decimal', notNull: true, precision: 38, scale: 18})
  Factor: any;

  @Field({ type: 'decimal', precision: 38, scale: 18})
  Rounding?: any;

  @Field({ type: 'boolean', notNull: true, default: () => true, index: true})
  IsActive: boolean;

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
    if (conflict) fail(_t('UoM Name must be unique within Category', { scope: 'service/models/uom' }));
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
        fail(_t('Each UoM category can only have one reference unit', { scope: 'service/models/uom' }));
      }
      return;
    }

    if (refsExcludingCurrent.length === 0) {
      fail(_t('Each UoM category must have one reference unit', { scope: 'service/models/uom' }));
    }
  }

  private static async validateEntity(values: Record<string, any>, currentId?: string): Promise<void> {
    const name = normalizeName(values.Name);
    const categoryId = requireRefId(values.CategoryId, 'CategoryId');

    const isRef = values.IsReference === true;
    const factor = mapNormalizationToBase(
      () => toPositiveDecimal(values.Factor),
      err =>
        err.code === 'non_positive_decimal'
          ? _t('Factor must be greater than 0', { scope: 'service/models/uom' })
          : _t('Factor must be a valid decimal', { scope: 'service/models/uom' })
    );
    if (isRef && !factor.eq(1)) {
      fail(_t('Reference UoM must have Factor = 1', { scope: 'service/models/uom' }));
    }

    if (values.Rounding !== undefined && values.Rounding !== null && values.Rounding !== '') {
      const rounding = mapNormalizationToBase(
        () => toPositiveDecimal(values.Rounding),
        err =>
          err.code === 'non_positive_decimal'
            ? _t('Rounding must be greater than 0', { scope: 'service/models/uom' })
            : _t('Rounding must be a valid decimal', { scope: 'service/models/uom' })
      );
      values.Rounding = rounding.toString();
    } else {
      values.Rounding = null;
    }

    values.CategoryId = categoryId;
    values.Name = name;
    values.IsReference = isRef;
    values.Factor = factor.toString();

    await this.ensureNameUnique(categoryId, name, currentId);
    await this.ensureCategoryReferenceInvariant(categoryId, isRef, currentId);
  }

  @Constraint<UoM>(['Name', 'CategoryId', 'IsReference', 'Factor', 'Rounding'])
  async validateUoMConstraint(): Promise<void> {
    const currentId = String((this as any).Id || '').trim() || undefined;
    await UoM.validateEntity(this as any, currentId);
  }
}

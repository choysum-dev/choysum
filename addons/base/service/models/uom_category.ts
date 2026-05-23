// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { Constraint } from '@/core/service/api/constraint';

@Model('UoMCategory')
export default class UoMCategory extends BaseModel {
  @Field({ type: 'varchar', column: { size: 100, notNull: true, index: true } })
  Name: string;

  @Field({ type: 'varchar', column: { size: 40, unique: true, index: true } })
  Code?: string;

  @Field({ type: 'boolean', column: { notNull: true, default: () => true, index: true } })
  IsActive: boolean;

  private static normalizeCode(value: unknown): string | null | undefined {
    if (value === undefined) return undefined;
    if (value === null) return null;
    const code = String(value ?? '')
      .trim()
      .toUpperCase();
    return code ? code : null;
  }

  private static validateEntity(values: Record<string, any>): void {
    if (Object.prototype.hasOwnProperty.call(values, 'Code')) {
      values.Code = this.normalizeCode(values.Code);
    }
  }

  @Constraint<UoMCategory>(['Code'])
  static validateUoMCategoryConstraint(self: UoMCategory, ctx: any): void {
    const values = (ctx?.values || {}) as Record<string, any>;
    UoMCategory.validateEntity(self as any);

    if (Object.prototype.hasOwnProperty.call(values, 'Code')) {
      values.Code = self.Code;
    }
  }
}

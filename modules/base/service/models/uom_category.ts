// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { Constraint } from '@/core/service/api/constraint';
import { normalizeCodeOptional } from './_normalizers';

@Model('UoMCategory')
export default class UoMCategory extends BaseModel {
  @Field({ type: 'varchar', size: 100, notNull: true, index: true})
  Name: string;

  @Field({ type: 'varchar', size: 40, unique: true, index: true})
  Code?: string;

  @Field({ type: 'boolean', notNull: true, default: () => true, index: true})
  IsActive: boolean;

  @Constraint<UoMCategory>(['Code'])
  validateUoMCategoryConstraint(): void {
    if (this.Code != null) {
      (this as any).Code = normalizeCodeOptional(this.Code as string);
    }
  }
}

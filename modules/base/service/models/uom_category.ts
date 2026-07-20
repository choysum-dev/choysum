// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { Constraint } from '@/core/service/api/constraint';
import { _lt } from '../i18n';
import { normalizeCodeOptional } from './_normalizers';

@Model('UoMCategory')
export default class UoMCategory extends BaseModel {
  @Field({
    type: 'varchar',
    size: 100,
    notNull: true,
    index: true,
    string: _lt('Name', { scope: 'base.model.UoMCategory.fields' }),
  })
  Name: string;

  @Field({
    type: 'varchar',
    size: 40,
    unique: true,
    index: true,
    string: _lt('Code', { scope: 'base.model.UoMCategory.fields' }),
  })
  Code?: string;

  @Field({
    type: 'boolean',
    notNull: true,
    default: () => true,
    index: true,
    string: _lt('Active', { scope: 'base.model.UoMCategory.fields' }),
  })
  IsActive: boolean;

  @Constraint<UoMCategory>(['Code'])
  validateUoMCategoryConstraint(): void {
    if (this.Code != null) {
      (this as any).Code = normalizeCodeOptional(this.Code as string);
    }
  }
}

// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { _lt } from '../i18n';

/** Hierarchical category for M2O condition + M2MRef tree hand tests. */
@Model('SandboxCategory')
export default class SandboxCategory extends BaseModel {
  @Field({
    type: 'varchar',
    size: 100,
    notNull: true,
    index: 'trigram',
    string: _lt('Name', { scope: 'sandbox.model.SandboxCategory.fields' }),
  })
  Name: string;

  @Field({
    type: 'varchar',
    size: 40,
    notNull: true,
    unique: true,
    index: true,
    string: _lt('Code', { scope: 'sandbox.model.SandboxCategory.fields' }),
  })
  Code: string;

  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => SandboxCategory },
    condition: ['IsActive', '=', true],
    index: true,
    string: _lt('Parent', { scope: 'sandbox.model.SandboxCategory.fields' }),
  })
  ParentId?: SandboxCategory;

  @Field({
    type: 'OneToMany',
    relation: { targetModel: () => SandboxCategory, inverseField: 'ParentId' },
    condition: ['IsActive', '=', true],
    string: _lt('Children', { scope: 'sandbox.model.SandboxCategory.fields' }),
  })
  Children?: SandboxCategory[];

  @Field({
    type: 'boolean',
    notNull: true,
    default: () => true,
    index: true,
    string: _lt('Active', { scope: 'sandbox.model.SandboxCategory.fields' }),
  })
  IsActive: boolean;
}

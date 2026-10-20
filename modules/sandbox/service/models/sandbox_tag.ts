// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { _lt } from '../i18n';

/** Tag master for ManyToMany and ManyToManyRef Tags widget. */
@Model('SandboxTag')
export default class SandboxTag extends BaseModel {
  @Field({
    type: 'varchar',
    size: 80,
    notNull: true,
    index: 'trigram',
    string: _lt('Name', { scope: 'sandbox.model.SandboxTag.fields' }),
  })
  Name: string;

  @Field({
    type: 'varchar',
    size: 40,
    notNull: true,
    unique: true,
    index: true,
    string: _lt('Code', { scope: 'sandbox.model.SandboxTag.fields' }),
  })
  Code: string;

  @Field({
    type: 'boolean',
    notNull: true,
    default: () => true,
    index: true,
    string: _lt('Active', { scope: 'sandbox.model.SandboxTag.fields' }),
  })
  IsActive: boolean;
}

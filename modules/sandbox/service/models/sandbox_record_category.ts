// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { _lt } from '../i18n';
import SandboxCategory from './sandbox_category';
import SandboxRecord from './sandbox_record';

/** Join row for SandboxRecord ↔ SandboxCategory ManyToMany (Tree widget). */
@Model('SandboxRecordCategory')
export default class SandboxRecordCategory extends BaseModel {
  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => SandboxRecord },
    notNull: true,
    index: true,
    uniqueIndex: 'uidx_sandbox_record_category',
    string: _lt('Record', { scope: 'sandbox.model.SandboxRecordCategory.fields' }),
  })
  RecordId: SandboxRecord;

  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => SandboxCategory },
    notNull: true,
    index: true,
    uniqueIndex: 'uidx_sandbox_record_category',
    condition: ['IsActive', '=', true],
    string: _lt('Category', { scope: 'sandbox.model.SandboxRecordCategory.fields' }),
  })
  CategoryId: SandboxCategory;
}

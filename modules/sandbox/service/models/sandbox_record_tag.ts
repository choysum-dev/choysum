// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { _lt } from '../i18n';
import SandboxRecord from './sandbox_record';
import SandboxTag from './sandbox_tag';

/**
 * Join row for SandboxRecord ↔ SandboxTag ManyToMany.
 * Pure FK-pair link: softDelete false so unlink physically removes the row.
 */
@Model('SandboxRecordTag', { softDelete: false })
export default class SandboxRecordTag extends BaseModel {
  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => SandboxRecord },
    notNull: true,
    index: true,
    uniqueIndex: 'uidx_sandbox_record_tag',
    string: _lt('Record', { scope: 'sandbox.model.SandboxRecordTag.fields' }),
  })
  RecordId: SandboxRecord;

  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => SandboxTag },
    notNull: true,
    index: true,
    uniqueIndex: 'uidx_sandbox_record_tag',
    condition: ['IsActive', '=', true],
    string: _lt('Tag', { scope: 'sandbox.model.SandboxRecordTag.fields' }),
  })
  TagId: SandboxTag;
}

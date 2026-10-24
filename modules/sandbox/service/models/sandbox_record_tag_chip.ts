// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { _lt } from '../i18n';
import SandboxRecord from './sandbox_record';
import SandboxTag from './sandbox_tag';

/** Join row for SandboxRecord ↔ SandboxTag chips ManyToMany (Tags widget). */
@Model('SandboxRecordTagChip', { softDelete: false })
export default class SandboxRecordTagChip extends BaseModel {
  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => SandboxRecord },
    notNull: true,
    index: true,
    uniqueIndex: 'uidx_sandbox_record_tag_chip',
    string: _lt('Record', { scope: 'sandbox.model.SandboxRecordTagChip.fields' }),
  })
  RecordId: SandboxRecord;

  @Field({
    type: 'ManyToOne',
    relation: { targetModel: () => SandboxTag },
    notNull: true,
    index: true,
    uniqueIndex: 'uidx_sandbox_record_tag_chip',
    condition: ['IsActive', '=', true],
    string: _lt('Tag', { scope: 'sandbox.model.SandboxRecordTagChip.fields' }),
  })
  TagId: SandboxTag;
}

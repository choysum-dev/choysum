// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { _lt } from '../i18n';

/** Default V1 subtype internal name for comment/discussion notifications. */
export const MESSAGE_SUBTYPE_DISCUSSIONS = 'discussions';

/**
 * Message subtype for follower subscription granularity (Odoo mail.message.subtype analogue).
 * Table: message_message_subtype.
 */
@Model('MessageSubtype', {
  application: 'message',
  softDelete: false,
  orderBy: { field: 'InternalName', order: 'asc' },
})
export default class MessageSubtype extends BaseModel {
  @Field({
    type: 'varchar',
    size: 64,
    notNull: true,
    index: true,
    unique: true,
    string: _lt('Internal Name', { scope: 'message.model.MessageSubtype.fields' }),
  })
  InternalName: string;

  @Field({
    type: 'varchar',
    size: 255,
    notNull: true,
    string: _lt('Name', { scope: 'message.model.MessageSubtype.fields' }),
  })
  Name: string;

  @Field({
    type: 'text',
    string: _lt('Description', { scope: 'message.model.MessageSubtype.fields' }),
  })
  Description: string | null;
}

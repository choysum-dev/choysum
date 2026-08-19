// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { getUserId } from '@/core/service/api/context';
import type { FieldSelection } from '@/core/service/api/selection';
import type { QueryCondition, SearchOptions } from '@/core/service/api/query';
import { MessageErrCode, newMessageError } from '../error';
import { _lt } from '../i18n';
import type MessageSubtype from './message_subtype';

export type FollowRecordReq = {
  Model: string;
  ResId: string;
  UserId?: string | null;
  SubtypeId?: string | null;
  CompanyId?: string | null;
};

export type UnfollowRecordReq = {
  Model: string;
  ResId: string;
  UserId?: string | null;
};

function resolveActorUserId(explicit?: string | null): string | null {
  const fromReq = explicit == null || explicit === '' ? null : String(explicit).trim();
  if (fromReq) return fromReq;
  const uid = getUserId();
  if (uid == null || String(uid).trim() === '') return null;
  return String(uid).trim();
}

/**
 * Record follower subscription for Chatter / notification fan-out.
 * Table: message_follower.
 */
@Model('Follower', {
  application: 'message',
  softDelete: true,
  companyField: 'CompanyId',
  orderBy: { field: 'CreatedAt', order: 'asc' },
})
export default class Follower extends BaseModel {
  @Field({
    type: 'varchar',
    size: 120,
    notNull: true,
    index: true,
    uniqueIndex: 'uidx_message_follower_record_user',
    string: _lt('Model', { scope: 'message.model.Follower.fields' }),
  })
  Model: string;

  @Field({
    type: 'char',
    size: 20,
    notNull: true,
    index: true,
    uniqueIndex: 'uidx_message_follower_record_user',
    string: _lt('Record', { scope: 'message.model.Follower.fields' }),
  })
  ResId: string;

  @Field({
    type: 'ManyToOneRef',
    relation: { targetModel: 'auth.User' },
    size: 20,
    notNull: true,
    index: true,
    uniqueIndex: 'uidx_message_follower_record_user',
    string: _lt('User', { scope: 'message.model.Follower.fields' }),
  })
  UserId: string;

  @Field<MessageSubtype>({
    type: 'ManyToOneRef',
    relation: { targetModel: 'message.MessageSubtype' },
    size: 20,
    index: true,
    string: _lt('Subtype', { scope: 'message.model.Follower.fields' }),
  })
  SubtypeId: string | null;

  @Field({
    type: 'char',
    size: 20,
    index: true,
    string: _lt('Company', { scope: 'message.model.Follower.fields' }),
  })
  CompanyId: string | null;

  /**
   * Subscribe the current (or explicit) user to one business record thread.
   * Idempotent when the same (Model, ResId, UserId) row already exists.
   */
  public static async Follow(req: FollowRecordReq, fields?: FieldSelection<Follower>): Promise<Follower> {
    if (!req || typeof req !== 'object') {
      throw newMessageError({ code: MessageErrCode.INVALID_ARGUMENT, message: 'Follow requires a payload' });
    }
    const model = String(req.Model || '').trim();
    const resId = String(req.ResId || '').trim();
    const userId = resolveActorUserId(req.UserId);
    if (!model || !resId) {
      throw newMessageError({ code: MessageErrCode.INVALID_ARGUMENT, message: 'Follow requires Model and ResId' });
    }
    if (!userId) {
      throw newMessageError({ code: MessageErrCode.INVALID_ARGUMENT, message: 'Follow requires a user identity' });
    }

    const existing = await this.Search(
      {
        And: [
          ['Model', '=', model],
          ['ResId', '=', resId],
          ['UserId', '=', userId],
        ],
      },
      { fields: fields ?? ['Id', 'Model', 'ResId', 'UserId', 'SubtypeId', 'CompanyId'], limit: 1 }
    );
    if (existing.length > 0) {
      return existing[0] as Follower;
    }

    const subtypeId = req.SubtypeId == null || req.SubtypeId === '' ? null : String(req.SubtypeId);
    const companyId = req.CompanyId == null || req.CompanyId === '' ? null : String(req.CompanyId);
    return await this.Create(
      {
        Model: model,
        ResId: resId,
        UserId: userId,
        SubtypeId: subtypeId,
        CompanyId: companyId,
      },
      fields ?? ['Id', 'Model', 'ResId', 'UserId', 'SubtypeId', 'CompanyId']
    );
  }

  /**
   * Remove one follower row for the current (or explicit) user on a record.
   */
  public static async Unfollow(req: UnfollowRecordReq): Promise<number> {
    if (!req || typeof req !== 'object') {
      throw newMessageError({ code: MessageErrCode.INVALID_ARGUMENT, message: 'Unfollow requires a payload' });
    }
    const model = String(req.Model || '').trim();
    const resId = String(req.ResId || '').trim();
    const userId = resolveActorUserId(req.UserId);
    if (!model || !resId) {
      throw newMessageError({ code: MessageErrCode.INVALID_ARGUMENT, message: 'Unfollow requires Model and ResId' });
    }
    if (!userId) {
      throw newMessageError({ code: MessageErrCode.INVALID_ARGUMENT, message: 'Unfollow requires a user identity' });
    }

    const rows = await this.Search(
      {
        And: [
          ['Model', '=', model],
          ['ResId', '=', resId],
          ['UserId', '=', userId],
        ],
      },
      { fields: ['Id'], limit: 100 }
    );
    let deleted = 0;
    for (const row of rows) {
      const id = String((row as Follower).Id || '').trim();
      if (!id) continue;
      await this.DeleteById(id);
      deleted += 1;
    }
    return deleted;
  }

  /**
   * Lists followers for one target record.
   */
  public static async SearchByRecord(
    model: string,
    resId: string,
    fields?: FieldSelection<Follower>
  ): Promise<Partial<Follower>[]> {
    const m = String(model || '').trim();
    const id = String(resId || '').trim();
    if (!m || !id) {
      throw newMessageError({
        code: MessageErrCode.INVALID_ARGUMENT,
        message: 'SearchByRecord requires Model and ResId',
      });
    }
    const condition: QueryCondition<Follower> = {
      And: [
        ['Model', '=', m],
        ['ResId', '=', id],
      ],
    };
    const options: SearchOptions<Follower> = {
      fields,
      orderBy: { field: 'CreatedAt', order: 'asc' },
    };
    return await this.Search(condition, options);
  }
}

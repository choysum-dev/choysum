// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { getUserId } from '@/core/service/api/context';
import type { FieldSelection } from '@/core/service/api/selection';
import type { QueryCondition, SearchOptions } from '@/core/service/api/query';
import { MessageErrCode, newMessageError } from '../error';
import { _lt } from '../i18n';
import { assertTargetRecordReadable } from '../target_record';
import type MessageSubtype from './message_subtype';

export {
  __setMessageFollowDialForTest,
  __setMessageFollowTargetAuthForTest,
} from '../target_record';

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

function isUniqueConstraintError(err: unknown): boolean {
  const msg =
    err instanceof Error
      ? err.message
      : typeof err === 'object' && err !== null && 'message' in err
        ? String((err as { message: unknown }).message)
        : String(err);
  return /unique constraint|unique index|duplicate key|duplicate entry|UNIQUE constraint failed/i.test(msg);
}

const DEFAULT_FOLLOW_FIELDS: FieldSelection<Follower> = ['Id', 'Model', 'ResId', 'UserId', 'SubtypeId', 'CompanyId'];

function followReturnFields(fields?: FieldSelection<Follower>): FieldSelection<Follower> {
  return fields ?? DEFAULT_FOLLOW_FIELDS;
}

function withFollowLookupFields(fields: FieldSelection<Follower>): FieldSelection<Follower> {
  const next = [...(fields as string[])];
  for (const extra of ['Id', 'SubtypeId', 'CompanyId', 'DeletedAt']) {
    if (!next.includes('*') && !next.includes(extra)) next.push(extra);
  }
  return next as FieldSelection<Follower>;
}

function projectFollowRow(row: Follower, fields: FieldSelection<Follower>): Follower {
  const requested = fields as string[];
  if (requested.includes('*')) {
    const { DeletedAt: _deletedAt, ...next } = row as Follower & { DeletedAt?: unknown };
    void _deletedAt;
    return next as Follower;
  }
  const source = row as unknown as Record<string, unknown>;
  const next: Record<string, unknown> = {};
  for (const key of requested) {
    if (key in source) next[key] = source[key];
  }
  return next as unknown as Follower;
}

function nullableId(value: unknown): string | null {
  if (value == null || value === '') return null;
  const normalized = String(value).trim();
  return normalized ? normalized : null;
}

async function findFollowRow(
  model: string,
  resId: string,
  userId: string,
  fields: FieldSelection<Follower>
): Promise<Follower | null> {
  const rows = await Follower.Search(
    {
      And: [
        ['Model', '=', model],
        ['ResId', '=', resId],
        ['UserId', '=', userId],
      ],
    },
    { fields: withFollowLookupFields(fields), limit: 1, withDeleted: true }
  );
  return rows.length > 0 ? (rows[0] as Follower) : null;
}

function isDeletedFollower(row: Follower): boolean {
  const deletedAt = (row as Follower & { DeletedAt?: Date | string | number | null }).DeletedAt;
  if (deletedAt == null || deletedAt === '') return false;
  if (deletedAt instanceof Date) return !Number.isNaN(deletedAt.getTime());
  return true;
}

async function syncFollowRow(
  row: Follower,
  subtypeId: string | null,
  companyId: string | null,
  fields: FieldSelection<Follower>,
  restoreDeleted: boolean
): Promise<Follower> {
  const id = String(row.Id || '').trim();
  if (!id) {
    throw newMessageError({ code: MessageErrCode.INVALID_ARGUMENT, message: 'Follow requires a follower id' });
  }
  const currentSubtype = nullableId((row as Follower).SubtypeId);
  const currentCompany = nullableId((row as Follower).CompanyId);
  if (!restoreDeleted && currentSubtype === subtypeId && currentCompany === companyId) {
    return projectFollowRow(row, fields);
  }
  const values: Record<string, unknown> = { SubtypeId: subtypeId, CompanyId: companyId };
  if (restoreDeleted) values.DeletedAt = null;
  const updated = await Follower.UpdateById(id, values as any, fields, restoreDeleted ? { withDeleted: true } : undefined);
  return projectFollowRow(updated as Follower, fields);
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
   * Idempotent when the same (Model, ResId, UserId) row already exists, including
   * restoring a soft-deleted follower when the unique index is still occupied.
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
    await assertTargetRecordReadable(model, resId, 'Follow is not allowed for this record');

    const returnFields = followReturnFields(fields);
    const subtypeId = req.SubtypeId == null || req.SubtypeId === '' ? null : String(req.SubtypeId);
    const companyId = req.CompanyId == null || req.CompanyId === '' ? null : String(req.CompanyId);

    const existing = await findFollowRow(model, resId, userId, returnFields);
    if (existing) {
      return await syncFollowRow(existing, subtypeId, companyId, returnFields, isDeletedFollower(existing));
    }

    try {
      return await this.Create(
        {
          Model: model,
          ResId: resId,
          UserId: userId,
          SubtypeId: subtypeId,
          CompanyId: companyId,
        },
        returnFields
      );
    } catch (err) {
      if (!isUniqueConstraintError(err)) throw err;
      const raced = await findFollowRow(model, resId, userId, returnFields);
      if (!raced) throw err;
      return await syncFollowRow(raced, subtypeId, companyId, returnFields, isDeletedFollower(raced));
    }
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

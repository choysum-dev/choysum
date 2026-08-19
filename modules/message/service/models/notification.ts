// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Field, Model } from '@/core/service';
import { getUserId } from '@/core/service/api/context';
import type { Insertable } from '@/core/service/api/input';
import type { FieldSelection } from '@/core/service/api/selection';
import type { QueryCondition, SearchOptions } from '@/core/service/api/query';
import { MessageErrCode, newMessageError } from '../error';
import { publishNotificationUserTips } from '../tips';
import { _lt } from '../i18n';
import Follower from './follower';
import MessageSubtype, { MESSAGE_SUBTYPE_DISCUSSIONS } from './message_subtype';
import type Message from './message';

export type SearchInboxOptions = {
  unreadOnly?: boolean;
  limit?: number;
  fields?: FieldSelection<Notification>;
};

let markAllReadBatchSize = 500;
let markAllReadMaxIterations = 1000;

/**
 * Test-only override for MarkAllRead Search batch size.
 */
export function __setNotificationMarkAllReadBatchSizeForTest(size: number | undefined): void {
  markAllReadBatchSize = size && size > 0 ? size : 500;
}

/**
 * Test-only override for MarkAllRead loop cap. 0 runs no batches (hits the cap return).
 */
export function __setNotificationMarkAllReadMaxIterationsForTest(size: number | undefined): void {
  markAllReadMaxIterations = size == null || size < 0 ? 1000 : size;
}

async function resolveDiscussionsSubtypeId(): Promise<string | null> {
  const rows = await MessageSubtype.sudo(
    () =>
      MessageSubtype.Search(['InternalName', '=', MESSAGE_SUBTYPE_DISCUSSIONS], {
        fields: ['Id'],
        limit: 1,
      }),
    { hint: 'message.Notification.fanOut.discussionsSubtype' }
  );
  const id = String((rows[0] as MessageSubtype | undefined)?.Id || '').trim();
  return id || null;
}

function followerMatchesDiscussions(row: Follower, discussionsId: string | null): boolean {
  const subtypeId = String(row.SubtypeId || '').trim();
  if (!subtypeId) return true;
  if (!discussionsId) return true;
  return subtypeId === discussionsId;
}

function resolveInboxUserId(): string {
  const uid = getUserId();
  const normalized = uid == null ? '' : String(uid).trim();
  if (!normalized) {
    throw newMessageError({ code: MessageErrCode.INVALID_ARGUMENT, message: 'SearchInbox requires a user identity' });
  }
  return normalized;
}

/**
 * In-app notification row for one recipient about one Message on a thread.
 * Table: message_notification.
 */
@Model('Notification', {
  application: 'message',
  softDelete: false,
  companyField: 'CompanyId',
  orderBy: { field: 'CreatedAt', order: 'desc' },
})
export default class Notification extends BaseModel {
  @Field({
    type: 'ManyToOneRef',
    relation: { targetModel: 'auth.User' },
    size: 20,
    notNull: true,
    index: true,
    string: _lt('User', { scope: 'message.model.Notification.fields' }),
  })
  UserId: string;

  @Field<Message>({
    type: 'ManyToOneRef',
    relation: { targetModel: 'message.Message' },
    size: 20,
    notNull: true,
    index: true,
    string: _lt('Message', { scope: 'message.model.Notification.fields' }),
  })
  MessageId: string;

  @Field({
    type: 'varchar',
    size: 120,
    notNull: true,
    index: true,
    string: _lt('Model', { scope: 'message.model.Notification.fields' }),
  })
  Model: string;

  @Field({
    type: 'char',
    size: 20,
    notNull: true,
    index: true,
    string: _lt('Record', { scope: 'message.model.Notification.fields' }),
  })
  ResId: string;

  @Field({
    type: 'ManyToOneRef',
    relation: { targetModel: 'auth.User' },
    size: 20,
    index: true,
    string: _lt('Author', { scope: 'message.model.Notification.fields' }),
  })
  AuthorUid: string | null;

  @Field({
    type: 'boolean',
    notNull: true,
    default: () => false,
    index: true,
    string: _lt('Read', { scope: 'message.model.Notification.fields' }),
  })
  IsRead: boolean;

  @Field({
    type: 'char',
    size: 20,
    index: true,
    string: _lt('Company', { scope: 'message.model.Notification.fields' }),
  })
  CompanyId: string | null;

  /**
   * Lists inbox notifications for the authenticated user (newest first).
   */
  public static async SearchInbox(options?: SearchInboxOptions): Promise<Partial<Notification>[]> {
    const userId = resolveInboxUserId();
    const conditionParts: QueryCondition<Notification>[] = [['UserId', '=', userId]];
    if (options?.unreadOnly) {
      conditionParts.push(['IsRead', '=', false]);
    }
    const condition: QueryCondition<Notification> =
      conditionParts.length === 1 ? conditionParts[0] : { And: conditionParts };
    const searchOptions: SearchOptions<Notification> = {
      fields: options?.fields ?? ['Id', 'MessageId', 'Model', 'ResId', 'AuthorUid', 'IsRead', 'CreatedAt'],
      orderBy: { field: 'CreatedAt', order: 'desc' },
    };
    if (options?.limit != null && options.limit > 0) {
      searchOptions.limit = options.limit;
    }
    return await this.Search(condition, searchOptions);
  }

  /**
   * Marks selected inbox rows read for the authenticated user.
   */
  public static async MarkRead(notificationIds: string[]): Promise<number> {
    const userId = resolveInboxUserId();
    const ids = (notificationIds || []).map(id => String(id || '').trim()).filter(Boolean);
    if (ids.length === 0) {
      throw newMessageError({ code: MessageErrCode.INVALID_ARGUMENT, message: 'MarkRead requires notification ids' });
    }

    const rows = await this.Search(
      {
        And: [
          ['UserId', '=', userId],
          ['Id', 'in', ids],
        ],
      },
      { fields: ['Id', 'IsRead'], limit: ids.length }
    );
    let updated = 0;
    for (const row of rows) {
      const id = String((row as Notification).Id || '').trim();
      if (!id || (row as Notification).IsRead === true) continue;
      await this.UpdateById(id, { IsRead: true } as Partial<Insertable<Notification>>, ['Id', 'IsRead']);
      updated += 1;
    }
    return updated;
  }

  /**
   * Marks every unread inbox row read for the authenticated user.
   */
  public static async MarkAllRead(): Promise<number> {
    const userId = resolveInboxUserId();
    let updated = 0;
    let previousKey = '';
    for (let iteration = 0; iteration < markAllReadMaxIterations; iteration += 1) {
      const rows = await this.Search(
        {
          And: [
            ['UserId', '=', userId],
            ['IsRead', '=', false],
          ],
        },
        { fields: ['Id'], limit: markAllReadBatchSize }
      );
      if (rows.length === 0) return updated;
      const key = rows.map(row => String((row as Notification).Id || '').trim()).join('\0');
      if (key === previousKey) return updated;
      previousKey = key;
      let batchUpdated = 0;
      for (const row of rows) {
        const id = String((row as Notification).Id || '').trim();
        if (!id) continue;
        await this.UpdateById(id, { IsRead: true } as Partial<Insertable<Notification>>, ['Id', 'IsRead']);
        updated += 1;
        batchUpdated += 1;
      }
      if (batchUpdated === 0) return updated;
    }
    return updated;
  }

  /**
   * Creates Notification rows for record followers and best-effort inbox tips.
   * Called from Message.Post after the Message row is authoritative.
   */
  public static async FanOutForMessage(created: Partial<Message>): Promise<void> {
    const messageId = String(created.Id || '').trim();
    const model = String(created.Model || '').trim();
    const resId = String(created.ResId || '').trim();
    const authorUid = String(created.AuthorUid || '').trim();
    if (!messageId || !model || !resId) return;

    const followers = await Follower.sudo(
      () => Follower.SearchByRecord(model, resId, ['UserId', 'SubtypeId']),
      { hint: 'message.Notification.fanOut.searchFollowers' }
    );
    const discussionsId = await resolveDiscussionsSubtypeId();
    const recipientIds = new Set<string>();
    for (const row of followers) {
      const uid = String((row as Follower).UserId || '').trim();
      if (!uid || uid === authorUid) continue;
      if (!followerMatchesDiscussions(row as Follower, discussionsId)) continue;
      recipientIds.add(uid);
    }
    if (recipientIds.size === 0) return;

    const companyId = created.CompanyId == null || created.CompanyId === '' ? null : String(created.CompanyId);
    await this.sudo(
      () =>
        this.CreateMany(
          [...recipientIds].map(userId => ({
            UserId: userId,
            MessageId: messageId,
            Model: model,
            ResId: resId,
            AuthorUid: authorUid || null,
            CompanyId: companyId,
            IsRead: false,
          })),
          ['Id', 'UserId']
        ),
      { hint: 'message.Notification.fanOut.create' }
    );

    await publishNotificationUserTips(recipientIds, (created as Message).CreatedAt);
  }
}

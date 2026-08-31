// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel } from '@/core/service';
import type { FieldSelection } from '@/core/service/api/selection';
import Follower, { type FollowRecordReq, type UnfollowRecordReq } from '../models/follower';
import Message, { type PostMessageReq } from '../models/message';

/**
 * Opt-in mixin for business models that participate in a message thread
 * (Odoo `mail.thread` style). Other apps extend this default-exported base:
 *
 * ```ts
 * import MessageThreadModel from '@/message/service/mixins/message_thread_model';
 * @Model('Partner', { application: 'partner' })
 * export default class Partner extends MessageThreadModel {}
 * ```
 *
 * Implementation dials message.Message / message.Follower — does not live on BaseModel.
 */
export default abstract class MessageThreadModel extends BaseModel {
  /** Post a collaboration message on a business record (Unary). */
  public static async MessagePost(req: PostMessageReq, fields?: FieldSelection<Message>): Promise<Message> {
    return Message.Post(req, fields);
  }

  /** List messages for one business record. */
  public static async MessageSearchByRecord(
    model: string,
    resId: string,
    fields?: FieldSelection<Message>
  ): Promise<Partial<Message>[]> {
    return Message.SearchByRecord(model, resId, fields);
  }

  /** Subscribe a user to a business record thread. */
  public static async MessageFollow(req: FollowRecordReq, fields?: FieldSelection<Follower>): Promise<Follower> {
    return Follower.Follow(req, fields);
  }

  /** Remove a follower from a business record thread. */
  public static async MessageUnfollow(req: UnfollowRecordReq): Promise<number> {
    return Follower.Unfollow(req);
  }

  /** List followers for one business record. */
  public static async MessageSearchFollowersByRecord(
    model: string,
    resId: string,
    fields?: FieldSelection<Follower>
  ): Promise<Partial<Follower>[]> {
    return Follower.SearchByRecord(model, resId, fields);
  }
}

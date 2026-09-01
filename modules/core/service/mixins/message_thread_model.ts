// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { FieldSelection } from '../api/selection';
import BaseModel from '../orm/model/model';
import { dial } from '../orm/model/model_pool';
import type {
  MessageThreadFollowReq,
  MessageThreadPostReq,
  MessageThreadUnfollowReq,
} from './message_thread_contracts';

type MessageService = {
  Post(req: MessageThreadPostReq, fields?: FieldSelection<any>): Promise<any>;
  SearchByRecord(model: string, resId: string, fields?: FieldSelection<any>): Promise<Partial<any>[]>;
};

type FollowerService = {
  Follow(req: MessageThreadFollowReq, fields?: FieldSelection<any>): Promise<any>;
  Unfollow(req: MessageThreadUnfollowReq): Promise<number>;
  SearchByRecord(model: string, resId: string, fields?: FieldSelection<any>): Promise<Partial<any>[]>;
};

/**
 * Opt-in dial facade for business models that participate in a message thread
 * (Odoo `mail.thread` style).
 *
 * Lives in core so business apps may `extends` it under the hard rule that
 * cross-Application value imports are forbidden except from core. Runtime calls
 * dial `message.Message` / `message.Follower`. Attachment owner APIs stay on
 * {@link AttachmentOwnerMixin}; compose in the consumer when both are needed.
 *
 * ```ts
 * import MessageThreadModel from '@/core/service/mixins/message_thread_model';
 * @Model('Partner', { application: 'partner' })
 * export default class Partner extends MessageThreadModel {}
 * ```
 */
export default abstract class MessageThreadModel extends BaseModel {
  /** Post a collaboration message on a business record (Unary). */
  public static async MessagePost(req: MessageThreadPostReq, fields?: FieldSelection<any>): Promise<any> {
    return dial<MessageService>('message.Message').Post(req, fields);
  }

  /** List messages for one business record. */
  public static async MessageSearchByRecord(
    model: string,
    resId: string,
    fields?: FieldSelection<any>
  ): Promise<Partial<any>[]> {
    return dial<MessageService>('message.Message').SearchByRecord(model, resId, fields);
  }

  /** Subscribe a user to a business record thread. */
  public static async MessageFollow(req: MessageThreadFollowReq, fields?: FieldSelection<any>): Promise<any> {
    return dial<FollowerService>('message.Follower').Follow(req, fields);
  }

  /** Remove a follower from a business record thread. */
  public static async MessageUnfollow(req: MessageThreadUnfollowReq): Promise<number> {
    return dial<FollowerService>('message.Follower').Unfollow(req);
  }

  /** List followers for one business record. */
  public static async MessageSearchFollowersByRecord(
    model: string,
    resId: string,
    fields?: FieldSelection<any>
  ): Promise<Partial<any>[]> {
    return dial<FollowerService>('message.Follower').SearchByRecord(model, resId, fields);
  }
}

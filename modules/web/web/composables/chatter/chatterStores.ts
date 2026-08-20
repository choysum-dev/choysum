// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createStoreByModel } from '@/web/web/stores/registry';
import type { PostMessageReq } from '@/message/service/models/message';
import type { FollowRecordReq, UnfollowRecordReq } from '@/message/service/models/follower';
import type { SearchInboxOptions } from '@/message/service/models/notification';
import type {
  ChatterFieldChangeRow,
  ChatterMessageRow,
  InboxNotificationRow,
} from './chatterTypes';

type MessageStoreLike = {
  Post: (req: PostMessageReq) => Promise<ChatterMessageRow>;
  SearchByRecord: (model: string, resId: string, fields: readonly string[]) => Promise<ChatterMessageRow[]>;
};

type FieldChangeStoreLike = {
  SearchByRecord: (model: string, resId: string, fields: readonly string[]) => Promise<ChatterFieldChangeRow[]>;
};

type FollowerStoreLike = {
  Follow: (req: FollowRecordReq) => Promise<{ UserId?: string | null }>;
  Unfollow: (req: UnfollowRecordReq) => Promise<number>;
  SearchByRecord: (model: string, resId: string, fields: readonly string[]) => Promise<Array<{ UserId?: string | null }>>;
};

type NotificationStoreLike = {
  SearchInbox: (options?: SearchInboxOptions) => Promise<InboxNotificationRow[]>;
  MarkRead: (notificationIds: string[]) => Promise<number>;
  MarkAllRead: () => Promise<number>;
};

export function getMessageStore(): MessageStoreLike {
  return createStoreByModel('message.Message') as unknown as MessageStoreLike;
}

export function getFieldChangeStore(): FieldChangeStoreLike {
  return createStoreByModel('audit.FieldChange') as unknown as FieldChangeStoreLike;
}

export function getFollowerStore(): FollowerStoreLike {
  return createStoreByModel('message.Follower') as unknown as FollowerStoreLike;
}

export function getNotificationStore(): NotificationStoreLike {
  return createStoreByModel('message.Notification') as unknown as NotificationStoreLike;
}

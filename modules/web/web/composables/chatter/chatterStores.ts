// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { inject, type InjectionKey } from 'vue';
import { createStoreByModel as defaultCreateStoreByModel } from '@/web/web/stores/registry';
import type { PostMessageReq } from '@/message/service/models/message';
import type { FollowRecordReq, UnfollowRecordReq } from '@/message/service/models/follower';
import type { SearchInboxOptions } from '@/message/service/models/notification';
import type {
  ChatterFieldChangeRow,
  ChatterMessageRow,
  InboxNotificationRow,
} from './chatterTypes';

type CreateStoreByModel = typeof defaultCreateStoreByModel;

export type ChatterStoreDeps = {
  createStoreByModel?: CreateStoreByModel;
};

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

function createStore(modelName: string, deps?: ChatterStoreDeps) {
  const create = deps?.createStoreByModel ?? defaultCreateStoreByModel;
  return create(modelName);
}

export function getMessageStore(deps?: ChatterStoreDeps): MessageStoreLike {
  return createStore('message.Message', deps) as unknown as MessageStoreLike;
}

export function getFieldChangeStore(deps?: ChatterStoreDeps): FieldChangeStoreLike {
  return createStore('audit.FieldChange', deps) as unknown as FieldChangeStoreLike;
}

export function getFollowerStore(deps?: ChatterStoreDeps): FollowerStoreLike {
  return createStore('message.Follower', deps) as unknown as FollowerStoreLike;
}

export function getNotificationStore(deps?: ChatterStoreDeps): NotificationStoreLike {
  return createStore('message.Notification', deps) as unknown as NotificationStoreLike;
}

/** Optional override for `getMessageStore` (unit harness). */
export const GetMessageStoreKey: InjectionKey<typeof getMessageStore> = Symbol('GetMessageStore');

/** Optional override for `getFollowerStore` (unit harness). */
export const GetFollowerStoreKey: InjectionKey<typeof getFollowerStore> = Symbol('GetFollowerStore');

/** Resolve injected message store factory, else the product default. */
export function useInjectedMessageStore(deps?: ChatterStoreDeps): MessageStoreLike {
  const override = inject(GetMessageStoreKey, null);
  return (override ?? getMessageStore)(deps);
}

/** Resolve injected follower store factory, else the product default. */
export function useInjectedFollowerStore(deps?: ChatterStoreDeps): FollowerStoreLike {
  const override = inject(GetFollowerStoreKey, null);
  return (override ?? getFollowerStore)(deps);
}

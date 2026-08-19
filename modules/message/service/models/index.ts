// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

export { default as Message } from './message';
export {
  assertMessageType,
  MESSAGE_TYPES,
  MESSAGE_ATTACHMENT_FIELD,
  TOPIC_MESSAGE_THREAD_CHANGED,
  TOPIC_MESSAGE_NOTIFICATION_USER,
  MESSAGE_POST_TIP_SOURCE,
  __setMessagePublishTipForTest,
  type MessageTypeLiteral,
  type PostMessageReq,
} from './message';
export { default as Follower, type FollowRecordReq, type UnfollowRecordReq } from './follower';
export { default as Notification, type SearchInboxOptions } from './notification';
export { default as MessageSubtype, MESSAGE_SUBTYPE_DISCUSSIONS } from './message_subtype';

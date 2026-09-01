// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Opt-in abstract bases for business / platform apps.
 *
 * Business apps may value-import these under the hard rule that non-core
 * cross-Application value imports are forbidden.
 *
 * - Dial facades (`MessageThreadModel`, `AttachmentOwnerMixin`): runtime dials
 *   platform services; they must not import message/document model classes.
 * - Templates (`PolymorphicRecordModel`): shared SearchByRecord skeleton with
 *   protected hooks; no dial.
 */
export { default as AttachmentOwnerMixin } from './attachment_owner_model';
export { default as MessageThreadModel } from './message_thread_model';
export { default as PolymorphicRecordModel } from './polymorphic_record_model';
export type {
  AttachmentOwnerBindReq,
  AttachmentOwnerBindResp,
  AttachmentOwnerDownloadDisposition,
  AttachmentOwnerUnbindReason,
  AttachmentOwnerUnbindReq,
  AttachmentOwnerUnbindResp,
} from './attachment_owner_contracts';
export type {
  MessageThreadFollowReq,
  MessageThreadPostReq,
  MessageThreadUnfollowReq,
} from './message_thread_contracts';

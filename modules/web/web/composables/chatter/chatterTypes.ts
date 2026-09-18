// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type FieldChange from '@/audit/service/models/field_change';
import type Message from '@/message/service/models/message';
import type Notification from '@/message/service/models/notification';

export type ChatterMessageEntry = {
  kind: 'message';
  id: string;
  at: number;
  type: string;
  body: string;
  authorUid: string | null;
};

export type ChatterFieldChangeEntry = {
  kind: 'fieldChange';
  id: string;
  at: number;
  field: string | null;
  changeKind: string;
  oldValue: string | null;
  newValue: string | null;
  actorUid: string | null;
};

export type ChatterTimelineEntry = ChatterMessageEntry | ChatterFieldChangeEntry;

/**
 * Message / FieldChange / Notification Search projections for chatter FE.
 * Timestamp fields stay wire-loose (ISO string / epoch / Date).
 */
export type ChatterMessageRow = Partial<Pick<Message, 'Type' | 'AuthorUid'>> & {
  Id?: string | null;
  Body?: string | null;
  CreatedAt?: Date | string | number | null;
};

export type ChatterFieldChangeRow = Partial<Pick<FieldChange, 'Field' | 'Kind' | 'OldValue' | 'NewValue' | 'ActorUid'>> & {
  Id?: string | null;
  At?: Date | string | number | null;
};

export type InboxNotificationRow = Partial<Pick<Notification, 'MessageId' | 'Model' | 'ResId' | 'AuthorUid' | 'IsRead'>> & {
  Id?: string | null;
  CreatedAt?: Date | string | number | null;
};

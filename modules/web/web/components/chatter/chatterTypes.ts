// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/** Message entry on the Chatter timeline. */
export type ChatterMessageEntry = {
  kind: 'message';
  id: string;
  at: number;
  type: string;
  body: string;
  authorUid: string | null;
};

/** Field-change / audit entry on the Chatter timeline. */
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

/** Wire-loose message row for mergeChatterTimeline. */
export type ChatterMessageRow = {
  Id?: string | null;
  Type?: string | null;
  Body?: string | null;
  AuthorUid?: string | null;
  CreatedAt?: Date | string | number | null;
};

/** Wire-loose field-change row for mergeChatterTimeline. */
export type ChatterFieldChangeRow = {
  Id?: string | null;
  Field?: string | null;
  Kind?: string | null;
  OldValue?: string | null;
  NewValue?: string | null;
  ActorUid?: string | null;
  At?: Date | string | number | null;
};

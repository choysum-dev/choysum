// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Dial-facing DTO shapes for {@link MessageThreadModel}.
 * Owning implementation remains `message.Message` / `message.Follower`.
 */

export type MessageThreadPostReq = {
  Model: string;
  ResId: string;
  Body: string;
  Type?: string | null;
  CompanyId?: string | null;
  AttachmentObjectId?: string | null;
  AttachmentMutationId?: string | null;
};

export type MessageThreadFollowReq = {
  Model: string;
  ResId: string;
  UserId?: string | null;
  SubtypeId?: string | null;
  CompanyId?: string | null;
};

export type MessageThreadUnfollowReq = {
  Model: string;
  ResId: string;
  UserId?: string | null;
};

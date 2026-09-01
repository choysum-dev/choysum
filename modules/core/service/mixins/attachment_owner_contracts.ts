// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Dial-facing DTO shapes for {@link AttachmentOwnerMixin}.
 * Owning implementation remains `document.AttachmentBinding`; these types live in
 * core so business apps can extend the facade without value-importing document.
 */

export type AttachmentOwnerDownloadDisposition = 'inline' | 'attachment';

export type AttachmentOwnerUnbindReason = 'replace' | 'clear' | 'owner_deleted' | 'cleanup' | 'other';

export type AttachmentOwnerBindReq = {
  attachmentObjectId: string;
  ownerModel: string;
  ownerRecordId: string;
  fieldName: string;
  displayFileName?: string;
  downloadDisposition?: AttachmentOwnerDownloadDisposition;
  mutationId: string;
};

export type AttachmentOwnerBindResp = {
  attachmentBindingId: string;
  status: 'active';
  descriptor?: unknown;
};

export type AttachmentOwnerUnbindReq = {
  attachmentBindingId: string;
  mutationId: string;
  reason?: AttachmentOwnerUnbindReason;
};

export type AttachmentOwnerUnbindResp = {
  attachmentBindingId: string;
  status: 'unbound';
  gcEligibleAfter?: string;
};

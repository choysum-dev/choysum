// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Stable identifier for attachment content records.
 */
export type AttachmentContentId = string;

/**
 * Stable identifier for attachment binding records.
 */
export type AttachmentBindingId = string;

/**
 * Stable identifier for attachment upload sessions.
 */
export type AttachmentUploadSessionId = string;

/**
 * Stable identifier for bind and unbind mutations.
 */
export type MutationId = string;

/**
 * Storage backend used for persisted payload content.
 */
export type AttachmentBackend = 'db' | 's3';

/**
 * Lifecycle states for attachment content rows.
 */
export type AttachmentContentStatus = 'staging' | 'active' | 'deleted';

/**
 * Lifecycle states for attachment bindings.
 */
export type AttachmentBindingStatus = 'active' | 'unbound';

/**
 * Canonical upload session status values accepted by the document service.
 */
export const ATTACHMENT_UPLOAD_SESSION_STATUS_VALUES = ['prepared', 'uploaded', 'finalized', 'expired'] as const;

/**
 * Upload session status union derived from the canonical value list.
 */
export type AttachmentUploadSessionStatus = (typeof ATTACHMENT_UPLOAD_SESSION_STATUS_VALUES)[number];

/**
 * Backward-compatible alias for upload session status values.
 */
export type UploadSessionStatus = AttachmentUploadSessionStatus;

/**
 * Legal state transitions for upload sessions.
 */
export const ATTACHMENT_UPLOAD_SESSION_TRANSITIONS: Readonly<Record<AttachmentUploadSessionStatus, readonly AttachmentUploadSessionStatus[]>> = {
  prepared: ['uploaded', 'expired'],
  uploaded: ['finalized', 'expired'],
  finalized: [],
  expired: [],
};

/**
 * Idempotency rules applied when commit requests replay or race.
 */
export const COMMIT_UPLOAD_IDEMPOTENCY_RULES = {
  duplicateCommit: 'return_current_state',
  concurrentCommit: 'first_writer_wins',
  expiredSession: 'reject_gone',
  finalizedSession: 'reject_conflict',
  checksumMismatch: 'reject_unprocessable_entity',
  issuerMismatch: 'reject_permission_denied',
  companyMismatch: 'reject_permission_denied',
} as const;

/**
 * Normalized outcome labels for upload commit idempotency handling.
 */
export type CommitUploadIdempotencyRule = (typeof COMMIT_UPLOAD_IDEMPOTENCY_RULES)[keyof typeof COMMIT_UPLOAD_IDEMPOTENCY_RULES];

/**
 * Reports whether an upload session may move from one status to another.
 */
export function isAttachmentUploadSessionTransitionAllowed(from: AttachmentUploadSessionStatus, to: AttachmentUploadSessionStatus): boolean {
  return ATTACHMENT_UPLOAD_SESSION_TRANSITIONS[from].includes(to);
}

/**
 * Mutation kinds recorded in the document mutation ledger.
 */
export type MutationAction = 'bind' | 'unbind';

/**
 * Result states stored in the document mutation ledger.
 */
export type MutationLedgerStatus = 'succeeded' | 'failed';

/**
 * Owner-side intent for an upload flow.
 */
export type UploadOperation = 'create' | 'update';

/**
 * Download presentation modes exposed to clients.
 */
export type DownloadDisposition = 'inline' | 'attachment';

/**
 * Reasons recorded when a binding is unbound.
 */
export type UnbindReason = 'replace' | 'clear' | 'owner_deleted' | 'cleanup' | 'other';

/**
 * Permission-check stages that can surface document access denials.
 */
export const DOCUMENT_PERMISSION_DENIED_STAGE_VALUES = [
  'prepare',
  'finalize',
  'bind',
  'unbind',
  'descriptor',
  'binding_cleanup',
  'download',
  'authorize_upload_put',
  'commit_upload_put',
  'resolve_download_content',
] as const;

/**
 * Normalized permission-check stage labels.
 */
export type DocumentPermissionDeniedStage = (typeof DOCUMENT_PERMISSION_DENIED_STAGE_VALUES)[number];

/**
 * Machine-readable reasons attached to document permission denials.
 */
export const DOCUMENT_PERMISSION_DENIED_REASON_VALUES = [
  'binding_company_mismatch',
  'attachment_company_mismatch',
  'owner_record_rule_false',
  'owner_field_read_deny',
  'owner_field_write_deny',
  'owner_record_rule_scope_miss',
  'owner_record_rule_fetch_failed',
  'owner_field_rule_fetch_failed',
  'owner_read_input_incomplete',
  'owner_write_input_incomplete',
  'owner_read_authorization_denied',
  'owner_write_authorization_denied',
  'auth_service_unavailable',
  'invalid_record_rule_envelope',
  'field_missing',
  'unknown',
] as const;

/**
 * Normalized permission-denied reason labels.
 */
export type DocumentPermissionDeniedReason = (typeof DOCUMENT_PERMISSION_DENIED_REASON_VALUES)[number];

/**
 * Caller identity and company scope used by document RPC entry points.
 */
export interface PrincipalContext {
  userId: string;
  activeCompanyId: string;
  enabledCompanyIds?: string[];
}

/**
 * Reference to a payload already persisted in backing storage.
 */
export type UploadedPayloadRef = {
  kind: 'stored_content';
  storedContentId: string;
};

/**
 * Image metadata persisted for image-like attachment content.
 */
export interface ImageObjectMetadata {
  width: number;
  height: number;
  format: string;
}

/**
 * Client-facing upload target returned by prepare flows.
 */
export interface UploadTarget {
  method: 'PUT';
  url: string;
  headers?: Record<string, string>;
  expiresAt?: string;
  maxUploadBytes?: number;
  requiredChecksumAlgorithm?: 'sha256';
  allowedMimeTypes?: string[];
}

/**
 * Request payload for creating or replaying an upload session.
 */
export interface PrepareUploadReq {
  ownerModel: string;
  fieldName: string;
  operation: UploadOperation;
  ownerRecordId?: string;
  businessRequestId: string;
  proposedFileName?: string;
  proposedContentType?: string;
  proposedSizeBytes?: number;
  checksumSha256?: string;
  originalFileName?: string;
  clientContentType?: string;
}

/**
 * Response returned after an upload session is prepared.
 */
export interface PrepareUploadResp {
  uploadId: AttachmentUploadSessionId;
  uploadTarget: UploadTarget;
}

/**
 * Request payload for finalizing a prepared upload session.
 */
export interface FinalizeUploadReq {
  uploadId: AttachmentUploadSessionId;
  businessRequestId: string;
}

/**
 * Finalized attachment content metadata returned to callers.
 */
export interface FinalizeUploadResp {
  attachmentObjectId: AttachmentContentId;
  status: 'active';
  mimeType: string;
  sizeBytes: number;
  checksumSha256: string;
  imageMetadata?: ImageObjectMetadata;
}

/**
 * Authorization request for issuing a direct upload PUT ticket.
 */
export interface AuthorizeUploadPutReq {
  uploadId: AttachmentUploadSessionId;
  principal: PrincipalContext;
  requestMeta?: {
    contentType?: string;
    contentLength?: number;
    checksumSha256?: string;
  };
}

/**
 * Ticket material and constraints for a direct upload PUT.
 */
export interface AuthorizeUploadPutResp {
  uploadId: AttachmentUploadSessionId;
  maxUploadBytes: number;
  requiredChecksumAlgorithm: 'sha256';
  expectedChecksumSha256?: string;
  allowedMimeTypes?: string[];
  payloadWriteTicket: string;
}

/**
 * Commit request for a previously uploaded payload.
 */
export interface CommitUploadPutReq {
  uploadId: AttachmentUploadSessionId;
  principal: PrincipalContext;
  payloadReceipt: {
    payloadId: string;
    sizeBytes: number;
    checksumSha256: string;
    contentType?: string;
  };
}

/**
 * Upload session states observable after commit processing.
 */
export type CommitUploadSessionStatus = 'uploaded' | 'finalized';

/**
 * Commit result describing the persisted payload state.
 */
export interface CommitUploadPutResp {
  uploadId: AttachmentUploadSessionId;
  attachmentUploadSessionStatus: CommitUploadSessionStatus;
  attachmentContentId?: AttachmentContentId;
}

/**
 * Request payload for resolving a download ticket from a binding.
 */
export interface ResolveDownloadContentReq {
  attachmentBindingId: AttachmentBindingId;
  principal: PrincipalContext;
}

/**
 * Download ticket and content metadata resolved from a binding.
 */
export interface ResolveDownloadContentResp {
  attachmentBindingId: AttachmentBindingId;
  payloadReadTicket: string;
  mimeType: string;
  sizeBytes: number;
  checksumSha256?: string;
  fileName: string;
  downloadDisposition: DownloadDisposition;
  etag?: string;
}

/**
 * Attachment metadata exposed to owner-facing document UIs.
 */
export interface AttachmentDescriptor {
  id: AttachmentBindingId;
  fileName: string;
  mimeType: string;
  sizeBytes: number;
  checksumSha256: string;
  downloadUrl?: string;
  downloadUrlExpiresAt?: string;
}

/**
 * Request payload for binding content to an owner field.
 */
export interface BindReq {
  attachmentObjectId: AttachmentContentId;
  ownerModel: string;
  ownerRecordId: string;
  fieldName: string;
  displayFileName?: string;
  downloadDisposition?: DownloadDisposition;
  mutationId: MutationId;
}

/**
 * Result returned after content is bound to an owner field.
 */
export interface BindResp {
  attachmentBindingId: AttachmentBindingId;
  status: 'active';
  descriptor: AttachmentDescriptor;
}

/**
 * Request payload for unbinding content from an owner field.
 */
export interface UnbindReq {
  attachmentBindingId: AttachmentBindingId;
  mutationId: MutationId;
  reason?: UnbindReason;
}

/**
 * Result returned after a binding moves to the unbound state.
 */
export interface UnbindResp {
  attachmentBindingId: AttachmentBindingId;
  status: 'unbound';
  gcEligibleAfter?: string;
}

/**
 * Batch request for resolving multiple attachment descriptors.
 */
export interface BatchDescribeReq {
  attachmentBindingIds: AttachmentBindingId[];
}

/**
 * Descriptor entry returned for a single attachment binding.
 */
export interface BatchDescribeItem {
  attachmentBindingId: AttachmentBindingId;
  descriptor: AttachmentDescriptor;
  displayName?: string;
  previewUrl?: string;
  ownerModel?: string;
  ownerRecordId?: string;
  fieldName?: string;
}

/**
 * Batch descriptor response for owner-facing attachment listings.
 */
export interface BatchDescribeResp {
  items: BatchDescribeItem[];
}

// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package gateway

import (
	"net/http"
	"strings"
)

type documentPermissionDeniedStage string

const (
	permissionDeniedStageUnknown            documentPermissionDeniedStage = "unknown"
	permissionDeniedStagePrepare            documentPermissionDeniedStage = "prepare"
	permissionDeniedStageFinalize           documentPermissionDeniedStage = "finalize"
	permissionDeniedStageBind               documentPermissionDeniedStage = "bind"
	permissionDeniedStageUnbind             documentPermissionDeniedStage = "unbind"
	permissionDeniedStageDescriptor         documentPermissionDeniedStage = "descriptor"
	permissionDeniedStageBindingCleanup     documentPermissionDeniedStage = "binding_cleanup"
	permissionDeniedStageDownload           documentPermissionDeniedStage = "download"
	permissionDeniedStageAuthorizeUploadPut documentPermissionDeniedStage = "authorize_upload_put"
	permissionDeniedStageCommitUploadPut    documentPermissionDeniedStage = "commit_upload_put"
	permissionDeniedStageResolveDownload    documentPermissionDeniedStage = "resolve_download_content"
)

var permissionDeniedStageSet = map[string]struct{}{
	string(permissionDeniedStageUnknown):            {},
	string(permissionDeniedStagePrepare):            {},
	string(permissionDeniedStageFinalize):           {},
	string(permissionDeniedStageBind):               {},
	string(permissionDeniedStageUnbind):             {},
	string(permissionDeniedStageDescriptor):         {},
	string(permissionDeniedStageBindingCleanup):     {},
	string(permissionDeniedStageDownload):           {},
	string(permissionDeniedStageAuthorizeUploadPut): {},
	string(permissionDeniedStageCommitUploadPut):    {},
	string(permissionDeniedStageResolveDownload):    {},
}

type documentPermissionDeniedReason string

const (
	permissionDeniedReasonUnknown                   documentPermissionDeniedReason = "unknown"
	permissionDeniedReasonBindingCompanyMismatch    documentPermissionDeniedReason = "binding_company_mismatch"
	permissionDeniedReasonAttachmentCompanyMismatch documentPermissionDeniedReason = "attachment_company_mismatch"
	permissionDeniedReasonOwnerRecordRuleFalse      documentPermissionDeniedReason = "owner_record_rule_false"
	permissionDeniedReasonOwnerFieldReadDeny        documentPermissionDeniedReason = "owner_field_read_deny"
	permissionDeniedReasonOwnerFieldWriteDeny       documentPermissionDeniedReason = "owner_field_write_deny"
	permissionDeniedReasonOwnerRuleScopeMiss        documentPermissionDeniedReason = "owner_record_rule_scope_miss"
	permissionDeniedReasonOwnerRuleFetchFailed      documentPermissionDeniedReason = "owner_record_rule_fetch_failed"
	permissionDeniedReasonOwnerFieldFetchFailed     documentPermissionDeniedReason = "owner_field_rule_fetch_failed"
	permissionDeniedReasonOwnerReadInputIncomplete  documentPermissionDeniedReason = "owner_read_input_incomplete"
	permissionDeniedReasonOwnerWriteInputIncomplete documentPermissionDeniedReason = "owner_write_input_incomplete"
	permissionDeniedReasonOwnerReadDenied           documentPermissionDeniedReason = "owner_read_authorization_denied"
	permissionDeniedReasonOwnerWriteDenied          documentPermissionDeniedReason = "owner_write_authorization_denied"
	permissionDeniedReasonAuthServiceUnavailable    documentPermissionDeniedReason = "auth_service_unavailable"
	permissionDeniedReasonInvalidRuleEnvelope       documentPermissionDeniedReason = "invalid_record_rule_envelope"
	permissionDeniedReasonFieldMissing              documentPermissionDeniedReason = "field_missing"
)

var permissionDeniedReasonSet = map[string]struct{}{
	string(permissionDeniedReasonUnknown):                   {},
	string(permissionDeniedReasonBindingCompanyMismatch):    {},
	string(permissionDeniedReasonAttachmentCompanyMismatch): {},
	string(permissionDeniedReasonOwnerRecordRuleFalse):      {},
	string(permissionDeniedReasonOwnerFieldReadDeny):        {},
	string(permissionDeniedReasonOwnerFieldWriteDeny):       {},
	string(permissionDeniedReasonOwnerRuleScopeMiss):        {},
	string(permissionDeniedReasonOwnerRuleFetchFailed):      {},
	string(permissionDeniedReasonOwnerFieldFetchFailed):     {},
	string(permissionDeniedReasonOwnerReadInputIncomplete):  {},
	string(permissionDeniedReasonOwnerWriteInputIncomplete): {},
	string(permissionDeniedReasonOwnerReadDenied):           {},
	string(permissionDeniedReasonOwnerWriteDenied):          {},
	string(permissionDeniedReasonAuthServiceUnavailable):    {},
	string(permissionDeniedReasonInvalidRuleEnvelope):       {},
	string(permissionDeniedReasonFieldMissing):              {},
}

func normalizePermissionDeniedStage(value string) string {
	normalized := normalizeObservationLabel(value, string(permissionDeniedStageUnknown))
	if _, ok := permissionDeniedStageSet[normalized]; ok {
		return normalized
	}
	return string(permissionDeniedStageUnknown)
}

func normalizePermissionDeniedReason(value string) string {
	normalized := normalizeObservationLabel(value, string(permissionDeniedReasonUnknown))
	if _, ok := permissionDeniedReasonSet[normalized]; ok {
		return normalized
	}
	return string(permissionDeniedReasonUnknown)
}

type documentErrorCode string

const (
	documentErrorUnknown                documentErrorCode = "document.UNKNOWN"
	documentErrorInternal               documentErrorCode = "document.INTERNAL"
	documentErrorInvalidArgument        documentErrorCode = "document.INVALID_ARGUMENT"
	documentErrorUnauthenticated        documentErrorCode = "document.UNAUTHENTICATED"
	documentErrorPermissionDenied       documentErrorCode = "document.PERMISSION_DENIED"
	documentErrorNotFound               documentErrorCode = "document.NOT_FOUND"
	documentErrorFailedPrecondition     documentErrorCode = "document.FAILED_PRECONDITION"
	documentErrorIdempotencyReused      documentErrorCode = "document.IDEMPOTENCY_KEY_REUSED"
	documentErrorUploadNotFound         documentErrorCode = "document.UPLOAD_SESSION_NOT_FOUND"
	documentErrorUploadExpired          documentErrorCode = "document.UPLOAD_SESSION_EXPIRED"
	documentErrorUploadFinalized        documentErrorCode = "document.UPLOAD_SESSION_FINALIZED"
	documentErrorInvalidUploadBody      documentErrorCode = "document.INVALID_UPLOAD_BODY"
	documentErrorUploadTooLarge         documentErrorCode = "document.UPLOAD_TOO_LARGE"
	documentErrorMimeTypeNotAllowed     documentErrorCode = "document.MIME_TYPE_NOT_ALLOWED"
	documentErrorChecksumMismatch       documentErrorCode = "document.CHECKSUM_MISMATCH"
	documentErrorBindingNotFound        documentErrorCode = "document.BINDING_NOT_FOUND"
	documentErrorAttachmentNotFound     documentErrorCode = "document.ATTACHMENT_NOT_FOUND"
	documentErrorSkeletonNotImplemented documentErrorCode = "document.SKELETON_NOT_IMPLEMENTED"
)

var documentErrorHTTPStatus = map[documentErrorCode]int{
	documentErrorUnknown:                http.StatusInternalServerError,
	documentErrorInternal:               http.StatusInternalServerError,
	documentErrorInvalidArgument:        http.StatusBadRequest,
	documentErrorUnauthenticated:        http.StatusUnauthorized,
	documentErrorPermissionDenied:       http.StatusForbidden,
	documentErrorNotFound:               http.StatusNotFound,
	documentErrorFailedPrecondition:     http.StatusPreconditionFailed,
	documentErrorIdempotencyReused:      http.StatusConflict,
	documentErrorUploadNotFound:         http.StatusNotFound,
	documentErrorUploadExpired:          http.StatusGone,
	documentErrorUploadFinalized:        http.StatusConflict,
	documentErrorInvalidUploadBody:      http.StatusBadRequest,
	documentErrorUploadTooLarge:         http.StatusRequestEntityTooLarge,
	documentErrorMimeTypeNotAllowed:     http.StatusUnsupportedMediaType,
	documentErrorChecksumMismatch:       http.StatusUnprocessableEntity,
	documentErrorBindingNotFound:        http.StatusNotFound,
	documentErrorAttachmentNotFound:     http.StatusNotFound,
	documentErrorSkeletonNotImplemented: http.StatusNotImplemented,
}

func documentErrorStatus(code string) int {
	normalized := documentErrorCode(strings.TrimSpace(code))
	if status, ok := documentErrorHTTPStatus[normalized]; ok {
		return status
	}
	return http.StatusInternalServerError
}

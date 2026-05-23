// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package gateway

import (
	"net/http"
	"testing"
)

func TestDocumentErrorStatusMappingContract(t *testing.T) {
	cases := []struct {
		code documentErrorCode
		want int
	}{
		{code: documentErrorUnauthenticated, want: http.StatusUnauthorized},
		{code: documentErrorPermissionDenied, want: http.StatusForbidden},
		{code: documentErrorUploadExpired, want: http.StatusGone},
		{code: documentErrorUploadTooLarge, want: http.StatusRequestEntityTooLarge},
		{code: documentErrorMimeTypeNotAllowed, want: http.StatusUnsupportedMediaType},
		{code: documentErrorChecksumMismatch, want: http.StatusUnprocessableEntity},
		{code: documentErrorSkeletonNotImplemented, want: http.StatusNotImplemented},
	}

	for _, tc := range cases {
		if got := documentErrorStatus(string(tc.code)); got != tc.want {
			t.Fatalf("documentErrorStatus(%q) = %d, want %d", tc.code, got, tc.want)
		}
	}

	if got := documentErrorStatus("document.SOMETHING_UNKNOWN"); got != http.StatusInternalServerError {
		t.Fatalf("documentErrorStatus unknown = %d, want %d", got, http.StatusInternalServerError)
	}
}

func TestPermissionDeniedStageContractNormalization(t *testing.T) {
	if got := normalizePermissionDeniedStage(" download "); got != "download" {
		t.Fatalf("stage normalize download = %q, want download", got)
	}
	if got := normalizePermissionDeniedStage("commit-upload-put"); got != "commit_upload_put" {
		t.Fatalf("stage normalize commit-upload-put = %q, want commit_upload_put", got)
	}
	if got := normalizePermissionDeniedStage("totally_unknown"); got != "unknown" {
		t.Fatalf("stage normalize unknown = %q, want unknown", got)
	}
}

func TestPermissionDeniedReasonContractNormalization(t *testing.T) {
	if got := normalizePermissionDeniedReason("owner_record_rule_false"); got != "owner_record_rule_false" {
		t.Fatalf("reason normalize owner_record_rule_false = %q, want owner_record_rule_false", got)
	}
	if got := normalizePermissionDeniedReason("owner-field-read-deny"); got != "owner_field_read_deny" {
		t.Fatalf("reason normalize owner-field-read-deny = %q, want owner_field_read_deny", got)
	}
	if got := normalizePermissionDeniedReason("unexpected_reason"); got != "unknown" {
		t.Fatalf("reason normalize unknown = %q, want unknown", got)
	}
}

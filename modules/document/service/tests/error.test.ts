// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { DOCUMENT_ERROR_HTTP_STATUS, DocumentErrCode, documentErrorHttpStatus } from '../error';

test('document error contract: http status mapping is frozen for key codes', () => {
  expect(DOCUMENT_ERROR_HTTP_STATUS[DocumentErrCode.UNAUTHENTICATED]).toBe(401);
  expect(DOCUMENT_ERROR_HTTP_STATUS[DocumentErrCode.PERMISSION_DENIED]).toBe(403);
  expect(DOCUMENT_ERROR_HTTP_STATUS[DocumentErrCode.NOT_FOUND]).toBe(404);
  expect(DOCUMENT_ERROR_HTTP_STATUS[DocumentErrCode.UPLOAD_SESSION_EXPIRED]).toBe(410);
  expect(DOCUMENT_ERROR_HTTP_STATUS[DocumentErrCode.UPLOAD_TOO_LARGE]).toBe(413);
  expect(DOCUMENT_ERROR_HTTP_STATUS[DocumentErrCode.MIME_TYPE_NOT_ALLOWED]).toBe(415);
  expect(DOCUMENT_ERROR_HTTP_STATUS[DocumentErrCode.CHECKSUM_MISMATCH]).toBe(422);
  expect(DOCUMENT_ERROR_HTTP_STATUS[DocumentErrCode.SKELETON_NOT_IMPLEMENTED]).toBe(501);
});

test('document error contract: lookup helper falls back to internal error for unknown code', () => {
  expect(documentErrorHttpStatus(DocumentErrCode.INVALID_ARGUMENT)).toBe(400);
  expect(documentErrorHttpStatus(DocumentErrCode.UPLOAD_SESSION_FINALIZED)).toBe(409);
  expect(documentErrorHttpStatus('UNKNOWN_CUSTOM_CODE')).toBe(500);
  expect(documentErrorHttpStatus('')).toBe(500);
});

// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ChoysumError } from '@/core/service/error';
import { DocumentErrCode, documentErrorHttpStatus, throwDocumentError, DOCUMENT_ERROR_HTTP_STATUS } from '../error';

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

test('document error contract: throwDocumentError stringifies metadata values so withMetadata receives only strings', () => {
  let caught: ChoysumError | undefined;
  try {
    throwDocumentError(DocumentErrCode.INVALID_ARGUMENT, 'test message', 3, {
      field: 'attachmentBindingIds',
      count: 42,
      flag: true,
      nil: null,
      undef: undefined,
    });
  } catch (err) {
    caught = err as ChoysumError;
  }
  expect(caught).toBeDefined();
  expect(caught!.domain).toBe('document');
  expect(caught!.code).toBe('INVALID_ARGUMENT');
  expect(caught!.message).toBe('test message');
  expect(caught!.grpcCode).toBe(3);
  expect(caught!.metadata).toEqual({
    field: 'attachmentBindingIds',
    count: '42',
    flag: 'true',
    nil: '',
    undef: '',
  });
});

test('document error contract: throwDocumentError passes through when metadata is omitted', () => {
  let caught: ChoysumError | undefined;
  try {
    throwDocumentError(DocumentErrCode.NOT_FOUND, 'missing', 5);
  } catch (err) {
    caught = err as ChoysumError;
  }
  expect(caught).toBeDefined();
  expect(caught!.code).toBe('NOT_FOUND');
  expect(caught!.grpcCode).toBe(5);
  expect(caught!.metadata == null || Object.keys(caught!.metadata!).length === 0).toBe(true);
});

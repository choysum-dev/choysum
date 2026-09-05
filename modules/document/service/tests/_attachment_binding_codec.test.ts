// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ChoysumError } from '@/core/service/error';
import { DocumentErrCode } from '../error';
import { assertDownloadDisposition } from '../models/_attachment_binding_codec';

test('document.binding_codec: assertDownloadDisposition defaults blank and accepts whitelist', () => {
  expect(assertDownloadDisposition(undefined)).toBe('attachment');
  expect(assertDownloadDisposition('')).toBe('attachment');
  expect(assertDownloadDisposition('Inline')).toBe('inline');
  expect(assertDownloadDisposition('ATTACHMENT')).toBe('attachment');
});

test('document.binding_codec: assertDownloadDisposition maps invalid values to domain INVALID_ARGUMENT', () => {
  let caught: ChoysumError | undefined;
  try {
    assertDownloadDisposition('stream');
  } catch (err) {
    caught = err as ChoysumError;
  }
  expect(caught).toBeDefined();
  expect(caught!.domain).toBe('document');
  expect(caught!.code).toBe(DocumentErrCode.INVALID_ARGUMENT);
  expect(String(caught!.message)).toMatch(/inline or attachment/i);
  expect(caught!.metadata).toMatchObject({ downloadDisposition: 'stream' });
});

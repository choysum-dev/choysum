// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { documentOwnerAuthErrorMessageForTest } from '../models/_owner_authorization';

test('document owner auth: errorMessage normalizes Error and non-Error details', () => {
  expect(documentOwnerAuthErrorMessageForTest(new Error('  boom  '))).toBe('boom');
  expect(documentOwnerAuthErrorMessageForTest(42)).toBe('42');
  expect(documentOwnerAuthErrorMessageForTest('')).toBe('unknown_error');
  expect(documentOwnerAuthErrorMessageForTest(undefined)).toBe('unknown_error');
});

// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { parseUploadedPayloadRefFromUnknown } from '../models/_attachment_upload_codec';

test('document.upload_codec: parseUploadedPayloadRefFromUnknown returns undefined for empty input', () => {
  expect(parseUploadedPayloadRefFromUnknown(undefined)).toBeUndefined();
  expect(parseUploadedPayloadRefFromUnknown(null)).toBeUndefined();
  expect(parseUploadedPayloadRefFromUnknown('')).toBeUndefined();
  expect(parseUploadedPayloadRefFromUnknown('   ')).toBeUndefined();
});

test('document.upload_codec: parseUploadedPayloadRefFromUnknown parses JSON string recursively', () => {
  const ref = parseUploadedPayloadRefFromUnknown(
    JSON.stringify({ kind: 'stored_content', storedContentId: 'sc_json' })
  );
  expect(ref).toEqual({ kind: 'stored_content', storedContentId: 'sc_json' });
});

test('document.upload_codec: parseUploadedPayloadRefFromUnknown accepts opaque payload id string', () => {
  const ref = parseUploadedPayloadRefFromUnknown('sc:stored_1');
  expect(ref).toEqual({ kind: 'stored_content', storedContentId: 'stored_1' });
});

test('document.upload_codec: parseUploadedPayloadRefFromUnknown reads payloadId object form', () => {
  const ref = parseUploadedPayloadRefFromUnknown({ payloadId: 'sc:from_obj' });
  expect(ref).toEqual({ kind: 'stored_content', storedContentId: 'from_obj' });
});

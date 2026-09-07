// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { normalizeUiResourceRequires } from './role_ui_requires_explain';

test('Role UiResources field binding: normalizeUiResourceRequires accepts arrays and JSON', () => {
  expect(normalizeUiResourceRequires(['rpc:/auth.User/Browse', 'rpc:/auth.User/Browse'])).toEqual(['rpc:/auth.User/Browse']);
  expect(normalizeUiResourceRequires('["rpc:/auth.User/Browse"]')).toEqual(['rpc:/auth.User/Browse']);
  expect(normalizeUiResourceRequires(null)).toEqual([]);
});

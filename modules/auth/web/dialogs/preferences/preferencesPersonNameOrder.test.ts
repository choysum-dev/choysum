// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { preferencesPersonNameFieldOrder } from './preferencesPersonNameOrder';

test('preferencesPersonNameFieldOrder: last-first vs first-last', () => {
  expect(preferencesPersonNameFieldOrder('{LastName}{FirstName}')).toEqual(['LastName', 'FirstName']);
  expect(preferencesPersonNameFieldOrder('{FirstName} {LastName}')).toEqual(['FirstName', 'LastName']);
  expect(preferencesPersonNameFieldOrder(undefined)).toEqual(['FirstName', 'LastName']);
});

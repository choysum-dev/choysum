// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { parseModelFullName, parseServiceFullName } from '@/core/service/utils/model_parsing';

test('parseModelFullName returns app and model for valid input', () => {
  expect(parseModelFullName('auth.User')).toEqual({ appName: 'auth', modelName: 'User' });
  expect(parseModelFullName('  auth.User  ')).toEqual({ appName: 'auth', modelName: 'User' });
  expect(parseModelFullName('meta.runtime.ModelA')).toEqual({ appName: 'meta.runtime', modelName: 'ModelA' });
});

test('parseModelFullName returns null for invalid input', () => {
  expect(parseModelFullName('')).toBe(null);
  expect(parseModelFullName('   ')).toBe(null);
  expect(parseModelFullName('auth')).toBe(null);
  expect(parseModelFullName('.User')).toBe(null);
  expect(parseModelFullName('auth.')).toBe(null);
  expect(parseModelFullName('.')).toBe(null);
});

test('parseServiceFullName parses grpc full method form', () => {
  expect(parseServiceFullName('/auth.User/Search')).toEqual({ appName: 'auth', modelName: 'User', methodName: 'Search' });
  expect(parseServiceFullName('auth.User/Search')).toEqual({ appName: 'auth', modelName: 'User', methodName: 'Search' });
  expect(parseServiceFullName('  /meta.runtime.ModelA/Run  ')).toEqual({ appName: 'meta.runtime', modelName: 'ModelA', methodName: 'Run' });
});

test('parseServiceFullName returns null for invalid input', () => {
  expect(parseServiceFullName('')).toBe(null);
  expect(parseServiceFullName('/')).toBe(null);
  expect(parseServiceFullName('/auth.User')).toBe(null);
  expect(parseServiceFullName('/auth.User/')).toBe(null);
  expect(parseServiceFullName('/auth/User/Search')).toBe(null);
  expect(parseServiceFullName('/auth./Search')).toBe(null);
});

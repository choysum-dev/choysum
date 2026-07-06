// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ensureModuleName } from '@/meta/service/models/_module_op_request';

test('ensureModuleName returns trimmed name', () => {
  expect(ensureModuleName('  auth  ')).toBe('auth');
  expect(ensureModuleName('base')).toBe('base');
});

test('ensureModuleName throws on empty or whitespace', () => {
  expect(() => ensureModuleName('')).toThrow('moduleName cannot be empty');
  expect(() => ensureModuleName('   ')).toThrow('moduleName cannot be empty');
  expect(() => ensureModuleName(undefined)).toThrow('moduleName cannot be empty');
  expect(() => ensureModuleName(null as any)).toThrow('moduleName cannot be empty');
});

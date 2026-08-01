// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import MetaModule from '@/meta/service/models/module';

test('ensureModuleName returns trimmed name', () => {
  expect((MetaModule as any).ensureModuleName('  auth  ')).toBe('auth');
  expect((MetaModule as any).ensureModuleName('base')).toBe('base');
});

test('ensureModuleName throws on empty or whitespace', () => {
  expect(() => (MetaModule as any).ensureModuleName('')).toThrow('moduleName cannot be empty');
  expect(() => (MetaModule as any).ensureModuleName('   ')).toThrow('moduleName cannot be empty');
  expect(() => (MetaModule as any).ensureModuleName(undefined)).toThrow('moduleName cannot be empty');
  expect(() => (MetaModule as any).ensureModuleName(null as any)).toThrow('moduleName cannot be empty');
});

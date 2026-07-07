// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import IrModule from '@/meta/service/models/ir_module';

test('ensureModuleName returns trimmed name', () => {
  expect((IrModule as any).ensureModuleName('  auth  ')).toBe('auth');
  expect((IrModule as any).ensureModuleName('base')).toBe('base');
});

test('ensureModuleName throws on empty or whitespace', () => {
  expect(() => (IrModule as any).ensureModuleName('')).toThrow('moduleName cannot be empty');
  expect(() => (IrModule as any).ensureModuleName('   ')).toThrow('moduleName cannot be empty');
  expect(() => (IrModule as any).ensureModuleName(undefined)).toThrow('moduleName cannot be empty');
  expect(() => (IrModule as any).ensureModuleName(null as any)).toThrow('moduleName cannot be empty');
});

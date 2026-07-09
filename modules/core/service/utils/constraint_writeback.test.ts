// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { writeConstraintFields } from '@/core/service/utils/constraint_writeback';

test('writeConstraintFields writes selected fields from self to ctx.values on create with forceOnCreate', () => {
  const self: Record<string, unknown> = { Code: 'ABC', Name: 'Test' };
  const ctx: { values?: Record<string, unknown>; mode?: string } = { values: {}, mode: 'create' };

  writeConstraintFields(self, ctx, ['Code'], { forceOnCreate: true });

  expect(ctx.values?.Code).toBe('ABC');
});

test('writeConstraintFields does not write fields not present on ctx.values when not create', () => {
  const self: Record<string, unknown> = { Code: 'ABC', Name: 'Test' };
  const ctx: { values?: Record<string, unknown>; mode?: string } = { values: {}, mode: 'update' };

  writeConstraintFields(self, ctx, ['Code']);

  expect(ctx.values?.Code).toBeUndefined();
});

test('writeConstraintFields writes back when field already exists on ctx.values', () => {
  const self: Record<string, unknown> = { Code: 'NEW' };
  const ctx: { values?: Record<string, unknown>; mode?: string } = { values: { Code: 'OLD' }, mode: 'update' };

  writeConstraintFields(self, ctx, ['Code']);

  expect(ctx.values?.Code).toBe('NEW');
});

test('writeConstraintFields writes targetField when triggerFields present', () => {
  const self: Record<string, unknown> = { CompanyId: 'C1', CompanyScopeKey: '__GLOBAL__' };
  const ctx: { values?: Record<string, unknown>; mode?: string } = { values: { CompanyId: 'C2' }, mode: 'update' };

  writeConstraintFields(self, ctx, [], {
    triggerFields: ['CompanyId'],
    targetField: 'CompanyScopeKey',
  });

  expect(ctx.values?.CompanyScopeKey).toBe('__GLOBAL__');
});

test('writeConstraintFields writes targetField with forceOnCreate when no trigger fields present', () => {
  const self: Record<string, unknown> = { CompanyScopeKey: 'C1' };
  const ctx: { values?: Record<string, unknown>; mode?: string } = { values: {}, mode: 'create' };

  writeConstraintFields(self, ctx, [], {
    forceOnCreate: true,
    triggerFields: ['CompanyId'],
    targetField: 'CompanyScopeKey',
  });

  expect(ctx.values?.CompanyScopeKey).toBe('C1');
});

test('writeConstraintFields initializes ctx.values when missing', () => {
  const self: Record<string, unknown> = { Code: 'ABC' };
  const ctx: { values?: Record<string, unknown>; mode?: string } = { mode: 'create' };

  writeConstraintFields(self, ctx, ['Code'], { forceOnCreate: true });

  expect(ctx.values?.Code).toBe('ABC');
});

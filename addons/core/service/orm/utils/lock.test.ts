// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { LockUtils } from './lock';

test('buildOptimisticLockCondition returns id condition without updatedAt', () => {
  const condition = LockUtils.buildOptimisticLockCondition('rec_1');
  expect(condition).toEqual(['Id', '=', 'rec_1']);
});

test('buildOptimisticLockCondition returns composite condition with updatedAt', () => {
  const updatedAt = new Date('2026-01-02T03:04:05.000Z');
  const condition = LockUtils.buildOptimisticLockCondition('rec_2', updatedAt) as any;

  expect(condition.And).toHaveLength(2);
  expect(condition.And[0]).toEqual(['Id', '=', 'rec_2']);
  expect(condition.And[1]).toEqual(['UpdatedAt', '=', updatedAt]);
});

test('validateUpdateResult passes when update has rows', () => {
  LockUtils.validateUpdateResult([{ Id: 'ok' }]);
  expect(true).toBe(true);
});

test('validateUpdateResult throws when result is empty or undefined', () => {
  expect(() => LockUtils.validateUpdateResult([])).toThrow('has been modified');
  expect(() => LockUtils.validateUpdateResult(undefined as any)).toThrow('has been modified');
});

test('isOptimisticLockError detects optimistic conflict message', () => {
  const optimistic = new Error('record has been modified by another user');
  const generic = new Error('network timeout');

  expect(LockUtils.isOptimisticLockError(optimistic)).toBe(true);
  expect(LockUtils.isOptimisticLockError(generic)).toBe(false);
});

test('formatLockError returns same optimistic error and wraps generic errors', () => {
  const optimistic = new Error('record has been modified by another user');
  const generic = new Error('disk full');

  const optimisticOut = LockUtils.formatLockError(optimistic);
  const genericOut = LockUtils.formatLockError(generic);

  expect(optimisticOut).toBe(optimistic);
  expect(genericOut).not.toBe(generic);
  expect(genericOut.message).toContain('Update failed: disk full');
});

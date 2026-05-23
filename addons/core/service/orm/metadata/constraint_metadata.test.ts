// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel } from '@/core/service';
import { Constraint, getEffectiveConstraints } from '@/core/service/api/constraint';
import { MetadataStorage } from '@/core/service/api/metadata';

class ConstraintParentModel extends BaseModel {
  Name?: string;
}

Object.defineProperty(ConstraintParentModel.prototype, 'baseConstraint', {
  value: function baseConstraint() {},
  configurable: true,
  writable: true,
});

Constraint<ConstraintParentModel>('Name', { priority: 40 })(ConstraintParentModel.prototype, 'baseConstraint', undefined as any);

class ConstraintChildModel extends ConstraintParentModel {
  Code?: string;
}

Object.defineProperty(ConstraintChildModel.prototype, 'baseConstraint', {
  value: function baseConstraint() {},
  configurable: true,
  writable: true,
});
Object.defineProperty(ConstraintChildModel.prototype, 'childConstraint', {
  value: function childConstraint() {},
  configurable: true,
  writable: true,
});

Constraint<ConstraintChildModel>('Name', { priority: 5 })(ConstraintChildModel.prototype, 'baseConstraint', undefined as any);
Constraint<ConstraintChildModel>('Code', { priority: 20, preview: true })(ConstraintChildModel.prototype, 'childConstraint', undefined as any);

test('constraint metadata merges by method name across inheritance', () => {
  const meta = MetadataStorage.instance.getModelMetadata(ConstraintChildModel as any);
  const handlers = meta.constraintHandlers || [];

  expect(handlers.length).toBe(2);

  const base = handlers.find(handler => handler.method === 'baseConstraint');
  const child = handlers.find(handler => handler.method === 'childConstraint');

  expect(Boolean(base)).toBe(true);
  expect(Boolean(child)).toBe(true);
  expect(base?.priority).toBe(5);
  expect(base?.fields).toEqual(['Name']);
  expect(child?.preview).toBe(true);
  expect(child?.fields).toEqual(['Code']);
});

test('effective constraints are deduplicated by method with correct priority and source', () => {
  const effective = getEffectiveConstraints(ConstraintChildModel as any);

  expect(effective.length).toBe(2);
  expect(effective[0]?.method).toBe('baseConstraint');
  expect(effective[0]?.priority).toBe(5);
  expect(String(effective[0]?.source || '').includes('ConstraintChildModel')).toBe(true);

  expect(effective[1]?.method).toBe('childConstraint');
  expect(effective[1]?.priority).toBe(20);
  expect(String(effective[1]?.source || '').includes('ConstraintChildModel')).toBe(true);
});

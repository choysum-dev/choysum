// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel, Model } from '@/core/service';
import { Constraint } from '@/core/service/api/constraint';
import { MetadataStorage } from '@/core/service/api/metadata';
import IrModel from '@/meta/service/models/ir_model';

class ConstraintDiagParent extends BaseModel {
  Name?: string;
}
Model('ConstraintDiagParent')(ConstraintDiagParent as any);

Object.defineProperty(ConstraintDiagParent.prototype, 'baseConstraint', {
  value: function baseConstraint() {},
  configurable: true,
  writable: true,
});
Constraint<ConstraintDiagParent>('Name', { priority: 40 })(ConstraintDiagParent.prototype, 'baseConstraint', undefined as any);

class ConstraintDiagChild extends ConstraintDiagParent {
  Code?: string;
}
Model('ConstraintDiagChild')(ConstraintDiagChild as any);

Object.defineProperty(ConstraintDiagChild.prototype, 'baseConstraint', {
  value: function baseConstraint() {},
  configurable: true,
  writable: true,
});
Object.defineProperty(ConstraintDiagChild.prototype, 'childConstraint', {
  value: function childConstraint() {},
  configurable: true,
  writable: true,
});
Constraint<ConstraintDiagChild>('Name', { priority: 5 })(ConstraintDiagChild.prototype, 'baseConstraint', undefined as any);
Constraint<ConstraintDiagChild>('Code', { priority: 20, preview: true })(ConstraintDiagChild.prototype, 'childConstraint', undefined as any);

test('meta.IrModel GetEffectiveConstraints returns deduplicated effective constraints', async () => {
  const meta = MetadataStorage.instance.getModelMetadata(ConstraintDiagChild as any);
  const modelIdentifier = String(meta.fullModelName || meta.modelName || meta.name || '').trim() || 'ConstraintDiagChild';

  const result = await IrModel.GetEffectiveConstraints(modelIdentifier);

  expect(String(result.model || '').length > 0).toBe(true);
  expect(Array.isArray(result.constraints)).toBe(true);
  expect(result.constraints.length).toBe(2);
  expect(result.total).toBe(2);
  expect(result.filtered).toBe(2);
  expect(result.offset).toBe(0);
  expect(result.returned).toBe(2);
  expect(result.constraints[0]?.method).toBe('baseConstraint');
  expect(result.constraints[0]?.priority).toBe(5);
  expect(result.constraints[1]?.method).toBe('childConstraint');
  expect(result.constraints[1]?.priority).toBe(20);
  expect(String(result.constraints[0]?.source || '').length > 0).toBe(true);
});

test('meta.IrModel GetEffectiveConstraints supports query filters and limit', async () => {
  const meta = MetadataStorage.instance.getModelMetadata(ConstraintDiagChild as any);
  const modelIdentifier = String(meta.fullModelName || meta.modelName || meta.name || '').trim() || 'ConstraintDiagChild';

  const result = await IrModel.GetEffectiveConstraints(modelIdentifier, {
    preview: true,
    methodPrefix: 'child',
    limit: 1,
  });

  expect(result.total).toBe(2);
  expect(result.filtered).toBe(1);
  expect(result.offset).toBe(0);
  expect(result.limit).toBe(1);
  expect(result.returned).toBe(1);
  expect(result.constraints.length).toBe(1);
  expect(result.constraints[0]?.method).toBe('childConstraint');
  expect(result.constraints[0]?.preview).toBe(true);
});

test('meta.IrModel GetEffectiveConstraints supports offset with priority-range filters', async () => {
  const meta = MetadataStorage.instance.getModelMetadata(ConstraintDiagChild as any);
  const modelIdentifier = String(meta.fullModelName || meta.modelName || meta.name || '').trim() || 'ConstraintDiagChild';

  const result = await IrModel.GetEffectiveConstraints(modelIdentifier, {
    minPriority: 5,
    maxPriority: 40,
    offset: 1,
    limit: 1,
  });

  expect(result.total).toBe(2);
  expect(result.filtered).toBe(2);
  expect(result.offset).toBe(1);
  expect(result.limit).toBe(1);
  expect(result.returned).toBe(1);
  expect(result.constraints.length).toBe(1);
  expect(result.constraints[0]?.method).toBe('childConstraint');
  expect(result.constraints[0]?.priority).toBe(20);
});

test('meta.IrModel GetEffectiveConstraints throws on unknown model', async () => {
  let error: unknown;
  try {
    await IrModel.GetEffectiveConstraints('meta.__UnknownModel__');
  } catch (err) {
    error = err;
  }

  expect(error instanceof Error).toBe(true);
  expect(String((error as Error)?.message || '').includes('model not found')).toBe(true);
});

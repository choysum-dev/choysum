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

// ---------------------------------------------------------------------------
// Instance constraint inheritance: override / add / reuse
// ---------------------------------------------------------------------------

class InheritParentModel extends BaseModel {
  Name?: string;
}

// Parent registers an instance constraint.
Object.defineProperty(InheritParentModel.prototype, 'validateName', {
  value: function validateName(this: InheritParentModel): void {
    void this.Name;
  },
  configurable: true,
  writable: true,
});
Constraint<InheritParentModel>('Name', { priority: 30 })(InheritParentModel.prototype, 'validateName', undefined as any);

// -- override: child re-decorates same method name --
class InheritOverrideModel extends InheritParentModel {
  Code?: string;
}
Object.defineProperty(InheritOverrideModel.prototype, 'validateName', {
  value: function validateName(this: InheritOverrideModel): void {
    void this.Name;
  },
  configurable: true,
  writable: true,
});
Constraint<InheritOverrideModel>('Name', { priority: 10 })(InheritOverrideModel.prototype, 'validateName', undefined as any);

test('instance constraint override: child re-decorating same method wins', () => {
  const effective = getEffectiveConstraints(InheritOverrideModel as any);
  const names = effective.filter(h => h.method === 'validateName');
  expect(names.length).toBe(1);
  expect(names[0]?.priority).toBe(10);
  expect(String(names[0]?.source || '').includes('InheritOverrideModel')).toBe(true);
  expect(names[0]?.isStatic).toBe(false);
});

// -- add: child adds new method --
class InheritAddModel extends InheritParentModel {
  Status?: string;
}
Object.defineProperty(InheritAddModel.prototype, 'validateStatus', {
  value: function validateStatus(this: InheritAddModel): void {
    void this.Status;
  },
  configurable: true,
  writable: true,
});
Constraint<InheritAddModel>('Status', { priority: 5 })(InheritAddModel.prototype, 'validateStatus', undefined as any);

test('instance constraint add: child new method coexists with inherited', () => {
  const effective = getEffectiveConstraints(InheritAddModel as any);
  const methods = effective.map(h => h.method).sort();
  expect(methods).toEqual(['validateName', 'validateStatus']);
  // validateStatus (child, priority 5) should sort before validateName (parent, priority 30).
  expect(effective[0]?.method).toBe('validateStatus');
  expect(effective[0]?.priority).toBe(5);
  expect(effective[1]?.method).toBe('validateName');
  expect(effective[1]?.priority).toBe(30);
});

// -- reuse: child inherits without re-decoration --
class InheritReuseModel extends InheritParentModel {
  Other?: string;
}

test('instance constraint reuse: child inherits parent constraint unchanged', () => {
  const effective = getEffectiveConstraints(InheritReuseModel as any);
  expect(effective.length).toBe(1);
  expect(effective[0]?.method).toBe('validateName');
  expect(effective[0]?.priority).toBe(30);
  expect(effective[0]?.isStatic).toBe(false);
});

// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel } from '@/core/service';
import { Constraint } from '@/core/service/api/constraint';
import type { LegacyConstraintMethod, InstanceConstraintMethod } from '@/core/service/api/constraint';

class ConstraintTypeParent extends BaseModel {
  Name?: string;
}

class ConstraintTypeModel extends BaseModel {
  Name?: string;
  ParentId?: ConstraintTypeParent;
}

class ConstraintTypeCases extends BaseModel {
  // -- legacy static self/ctx signature --
  @Constraint<ConstraintTypeModel>('Name')
  static validateName() {
    return undefined;
  }

  @Constraint<ConstraintTypeModel>(['Name', 'ParentId.Name'])
  static validateNestedPath() {
    return undefined;
  }

  @Constraint<ConstraintTypeModel>({ fields: ['ParentId.Name'] })
  static validateOptionsFields() {
    return undefined;
  }

  // @ts-expect-error unknown top-level field should fail strict overload
  @Constraint<ConstraintTypeModel>('UnknownField')
  static invalidTopLevelField() {
    return undefined;
  }

  // @ts-expect-error unknown nested field should fail strict overload
  @Constraint<ConstraintTypeModel>('ParentId.UnknownField')
  static invalidNestedField() {
    return undefined;
  }

  // @ts-expect-error without explicit generic, unknown field should still fail
  @Constraint('UnknownField')
  static nonGenericUnknownStillFails() {
    return undefined;
  }
}

// -- instance (non-static) constraint methods --
class ConstraintTypeInstanceCases extends BaseModel {
  Name?: string;
  Status?: string;

  @Constraint<ConstraintTypeInstanceCases>('Name')
  validateName(this: ConstraintTypeInstanceCases): void {
    void this.Name;
  }

  @Constraint<ConstraintTypeInstanceCases>(['Name', 'Status'])
  validateNameAndStatus(this: ConstraintTypeInstanceCases): void {
    void this.Name;
    void this.Status;
  }

  @Constraint<ConstraintTypeInstanceCases>({ fields: ['Status'], priority: 5 })
  validateStatus(this: ConstraintTypeInstanceCases): void {
    void this.Status;
  }

  // @ts-expect-error unknown field on instance constraint should still fail
  @Constraint<ConstraintTypeInstanceCases>('UnknownField')
  instanceInvalidField(this: ConstraintTypeInstanceCases): void {
    void this;
  }
}

// -- type-level contract verification --
const _legacy: LegacyConstraintMethod<ConstraintTypeCases> = (_self: ConstraintTypeCases, _ctx: any): void => {};
const _instance: InstanceConstraintMethod<ConstraintTypeInstanceCases> = function (this: ConstraintTypeInstanceCases): void {
  void this.Name;
};

test('constraint typing guards compile', () => {
  expect(typeof ConstraintTypeCases).toBe('function');
  expect(typeof ConstraintTypeInstanceCases).toBe('function');
  // Verify the stand-alone type aliases exist at runtime for tooling.
  expect(typeof _legacy).toBe('function');
  expect(typeof _instance).toBe('function');
});

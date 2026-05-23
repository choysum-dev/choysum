// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { BaseModel } from '@/core/service';
import { Constraint } from '@/core/service/api/constraint';

class ConstraintTypeParent extends BaseModel {
  Name?: string;
}

class ConstraintTypeModel extends BaseModel {
  Name?: string;
  ParentId?: ConstraintTypeParent;
}

class ConstraintTypeCases extends BaseModel {
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

test('constraint typing guards compile', () => {
  expect(typeof ConstraintTypeCases).toBe('function');
});

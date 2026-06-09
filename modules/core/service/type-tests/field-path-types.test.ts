// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { ClientModel } from '@/core/rpc';
import { BaseModel } from '@/core/service';
import type { FieldPath, FieldPathType } from '@/core/service/api/field';

class TestCompany extends BaseModel {
  Name: string;
}

class TestRoot extends BaseModel {
  ParentId?: TestCompany;
  Meta?: {
    Deep?: string;
  };
}

const assertStringType = (_value: string) => undefined;
assertStringType('' as FieldPathType<TestRoot, 'ParentId.Name'>);

const relationPath: FieldPath<TestRoot, ClientModel<BaseModel> | null | undefined> = 'ParentId';
const relationLeafPath: FieldPath<TestRoot, string | null | undefined> = 'ParentId.Name';

// @ts-expect-error non-BaseModel object path should be pruned
const prunedObjectPath: FieldPath<TestRoot, string | null | undefined> = 'Meta.Deep';

test('field path type guards compile', () => {
  expect(relationPath).toBe('ParentId');
  expect(relationLeafPath).toBe('ParentId.Name');
});

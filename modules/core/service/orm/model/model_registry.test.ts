// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { MetadataStorage } from '../metadata/storage';
import BaseModel from './model';
import { __setResolveModelConstructorForTest, resolveModelConstructor } from './model_registry';

test('resolveModelConstructor returns undefined for empty or unknown keys', () => {
  expect(resolveModelConstructor('')).toBe(undefined);
  expect(resolveModelConstructor('   ')).toBe(undefined);
  expect(resolveModelConstructor('__completely_unknown_model__')).toBe(undefined);
});

test('resolveModelConstructor resolves by fullModelName, modelName, name, and className', () => {
  const storage = MetadataStorage.instance as any;
  const savedModels = storage.models;

  class ResolveTestModel extends BaseModel {}
  const testCtor = ResolveTestModel as any;

  try {
    const models = new Map();
    models.set(testCtor, {
      fullModelName: 'test.ResolveModel',
      modelName: 'ResolveModel',
      name: 'TestResolveModelShort',
    });
    storage.models = models;

    expect(resolveModelConstructor('test.ResolveModel')).toBe(testCtor);
    expect(resolveModelConstructor('ResolveModel')).toBe(testCtor);
    expect(resolveModelConstructor('TestResolveModelShort')).toBe(testCtor);
    expect(resolveModelConstructor('ResolveTestModel')).toBe(testCtor); // className
  } finally {
    storage.models = savedModels;
  }
});

test('resolveModelConstructor test override is honored and cleared', () => {
  class OverrideModel extends BaseModel {}
  __setResolveModelConstructorForTest(() => OverrideModel as typeof BaseModel);
  try {
    expect(resolveModelConstructor('anything')).toBe(OverrideModel);
    expect(resolveModelConstructor('  anything  ')).toBe(OverrideModel);
    // Empty keys still short-circuit before the override (same as production).
    expect(resolveModelConstructor('')).toBe(undefined);
    expect(resolveModelConstructor('   ')).toBe(undefined);
  } finally {
    __setResolveModelConstructorForTest(undefined);
  }
  expect(resolveModelConstructor('__completely_unknown_model__')).toBe(undefined);
});

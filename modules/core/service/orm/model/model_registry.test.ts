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

test('resolveModelConstructor fuzzy scan skips null ctors and tolerates bad metadata maps', () => {
  const storage = MetadataStorage.instance as any;
  const savedModels = storage.models;
  const savedPool = (globalThis as any).pool;

  class FuzzyScanModel extends BaseModel {}
  const ctor = FuzzyScanModel as any;

  try {
    delete (globalThis as any).pool;
    const models = new Map();
    models.set(null, { modelName: 'NullShort', name: 'NullName' });
    models.set(ctor, { modelName: 'FuzzyShort', name: 'FuzzyName' });
    storage.models = models;

    expect(resolveModelConstructor('FuzzyShort')).toBe(ctor);
    expect(resolveModelConstructor('FuzzyName')).toBe(ctor);

    storage.models = undefined;
    expect(resolveModelConstructor('FuzzyShort')).toBe(undefined);

    storage.models = { notEntries: true };
    expect(resolveModelConstructor('FuzzyShort')).toBe(undefined);
  } finally {
    storage.models = savedModels;
    if (savedPool !== undefined) (globalThis as any).pool = savedPool;
    else delete (globalThis as any).pool;
  }
});

test('resolveModelConstructor prefers exact fullModelName over earlier alias collisions', () => {
  const storage = MetadataStorage.instance as any;
  const savedModels = storage.models;
  const savedPool = (globalThis as any).pool;

  class AliasCollisionModel extends BaseModel {}
  class ExactFullNameModel extends BaseModel {}
  const aliasCtor = AliasCollisionModel as any;
  const exactCtor = ExactFullNameModel as any;

  try {
    delete (globalThis as any).pool;
    const models = new Map();
    // Earlier entry aliases the later model's full name.
    models.set(aliasCtor, {
      fullModelName: 'test.AliasOwner',
      modelName: 'test.ExactTarget',
      name: 'AliasOwner',
    });
    models.set(exactCtor, {
      fullModelName: 'test.ExactTarget',
      modelName: 'ExactTarget',
      name: 'ExactTargetShort',
    });
    storage.models = models;

    expect(resolveModelConstructor('test.ExactTarget')).toBe(exactCtor);
    expect(resolveModelConstructor('ExactTarget')).toBe(exactCtor);
    expect(resolveModelConstructor('AliasOwner')).toBe(aliasCtor);
  } finally {
    storage.models = savedModels;
    if (savedPool !== undefined) (globalThis as any).pool = savedPool;
    else delete (globalThis as any).pool;
  }
});

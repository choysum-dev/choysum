// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { MetadataStorage } from '../metadata/storage';
import BaseModel from './model';
import { lookupModelCtorByFullName } from './model_ctor_lookup';

test('lookupModelCtorByFullName returns undefined for empty or nullish keys', () => {
  expect(lookupModelCtorByFullName('')).toBe(undefined);
  expect(lookupModelCtorByFullName('   ')).toBe(undefined);
  expect(lookupModelCtorByFullName(undefined as any)).toBe(undefined);
  expect(lookupModelCtorByFullName(null as any)).toBe(undefined);
});

test('lookupModelCtorByFullName resolves via metadata fullModelName and skips null ctors', () => {
  const storage = MetadataStorage.instance as any;
  const savedModels = storage.models;
  const savedPool = (globalThis as any).pool;

  class LookupFullModel extends BaseModel {}
  const ctor = LookupFullModel as any;

  try {
    delete (globalThis as any).pool;
    const models = new Map();
    models.set(null, { fullModelName: 'test.NullCtor' });
    models.set(ctor, { fullModelName: 'test.LookupFull' });
    storage.models = models;

    expect(lookupModelCtorByFullName('test.LookupFull')).toBe(ctor);
    expect(lookupModelCtorByFullName('test.Missing')).toBe(undefined);
  } finally {
    storage.models = savedModels;
    if (savedPool !== undefined) (globalThis as any).pool = savedPool;
    else delete (globalThis as any).pool;
  }
});

test('lookupModelCtorByFullName prefers global pool and tolerates missing metadata map', () => {
  const storage = MetadataStorage.instance as any;
  const savedModels = storage.models;
  const savedPool = (globalThis as any).pool;

  class PoolOnlyModel extends BaseModel {}
  const ctor = PoolOnlyModel as any;

  try {
    (globalThis as any).pool = {
      get(name: string) {
        return name === 'pool.Only' ? ctor : undefined;
      },
    };
    expect(lookupModelCtorByFullName('pool.Only')).toBe(ctor);

    storage.models = undefined;
    expect(lookupModelCtorByFullName('missing.after.pool')).toBe(undefined);

    storage.models = { notEntries: true };
    expect(lookupModelCtorByFullName('missing.bad.map')).toBe(undefined);
  } finally {
    storage.models = savedModels;
    if (savedPool !== undefined) (globalThis as any).pool = savedPool;
    else delete (globalThis as any).pool;
  }
});

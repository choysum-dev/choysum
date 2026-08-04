// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { Field, Model } from '../decorator';
import { ChoysumError } from '@/core/service/error';
import AppSettingBaseModel, {
  __invalidateAppSettingMemoForTest,
  __isUniqueConstraintErrorForTest,
} from './app_setting_base_model';
import BaseModel from './model';
import { dial, pool } from './model_pool';
import { MetadataStorage } from '../metadata/storage';

@Model('Partner', { application: 'as1cov' })
class As1CovPartner extends BaseModel {
  @Field({ type: 'varchar', size: 64 })
  Name!: string;
}

@Model('AppSetting', { application: 'as1cov', softDelete: false })
class As1CovAppSetting extends AppSettingBaseModel {}

async function expectRejects(promise: Promise<unknown>, code: string) {
  try {
    await promise;
    expect(false).toBe(true);
  } catch (err) {
    expect(err instanceof ChoysumError).toBe(true);
    expect((err as ChoysumError).code).toBe(code);
  }
}

function installStoreMocks(ctor: typeof As1CovAppSetting, store: any[]) {
  const original = {
    Search: ctor.Search,
    Create: ctor.Create,
    UpdateById: ctor.UpdateById,
    DeleteById: ctor.DeleteById,
  };
  ctor.Search = (async (condition: any) => {
    const and = condition?.And || [];
    const key = and.find((x: any) => x[0] === 'Key')?.[2];
    return store.filter(row => row.Key === key);
  }) as any;
  ctor.Create = (async (value: any) => {
    const row = { Id: `AS-${store.length + 1}`, ...value };
    store.push(row);
    return row;
  }) as any;
  ctor.UpdateById = (async (id: string, value: any) => {
    const row = store.find(r => r.Id === id);
    Object.assign(row, value);
    return row;
  }) as any;
  ctor.DeleteById = (async (id: string) => {
    const idx = store.findIndex(r => r.Id === id);
    if (idx >= 0) store.splice(idx, 1);
    return 1;
  }) as any;
  return () => {
    ctor.Search = original.Search;
    ctor.Create = original.Create;
    ctor.UpdateById = original.UpdateById;
    ctor.DeleteById = original.DeleteById;
  };
}

test('AppSetting Set no-op when value unchanged still invalidates memo', async () => {
  const store: any[] = [{ Id: 'AS-1', Key: 'same', Value: 'v' }];
  const restore = installStoreMocks(As1CovAppSetting, store);
  try {
    expect(await As1CovAppSetting.Set('same', 'v')).toBe('v');
    expect(store[0].Value).toBe('v');
  } finally {
    restore();
  }
});

test('AppSetting Get/Set coerce nullish Value and nullish key via defaults', async () => {
  const store: any[] = [{ Id: 'AS-1', Key: 'nullable', Value: null }];
  const restore = installStoreMocks(As1CovAppSetting, store);
  try {
    expect(await As1CovAppSetting.Get('nullable')).toBe('');
    expect(await As1CovAppSetting.Set('nullable', 'x')).toBe('');
    await expectRejects(As1CovAppSetting.Get(null as any), 'APP_SETTING_INVALID_KEY');
  } finally {
    restore();
  }
});

test('AppSetting Set null skips DeleteById when row has no Id', async () => {
  const store: any[] = [{ Key: 'noid', Value: 'z' }];
  const restore = installStoreMocks(As1CovAppSetting, store);
  let deleted = 0;
  As1CovAppSetting.DeleteById = (async () => {
    deleted += 1;
    return 0;
  }) as any;
  try {
    expect(await As1CovAppSetting.Set('noid', null)).toBe('z');
    expect(deleted).toBe(0);
    expect(store).toHaveLength(1);
  } finally {
    restore();
  }
});

test('AppSetting Set race recovery covers non-unique, missing row, and same-value', async () => {
  const store: any[] = [];
  const restore = installStoreMocks(As1CovAppSetting, store);
  const origCreate = As1CovAppSetting.Create;

  As1CovAppSetting.Create = (async () => {
    throw new Error('connection reset');
  }) as any;
  try {
    try {
      await As1CovAppSetting.Set('k1', 'a');
      expect(false).toBe(true);
    } catch (err) {
      expect((err as Error).message).toBe('connection reset');
    }
  } finally {
    As1CovAppSetting.Create = origCreate;
  }

  As1CovAppSetting.Create = (async () => {
    throw new Error('duplicate key value violates unique constraint');
  }) as any;
  try {
    try {
      await As1CovAppSetting.Set('k2', 'a');
      expect(false).toBe(true);
    } catch (err) {
      expect((err as Error).message).toContain('duplicate key');
    }
  } finally {
    As1CovAppSetting.Create = origCreate;
  }

  As1CovAppSetting.Create = (async (value: any) => {
    store.push({ Id: 'AS-same', Key: value.Key, Value: value.Value });
    throw 'UNIQUE constraint failed';
  }) as any;
  let updated = 0;
  const origUpdate = As1CovAppSetting.UpdateById;
  As1CovAppSetting.UpdateById = (async (...args: any[]) => {
    updated += 1;
    return origUpdate.apply(As1CovAppSetting, args as any);
  }) as any;
  try {
    expect(await As1CovAppSetting.Set('k3', 'same')).toBe('same');
    expect(updated).toBe(0);
  } finally {
    As1CovAppSetting.Create = origCreate;
    As1CovAppSetting.UpdateById = origUpdate;
  }

  store.length = 0;
  As1CovAppSetting.Create = (async (value: any) => {
    store.push({ Id: 'AS-nullish', Key: value.Key, Value: undefined });
    throw new Error('unique index "app_setting_key"');
  }) as any;
  try {
    expect(await As1CovAppSetting.Set('k4', 'filled')).toBe('');
    expect(store[0].Value).toBe('filled');
  } finally {
    As1CovAppSetting.Create = origCreate;
    restore();
  }
});

test('AppSetting memo helpers and unique matcher cover edge inputs', () => {
  __invalidateAppSettingMemoForTest('', 'k');
  __invalidateAppSettingMemoForTest('as1cov', '');
  __invalidateAppSettingMemoForTest('as1cov', 'orphan');

  expect(__isUniqueConstraintErrorForTest(new Error('UNIQUE constraint failed'))).toBe(true);
  expect(__isUniqueConstraintErrorForTest('duplicate key elsewhere')).toBe(true);
  expect(__isUniqueConstraintErrorForTest(new Error('boom'))).toBe(false);
});

test('AppSetting Get returns default when store application metadata is blank', async () => {
  const meta = MetadataStorage.instance.getModelMetadata(As1CovAppSetting as any) as any;
  const prev = meta.application;
  meta.application = undefined;
  try {
    expect(await As1CovAppSetting.Get('any', 'fallback')).toBe('fallback');
    await expectRejects(As1CovAppSetting.Set('any', '1'), 'APP_SETTING_APPLICATION_INVALID');
  } finally {
    meta.application = prev;
  }
});

test('pool falls back to MetadataStorage when global pool misses', () => {
  const previous = (globalThis as any).pool;
  (globalThis as any).pool = {
    get() {
      return undefined;
    },
  };
  try {
    expect(pool('as1cov', 'AppSetting')).toBe(As1CovAppSetting);
    expect(As1CovPartner.pool('AppSetting')).toBe(As1CovAppSetting);
  } finally {
    (globalThis as any).pool = previous;
  }
});

test('pool metadata fallback skips null ctors and missing models map', () => {
  const previousPool = (globalThis as any).pool;
  const storage = MetadataStorage.instance as any;
  const previousModels = storage.models;

  (globalThis as any).pool = { get: () => undefined };
  try {
    storage.models = undefined;
    try {
      pool('as1cov', 'AppSetting');
      expect(false).toBe(true);
    } catch (err) {
      expect((err as ChoysumError).code).toBe('POOL_MODEL_NOT_FOUND');
    }

    const map = new Map<any, any>([
      [null, { fullModelName: 'as1cov.AppSetting' }],
      [As1CovAppSetting, { fullModelName: 'as1cov.AppSetting' }],
    ]);
    storage.models = map;
    expect(pool('as1cov', 'AppSetting')).toBe(As1CovAppSetting);

    (globalThis as any).pool = {};
    expect(pool('as1cov', 'AppSetting')).toBe(As1CovAppSetting);

    (globalThis as any).pool = { get: () => ({ not: 'a function' }) };
    expect(pool('as1cov', 'AppSetting')).toBe(As1CovAppSetting);
  } finally {
    storage.models = previousModels;
    (globalThis as any).pool = previousPool;
  }
});

test('BaseModel.pool with blank application raises POOL_APPLICATION_INVALID', () => {
  class As1NoApp extends BaseModel {}
  try {
    As1NoApp.pool('AppSetting');
    expect(false).toBe(true);
  } catch (err) {
    expect((err as ChoysumError).code).toBe('POOL_APPLICATION_INVALID');
  }

  const storage = MetadataStorage.instance;
  const original = storage.getModelMetadata.bind(storage);
  (storage as any).getModelMetadata = () => undefined;
  try {
    try {
      As1NoApp.pool('AppSetting');
      expect(false).toBe(true);
    } catch (err) {
      expect((err as ChoysumError).code).toBe('POOL_APPLICATION_INVALID');
    }
  } finally {
    (storage as any).getModelMetadata = original;
  }
});

test('dial rejects whitespace-only model names', () => {
  try {
    dial('   ');
    expect(false).toBe(true);
  } catch (err) {
    expect((err as ChoysumError).code).toBe('DIAL_INVALID_MODEL');
  }
});

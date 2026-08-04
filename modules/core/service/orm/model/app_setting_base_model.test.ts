// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { Field, Model } from '../decorator';
import { ChoysumError } from '@/core/service/error';
import AppSettingBaseModel, { type AppSettingModelCtor } from './app_setting_base_model';
import BaseModel from './model';
import { dial, pool } from './model_pool';
import { createServiceByModel, registerServiceFactory } from '../../rpc/service_factory';

@Model('Partner', { application: 'as1partner' })
class As1Partner extends BaseModel {
  @Field({ type: 'varchar', size: 64 })
  Name!: string;
}

@Model('AppSetting', { application: 'as1partner', softDelete: false })
class As1AppSetting extends AppSettingBaseModel {}

@Model('User', { application: 'as1auth' })
class As1AuthUser extends BaseModel {
  @Field({ type: 'varchar', size: 64 })
  Name!: string;
}

@Model('AppSetting', { application: 'as1auth', softDelete: false })
class As1AuthAppSetting extends AppSettingBaseModel {}

async function expectRejects(promise: Promise<unknown>, code: string) {
  try {
    await promise;
    expect(false).toBe(true);
  } catch (err) {
    expect(err instanceof ChoysumError).toBe(true);
    expect((err as ChoysumError).code).toBe(code);
  }
}

async function withReq<T>(fn: () => Promise<T> | T): Promise<T> {
  const key = '$choysum';
  const hadOwn = Object.prototype.hasOwnProperty.call(globalThis as object, key);
  const previous = (globalThis as Record<string, unknown>)[key];
  (globalThis as Record<string, unknown>)[key] = {
    request: { context: { req: { id: `as1-${Date.now()}` } } },
  };
  try {
    return await fn();
  } finally {
    if (hadOwn) (globalThis as Record<string, unknown>)[key] = previous;
    else delete (globalThis as Record<string, unknown>)[key];
  }
}

function installStoreMocks(ctor: typeof As1AppSetting, store: any[]) {
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

test('AppSetting Get miss returns null or default; Set upserts and deletes', async () => {
  const store: any[] = [];
  const restore = installStoreMocks(As1AppSetting, store);
  try {
    expect(await As1AppSetting.Get('use_terms')).toBeNull();
    expect(await As1AppSetting.Get('use_terms', 'x')).toBe('x');

    expect(await As1AppSetting.Set('use_terms', '1')).toBeNull();
    expect(await As1AppSetting.Get('use_terms')).toBe('1');
    expect(store).toHaveLength(1);

    expect(await As1AppSetting.Set('use_terms', '2')).toBe('1');
    expect(await As1AppSetting.Get('use_terms')).toBe('2');
    expect(store).toHaveLength(1);

    expect(await As1AppSetting.Set('use_terms', null)).toBe('2');
    expect(store).toHaveLength(0);
    expect(await As1AppSetting.Get('use_terms', 'fallback')).toBe('fallback');

    await As1AppSetting.Set('use_terms', 'again');
    expect(store).toHaveLength(1);
    expect(store[0].Key).toBe('use_terms');

    expect(await As1AppSetting.Set('use_terms', undefined)).toBe('again');
    expect(store).toHaveLength(0);
  } finally {
    restore();
  }
});

test('AppSetting blank key raises APP_SETTING_INVALID_KEY', async () => {
  await expectRejects(As1AppSetting.Get('  '), 'APP_SETTING_INVALID_KEY');
  await expectRejects(As1AppSetting.Set('', '1'), 'APP_SETTING_INVALID_KEY');
});

@Model('AppSetting', { application: 'core', softDelete: false })
class As1CoreAppSetting extends AppSettingBaseModel {}

@Model('AppSetting', { application: 'as1soft', softDelete: true })
class As1SoftAppSetting extends AppSettingBaseModel {}

test('AppSetting Set rejects core/empty application; Get returns default', async () => {
  expect(await As1CoreAppSetting.Get('k', 'd')).toBe('d');
  await expectRejects(As1CoreAppSetting.Set('k', '1'), 'APP_SETTING_APPLICATION_INVALID');
});

test('AppSetting requires softDelete: false', async () => {
  await expectRejects(As1SoftAppSetting.Get('k'), 'APP_SETTING_SOFT_DELETE');
  await expectRejects(As1SoftAppSetting.Set('k', '1'), 'APP_SETTING_SOFT_DELETE');
});

test('AppSetting Set retries Create unique race as Update', async () => {
  const store: any[] = [];
  const restore = installStoreMocks(As1AppSetting, store);
  let createCalls = 0;
  const origCreate = As1AppSetting.Create;
  As1AppSetting.Create = (async (value: any) => {
    createCalls += 1;
    if (createCalls === 1) {
      store.push({ Id: 'AS-race', Key: value.Key, Value: 'from-other' });
      throw new Error('UNIQUE constraint failed: as1partner_app_setting.key');
    }
    return origCreate.call(As1AppSetting, value);
  }) as any;
  try {
    expect(await As1AppSetting.Set('race_key', 'mine')).toBe('from-other');
    expect(store).toHaveLength(1);
    expect(store[0].Value).toBe('mine');
  } finally {
    As1AppSetting.Create = origCreate;
    restore();
  }
});

test('AppSetting memo invalidation is exact-key (not prefix)', async () => {
  const store: any[] = [];
  const restore = installStoreMocks(As1AppSetting, store);
  let searchCount = 0;
  const origSearch = As1AppSetting.Search;
  As1AppSetting.Search = (async (condition: any, options?: any) => {
    searchCount += 1;
    return origSearch.call(As1AppSetting, condition, options);
  }) as any;
  try {
    await withReq(async () => {
      await As1AppSetting.Set('foo', '1');
      await As1AppSetting.Set('foo_bar', '2');
      searchCount = 0;
      expect(await As1AppSetting.Get('foo')).toBe('1');
      expect(await As1AppSetting.Get('foo_bar')).toBe('2');
      expect(searchCount).toBe(2);

      await As1AppSetting.Set('foo', '1b');
      searchCount = 0;
      expect(await As1AppSetting.Get('foo_bar')).toBe('2');
      expect(searchCount).toBe(0);
      expect(await As1AppSetting.Get('foo')).toBe('1b');
      expect(searchCount).toBe(1);
    });
  } finally {
    As1AppSetting.Search = origSearch;
    restore();
  }
});

test('AppSetting Get memoizes within request and Set invalidates', async () => {
  const store: any[] = [];
  const restore = installStoreMocks(As1AppSetting, store);
  let searchCount = 0;
  const origSearch = As1AppSetting.Search;
  As1AppSetting.Search = (async (condition: any, options?: any) => {
    searchCount += 1;
    return origSearch.call(As1AppSetting, condition, options);
  }) as any;

  try {
    await withReq(async () => {
      await As1AppSetting.Set('flag', 'on');
      searchCount = 0;
      expect(await As1AppSetting.Get('flag')).toBe('on');
      expect(await As1AppSetting.Get('flag')).toBe('on');
      expect(searchCount).toBe(1);

      await As1AppSetting.Set('flag', 'off');
      searchCount = 0;
      expect(await As1AppSetting.Get('flag')).toBe('off');
      expect(searchCount).toBe(1);
    });
  } finally {
    As1AppSetting.Search = origSearch;
    restore();
  }
});

test('pool resolves same-app AppSetting and rejects empty / core / missing', () => {
  const ctor = As1Partner.pool<AppSettingModelCtor>('AppSetting');
  expect(ctor).toBe(As1AppSetting);

  const viaModule = pool<AppSettingModelCtor>('as1partner', 'AppSetting');
  expect(viaModule).toBe(As1AppSetting);

  expect(As1AuthUser.pool('AppSetting')).toBe(As1AuthAppSetting);
  expect(As1Partner.pool('AppSetting')).not.toBe(As1AuthAppSetting);

  try {
    pool('as1partner', '');
    expect(false).toBe(true);
  } catch (err) {
    expect((err as ChoysumError).code).toBe('POOL_INVALID_SHORT_NAME');
  }
  try {
    pool('', 'AppSetting');
    expect(false).toBe(true);
  } catch (err) {
    expect((err as ChoysumError).code).toBe('POOL_APPLICATION_INVALID');
  }
  try {
    pool('core', 'AppSetting');
    expect(false).toBe(true);
  } catch (err) {
    expect((err as ChoysumError).code).toBe('POOL_APPLICATION_INVALID');
  }
  try {
    pool('as1partner', 'MissingModel');
    expect(false).toBe(true);
  } catch (err) {
    expect((err as ChoysumError).code).toBe('POOL_MODEL_NOT_FOUND');
  }
});

test('pool does not use global short-name scan for Ambiguous short names', () => {
  const a = pool('as1partner', 'AppSetting');
  const b = pool('as1auth', 'AppSetting');
  expect(a).toBe(As1AppSetting);
  expect(b).toBe(As1AuthAppSetting);
  expect(a).not.toBe(b);

  // resolveModelConstructor may still find *some* AppSetting via short-name scan;
  // pool must stay app-scoped and not follow that path.
  const scanned = BaseModel.resolveModelConstructor('AppSetting');
  expect(scanned === As1AppSetting || scanned === As1AuthAppSetting).toBe(true);
  expect(As1Partner.pool('AppSetting')).toBe(As1AppSetting);
});

test('dial wraps createServiceByModel; rejects empty and short names', () => {
  const modelName = `as1.DialProbe_${Date.now()}`;
  const svc = { Ref: (id: string) => `ok:${id}` };
  registerServiceFactory(modelName, () => svc);

  expect(BaseModel.dial(modelName)).toBe(svc);
  expect(dial(modelName)).toBe(createServiceByModel(modelName));
  expect((BaseModel.dial(modelName) as typeof svc).Ref('x')).toBe('ok:x');

  try {
    dial('');
    expect(false).toBe(true);
  } catch (err) {
    expect((err as ChoysumError).code).toBe('DIAL_INVALID_MODEL');
  }
  try {
    dial('SoloShort');
    expect(false).toBe(true);
  } catch (err) {
    expect((err as ChoysumError).code).toBe('DIAL_INVALID_MODEL');
  }
  for (const bad of ['app.', '.Model', 'app..Model']) {
    try {
      dial(bad);
      expect(false).toBe(true);
    } catch (err) {
      expect((err as ChoysumError).code).toBe('DIAL_INVALID_MODEL');
    }
  }

  // modelName may contain dots (`@Model('test.Foo')` → `app.test.Foo`).
  const dotted = `as1.test.DialProbe_${Date.now()}`;
  const dottedSvc = { ping: () => 'pong' };
  registerServiceFactory(dotted, () => dottedSvc);
  expect(dial(dotted)).toBe(dottedSvc);
});

// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import BaseModel from '../../orm/model/model';
import { MetadataStorage } from '../../orm/metadata/storage';
import { LRU_CACHE_SIZE } from './constants';
import { __resolveOnchangeCacheMaxEntriesForTest, __setOnchangeCacheSizeProviderForTest, OnchangeCacheManager } from './cache';

class CacheMetaModel extends BaseModel {}
class CacheOtherModel extends BaseModel {}
class CacheNameFallbackModel extends BaseModel {}
class CacheCtorFallbackModel extends BaseModel {}

test('onchange cache manager supports set/get/clear and reports stats', () => {
  OnchangeCacheManager.clear();

  const cacheKey = 'core.cache#set-get';
  const rows = new Map<string, any>([['A', { Id: 'A', Name: 'alpha' }]]);

  OnchangeCacheManager.set(cacheKey, rows);
  expect(OnchangeCacheManager.get(cacheKey)).toBe(rows);

  const stats = OnchangeCacheManager.getStats();
  expect(stats.max).toBe(LRU_CACHE_SIZE);
  expect(stats.ttl).toBe(5 * 60 * 1000);
  expect(stats.size > 0).toBe(true);

  OnchangeCacheManager.clear();
  expect(OnchangeCacheManager.get(cacheKey)).toBe(undefined);
  expect(OnchangeCacheManager.getStats().size).toBe(0);
});

test('onchange cache manager invalidate removes model-prefixed keys and keeps others', () => {
  OnchangeCacheManager.clear();

  MetadataStorage.instance.setModelMetadata(
    CacheMetaModel as any,
    {
      fullModelName: 'core.cache.meta',
    } as any
  );

  const hitKey = 'core.cache.meta#fields=Name';
  const keepKey = 'core.cache.other#fields=Name';

  OnchangeCacheManager.set(hitKey, new Map([['1', { Id: '1' }]]));
  OnchangeCacheManager.set(keepKey, new Map([['2', { Id: '2' }]]));

  const originalDebug = console.debug;
  const logs: string[] = [];
  console.debug = ((msg: unknown) => {
    logs.push(String(msg));
  }) as unknown as typeof console.debug;

  try {
    OnchangeCacheManager.invalidate(CacheMetaModel as any);
  } finally {
    console.debug = originalDebug;
  }

  expect(OnchangeCacheManager.get(hitKey)).toBe(undefined);
  expect(OnchangeCacheManager.get(keepKey)?.get('2')?.Id).toBe('2');
  expect(logs.some(line => line.includes('Invalidated 1 cache keys'))).toBe(true);
});

test('onchange cache manager invalidate with no matching keys does not emit invalidation log', () => {
  OnchangeCacheManager.clear();

  MetadataStorage.instance.setModelMetadata(
    CacheOtherModel as any,
    {
      fullModelName: 'core.cache.other',
    } as any
  );

  OnchangeCacheManager.set('core.cache.keep#x', new Map([['K', { Id: 'K' }]]));

  const originalDebug = console.debug;
  let logCount = 0;
  console.debug = (() => {
    logCount += 1;
  }) as unknown as typeof console.debug;

  try {
    OnchangeCacheManager.invalidate(CacheOtherModel as any);
  } finally {
    console.debug = originalDebug;
  }

  expect(logCount).toBe(0);
  expect(OnchangeCacheManager.get('core.cache.keep#x')?.get('K')?.Id).toBe('K');
});

test('onchange cache manager triggers eviction when entry count exceeds max', () => {
  OnchangeCacheManager.clear();

  const originalDebug = console.debug;
  const logs: string[] = [];
  console.debug = ((msg: unknown) => {
    logs.push(String(msg));
  }) as unknown as typeof console.debug;

  try {
    for (let i = 0; i < LRU_CACHE_SIZE + 3; i += 1) {
      OnchangeCacheManager.set(`evict#${i}`, new Map([['row', { Id: `${i}` }]]));
    }
  } finally {
    console.debug = originalDebug;
  }

  const stats = OnchangeCacheManager.getStats();
  expect(stats.size <= stats.max).toBe(true);
  expect(logs.some(line => line.includes('[LRU] Evicted'))).toBe(true);
});

test('onchange cache manager invalidate uses modelName when fullModelName is empty', () => {
  OnchangeCacheManager.clear();

  MetadataStorage.instance.setModelMetadata(
    CacheNameFallbackModel as any,
    {
      fullModelName: '',
      modelName: 'core.cache.by-model-name',
    } as any
  );

  const removeKey = 'core.cache.by-model-name#k1';
  const keepKey = 'core.cache.by-model-name-other#k2';
  OnchangeCacheManager.set(removeKey, new Map([['1', { Id: '1' }]]));
  OnchangeCacheManager.set(keepKey, new Map([['2', { Id: '2' }]]));

  OnchangeCacheManager.invalidate(CacheNameFallbackModel as any);

  expect(OnchangeCacheManager.get(removeKey)).toBe(undefined);
  expect(OnchangeCacheManager.get(keepKey)?.get('2')?.Id).toBe('2');
});

test('onchange cache manager invalidate falls back to constructor name when metadata names are missing', () => {
  OnchangeCacheManager.clear();

  MetadataStorage.instance.setModelMetadata(
    CacheCtorFallbackModel as any,
    {
      fullModelName: '',
      modelName: '',
    } as any
  );

  const removeKey = 'CacheCtorFallbackModel#k1';
  const keepKey = 'OtherCtor#k2';
  OnchangeCacheManager.set(removeKey, new Map([['1', { Id: '1' }]]));
  OnchangeCacheManager.set(keepKey, new Map([['2', { Id: '2' }]]));

  OnchangeCacheManager.invalidate(CacheCtorFallbackModel as any);

  expect(OnchangeCacheManager.get(removeKey)).toBe(undefined);
  expect(OnchangeCacheManager.get(keepKey)?.get('2')?.Id).toBe('2');
});

test('onchange cache manager invalidate falls back to Unknown when metadata and constructor names are empty', () => {
  OnchangeCacheManager.clear();

  const UnknownModel = class extends BaseModel {};
  Object.defineProperty(UnknownModel, 'name', { value: '' });

  const removeKey = 'Unknown#k1';
  const keepKey = 'Known#k2';
  OnchangeCacheManager.set(removeKey, new Map([['1', { Id: '1' }]]));
  OnchangeCacheManager.set(keepKey, new Map([['2', { Id: '2' }]]));

  OnchangeCacheManager.invalidate(UnknownModel as any);

  expect(OnchangeCacheManager.get(removeKey)).toBe(undefined);
  expect(OnchangeCacheManager.get(keepKey)?.get('2')?.Id).toBe('2');
});

test('onchange cache manager resolves max entries with default fallback', () => {
  expect(__resolveOnchangeCacheMaxEntriesForTest(123)).toBe(123);
  expect(__resolveOnchangeCacheMaxEntriesForTest(0)).toBe(500);
});

test('onchange cache manager bypasses get/set/invalidate when cache is disabled', () => {
  OnchangeCacheManager.clear();
  __setOnchangeCacheSizeProviderForTest(() => 0);

  const key = 'disabled#k1';
  const rows = new Map([['1', { Id: '1' }]]);

  try {
    OnchangeCacheManager.set(key, rows as any);
    expect(OnchangeCacheManager.get(key)).toBe(undefined);

    const originalDebug = console.debug;
    let called = false;
    console.debug = (() => {
      called = true;
    }) as unknown as typeof console.debug;
    try {
      OnchangeCacheManager.invalidate(CacheMetaModel as any);
    } finally {
      console.debug = originalDebug;
    }
    expect(called).toBe(false);
  } finally {
    __setOnchangeCacheSizeProviderForTest();
    OnchangeCacheManager.clear();
  }
});

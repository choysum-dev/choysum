// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { getReadonlyCtx, getUserId } from '../../runtime/context';
import { OnchangeCacheManager } from '../../runtime/onchange/cache';
import { getRepositoryReadonlyCtx, getRepositoryUserId, invalidateRepositoryRuntimeCache } from './repository_runtime_bridge';

test('repository runtime bridge proxies readonly ctx and user id helpers', () => {
  expect(getRepositoryReadonlyCtx()).toBe(getReadonlyCtx());
  expect(getRepositoryUserId()).toBe(getUserId());
});

test('repository runtime bridge invalidate cache accepts model ctor without throwing', () => {
  class DemoModel {}
  invalidateRepositoryRuntimeCache(DemoModel as any);
  expect(true).toBe(true);
});

test('repository runtime bridge skips invalidation when lru cache size override is zero', () => {
  class DemoModel {}
  const originalInvalidate = OnchangeCacheManager.invalidate;
  let callCount = 0;
  OnchangeCacheManager.invalidate = (() => {
    callCount += 1;
  }) as unknown as typeof OnchangeCacheManager.invalidate;

  try {
    invalidateRepositoryRuntimeCache(DemoModel as any, 0);
    expect(callCount).toBe(0);
  } finally {
    OnchangeCacheManager.invalidate = originalInvalidate;
  }
});

test('repository runtime bridge invalidates cache when lru cache size override is positive', () => {
  class DemoModel {}
  const originalInvalidate = OnchangeCacheManager.invalidate;
  let callCount = 0;
  OnchangeCacheManager.invalidate = (() => {
    callCount += 1;
  }) as unknown as typeof OnchangeCacheManager.invalidate;

  try {
    invalidateRepositoryRuntimeCache(DemoModel as any, 1);
    expect(callCount).toBe(1);
  } finally {
    OnchangeCacheManager.invalidate = originalInvalidate;
  }
});

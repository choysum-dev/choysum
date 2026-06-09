// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { getReadonlyCtx, getUserId } from '../../runtime/context';
import type { Context } from '../../runtime/context';
import { OnchangeCacheManager } from '../../runtime/onchange/cache';
import { LRU_CACHE_SIZE } from '../../runtime/onchange/constants';
import type BaseModel from '../model/model';

export type RepositoryRuntimeContext = Context;

export function getRepositoryReadonlyCtx(): RepositoryRuntimeContext {
  return getReadonlyCtx();
}

export function getRepositoryUserId(): string | undefined {
  return getUserId();
}

export function invalidateRepositoryRuntimeCache(ModelCtor: typeof BaseModel, lruCacheSize: number = LRU_CACHE_SIZE): void {
  if (lruCacheSize === 0) return;
  OnchangeCacheManager.invalidate(ModelCtor);
}

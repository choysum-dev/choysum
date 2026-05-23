// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { LRUCache } from 'lru-cache';
import type BaseModel from '../../orm/model/model';
import { MetadataStorage } from '../../orm/metadata/storage';
import { LRU_CACHE_SIZE } from './constants';
import type { UnknownRecord } from '../../../utils/types';

interface CachedEntry {
  data: Map<string, UnknownRecord>; // Id -> row
}

const DEFAULT_LRU_CACHE_SIZE = 500;

let cacheSizeProvider = (): number => LRU_CACHE_SIZE;

function getConfiguredCacheSize(): number {
  return cacheSizeProvider();
}

function resolveCacheMaxEntries(configuredSize: number): number {
  if (configuredSize > 0) {
    return configuredSize;
  }
  return DEFAULT_LRU_CACHE_SIZE;
}

function isCacheDisabled(): boolean {
  return getConfiguredCacheSize() === 0;
}

export function __setOnchangeCacheSizeProviderForTest(provider?: () => number): void {
  cacheSizeProvider = provider || (() => LRU_CACHE_SIZE);
}

export function __resolveOnchangeCacheMaxEntriesForTest(configuredSize: number): number {
  return resolveCacheMaxEntries(configuredSize);
}

/**
 * Onchange LRU cache manager.
 *
 * Responsibilities:
 * - Reuse prefetched results across requests at process scope.
 * - Expire entries automatically after 5 minutes.
 * - Support explicit invalidation when a model is updated.
 * - Protect memory through both entry-count and byte-size limits.
 */
export class OnchangeCacheManager {
  private static lru = new LRUCache<string, CachedEntry>({
    max: resolveCacheMaxEntries(getConfiguredCacheSize()),
    ttl: 5 * 60 * 1000, // 5-minute TTL.

    // Memory protection, optional but recommended.
    maxSize: 20 * 1024 * 1024, // 20 MB
    sizeCalculation: entry => {
      // Estimate 100 bytes per row, including V8 object overhead and field values.
      return entry.data.size * 100;
    },

    // Eviction monitoring for development.
    dispose: (entry, key, reason) => {
      console.debug(`[LRU] Evicted: ${key} (${entry.data.size} rows), reason: ${reason}`);
    },

    // Refresh TTL automatically on reads.
    updateAgeOnGet: true,
    updateAgeOnHas: false,
  });

  /**
   * Get cache data with automatic TTL expiry.
   */
  static get(cacheKey: string): Map<string, UnknownRecord> | undefined {
    if (isCacheDisabled()) return undefined;
    const entry = this.lru.get(cacheKey);
    return entry?.data;
  }

  /**
   * Store cache data.
   */
  static set(cacheKey: string, data: Map<string, UnknownRecord>): void {
    if (isCacheDisabled()) return;
    this.lru.set(cacheKey, { data });
  }

  /**
   * Explicit invalidation to be called when a model is updated.
   *
   * Strategy: remove the entire model cache at coarse granularity for simplicity.
   *
   * @param modelCtor Model constructor.
   */
  static invalidate(modelCtor: typeof BaseModel): void {
    if (isCacheDisabled()) return;

    const meta = MetadataStorage.instance.getModelMetadata(modelCtor);
    const modelKey = meta.fullModelName || meta.modelName || modelCtor.name || 'Unknown';
    const prefix = `${modelKey}#`;

    const keysToDelete: string[] = [];
    // lru-cache exposes an iterator over keys.
    for (const key of this.lru.keys()) {
      if (key.startsWith(prefix)) {
        keysToDelete.push(key);
      }
    }

    keysToDelete.forEach(k => this.lru.delete(k));

    if (keysToDelete.length > 0) {
      console.debug(`[LRU] Invalidated ${keysToDelete.length} cache keys: ${modelKey}`);
    }
  }

  /**
   * Clear all cache entries, mainly for tests and debugging.
   */
  static clear(): void {
    this.lru.clear();
  }

  /**
   * Get cache statistics for monitoring.
   */
  static getStats() {
    return {
      size: this.lru.size, // Current entry count.
      calculatedSize: this.lru.calculatedSize, // Current memory usage in bytes.
      max: this.lru.max, // Maximum entry count.
      maxSize: this.lru.maxSize, // Maximum memory usage in bytes.
      ttl: 5 * 60 * 1000, // TTL in milliseconds.
    };
  }
}

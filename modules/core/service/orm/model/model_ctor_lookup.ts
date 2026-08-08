// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { MetadataStorage } from '../metadata/storage';
import type BaseModel from './model';

/**
 * Exact-key resolve via `globalThis.pool` only (no metadata scan).
 */
export function getModelCtorFromGlobalPool(identifier: string): typeof BaseModel | undefined {
  const key = String(identifier || '').trim();
  if (!key) return undefined;

  const globalPool = (globalThis as { pool?: { get?: (name: string) => unknown } }).pool;
  if (!globalPool || typeof globalPool.get !== 'function') return undefined;

  const ctor = globalPool.get(key);
  return ctor && typeof ctor === 'function' ? (ctor as typeof BaseModel) : undefined;
}

/**
 * Resolve a model ctor by exact full model name via global pool, then metadata.
 * No short-name / className scan — that stays in `resolveModelConstructor`.
 */
export function lookupModelCtorByFullName(fullName: string): typeof BaseModel | undefined {
  const key = String(fullName || '').trim();
  if (!key) return undefined;

  const fromPool = getModelCtorFromGlobalPool(key);
  if (fromPool) return fromPool;

  const models = (MetadataStorage.instance as any)?.models as Map<typeof BaseModel, { fullModelName?: string }> | undefined;
  if (!models || typeof models.entries !== 'function') return undefined;

  for (const [ctor, meta] of models.entries()) {
    if (!ctor) continue;
    if (String(meta?.fullModelName || '').trim() === key) {
      return ctor;
    }
  }
  return undefined;
}

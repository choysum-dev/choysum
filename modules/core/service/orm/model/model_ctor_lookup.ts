// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { MetadataStorage } from '../metadata/storage';
import type BaseModel from './model';

/**
 * Resolve a model ctor by exact full model name via global pool, then metadata.
 * No short-name / className scan — that stays in `resolveModelConstructor`.
 */
export function lookupModelCtorByFullName(fullName: string): typeof BaseModel | undefined {
  const key = String(fullName || '').trim();
  if (!key) return undefined;

  const globalPool = (globalThis as { pool?: { get?: (name: string) => unknown } }).pool;
  if (globalPool && typeof globalPool.get === 'function') {
    const ctor = globalPool.get(key);
    if (ctor && typeof ctor === 'function') {
      return ctor as typeof BaseModel;
    }
  }

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

// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { MetadataStorage } from '../metadata/storage';
import { lookupModelCtorByFullName } from './model_ctor_lookup';
import type BaseModel from './model';

type ModelCtorResolver = (identifier: string) => typeof BaseModel | undefined;

let testResolver: ModelCtorResolver | undefined;

/**
 * Resolve a model constructor by identifier (engine / registry lookup).
 *
 * Accepts a full model name (e.g. `meta.MetaModule`), short model name,
 * metadata name, or constructor class name. Prefer `pool` for same-app short
 * names and `dial` for cross-app services from author code.
 */
export function resolveModelConstructor(identifier: string): typeof BaseModel | undefined {
  const key = String(identifier || '').trim();
  if (!key) return undefined;
  if (testResolver) return testResolver(key);

  const byFullName = lookupModelCtorByFullName(key);
  if (byFullName) return byFullName;

  const models = (MetadataStorage.instance as any)?.models as Map<typeof BaseModel, any> | undefined;
  if (!models || typeof models.entries !== 'function') return undefined;

  for (const [ctor, meta] of models.entries()) {
    if (!ctor) continue;
    const modelName = String(meta?.modelName || '').trim();
    const name = String(meta?.name || '').trim();
    const className = String(ctor.name || '').trim();
    if (key === modelName || key === name || key === className) {
      return ctor;
    }
  }

  return undefined;
}

/**
 * Test-only override for {@link resolveModelConstructor}.
 * Pass `undefined` to clear.
 *
 * @internal
 */
export function __setResolveModelConstructorForTest(resolver: ModelCtorResolver | undefined): void {
  testResolver = resolver;
}

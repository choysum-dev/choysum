// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { raiseDomainError } from '@/core/service/error';
import { createServiceByModel } from '../../rpc/service_factory';
import { MetadataStorage } from '../metadata/storage';
import { lookupModelCtorByFullName } from './model_ctor_lookup';
import type BaseModel from './model';

/**
 * Same-app typed resolve by short model name.
 *
 * Resolves only `${application}.${shortName}` — never scans the global short-name
 * registry (unlike `resolveModelConstructor` in `model_registry`).
 *
 * Not an alias of `globalThis.pool` (ApplicationModelPool registration table).
 */
export function pool<T = typeof BaseModel>(application: string, shortName: string): T {
  const app = String(application || '').trim();
  const short = String(shortName || '').trim();
  if (!short) {
    raiseDomainError('core', 'POOL_INVALID_SHORT_NAME', 'model short name must not be empty');
  }
  if (!app) {
    raiseDomainError('core', 'POOL_APPLICATION_INVALID', 'application must not be empty');
  }
  if (app === 'core') {
    raiseDomainError('core', 'POOL_APPLICATION_INVALID', 'pool does not resolve models for application core');
  }

  const fullName = `${app}.${short}`;
  const ctor = resolveSameAppModelConstructor(fullName);
  if (!ctor) {
    raiseDomainError('core', 'POOL_MODEL_NOT_FOUND', `model ${fullName} is not registered`);
  }
  return ctor as T;
}

/**
 * Cross-app service dial (product name for {@link createServiceByModel}).
 *
 * Returns a **service instance**, not a Model ctor. Does not imply network RPC —
 * the factory is often in-process. Requires a full `app.Model` name.
 */
export function dial<T = Record<string, (...args: unknown[]) => unknown>>(fullModelName: string): T {
  const key = String(fullModelName || '').trim();
  if (!key || !isValidFullModelName(key)) {
    raiseDomainError('core', 'DIAL_INVALID_MODEL', `dial requires a full model name app.Model, got ${key || '(empty)'}`);
  }
  return createServiceByModel(key) as T;
}

function isValidFullModelName(key: string): boolean {
  // fullModelName is `${application}.${modelName}`; modelName may itself contain dots
  // (e.g. `@Model('test.Foo')` → `app.test.Foo`). Reject empty segments only.
  const parts = key.split('.');
  return parts.length >= 2 && parts.every(part => part.length > 0);
}

/** Resolve only by exact fullModelName (pool table + metadata), no short-name fallback. */
function resolveSameAppModelConstructor(fullName: string): typeof BaseModel | undefined {
  return lookupModelCtorByFullName(fullName);
}
